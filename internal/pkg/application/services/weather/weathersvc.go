package weather

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"slices"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/context-broker/pkg/ngsild/geojson"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
)

//go:generate moq -rm -out weathersvc_mock.go . WeatherService
type WeatherService interface {
	Query() WeatherServiceQuery
}

//go:generate moq -rm -out weathersvcquery_mock.go . WeatherServiceQuery
type WeatherServiceQuery interface {
	BetweenTimes(from, to time.Time) WeatherServiceQuery
	NearPoint(distance int64, lat, lon float64) WeatherServiceQuery
	ID(id string) WeatherServiceQuery
	Aggr(res string) WeatherServiceQuery
	Get(ctx context.Context) ([]domain.Weather, error)
	GetByID(ctx context.Context) (domain.Weather, error)
}

func NewWeatherService(ctx context.Context, contextBrokerURL string, contextBrokerTenant string) WeatherService {
	return &ws{
		contextBrokerURL:    contextBrokerURL,
		contextBrokerTenant: contextBrokerTenant,
	}
}

type ws struct {
	contextBrokerURL    string
	contextBrokerTenant string
}

type wsq struct {
	ws
	id       string
	lat      float64
	lon      float64
	distance int64
	from     time.Time
	to       time.Time
	err      error
	aggr     string
}

func (svc *ws) Query() WeatherServiceQuery {
	return &wsq{ws: *svc}
}

func (q wsq) BetweenTimes(from, to time.Time) WeatherServiceQuery {
	q.from = from
	q.to = to
	return q
}

func (q wsq) NearPoint(dist int64, lat, lon float64) WeatherServiceQuery {
	q.lat = lat
	q.lon = lon
	q.distance = dist
	return q
}

func (q wsq) ID(id string) WeatherServiceQuery {
	q.id = id
	return q
}

func (q wsq) Aggr(aggr string) WeatherServiceQuery {
	if aggr != "hour" && aggr != "day" && aggr != "month" && aggr != "year" {
		q.aggr = ""
		return q
	}

	q.aggr = aggr
	return q
}

type WeatherDTO struct {
	ID           string
	DateObserved *string
	Temperature  *float64
	Location     *struct {
		Lat float64
		Lon float64
	}
	Temperatures []TemperatureDTO
}

type TemperatureDTO struct {
	DateObserved *time.Time
	Temperature  *float64
	Max          *float64
	Min          *float64
	From         *time.Time
	To           *time.Time
}

func NewDTO(id string) WeatherDTO {
	return WeatherDTO{
		ID:           id,
		Temperatures: make([]TemperatureDTO, 0),
	}
}

func (q wsq) Get(ctx context.Context) ([]domain.Weather, error) {
	headers := map[string][]string{
		"Accept": {"application/ld+json"},
		"Link":   {entities.LinkHeader},
	}

	cbClient := contextbroker.NewContextBrokerClient(q.contextBrokerURL, contextbroker.Tenant(q.contextBrokerTenant))

	params := url.Values{}
	params.Add("type", fiware.WeatherObservedTypeName)
	params.Add("geoproperty", "location")
	params.Add("georel", fmt.Sprintf("near;maxDistance==%d", q.distance))
	params.Add("geometry", "Point")
	params.Add("coordinates", fmt.Sprintf("[%f,%f]", q.lon, q.lat))

	reqUrl := fmt.Sprintf("/ngsi-ld/v1/entities?%s", params.Encode())

	wos, err := cbClient.QueryEntities(ctx, nil, nil, reqUrl, headers)
	if err != nil {
		return nil, fmt.Errorf("invalid temperature service query: %s", q.err.Error())
	}

	weather := make([]WeatherDTO, 0)

	for {
		entity := <-wos.Found
		if entity == nil {
			break
		}

		weather = append(weather, weatherObservedToWeatherDto(entity))
	}

	return toWeatherSlice(weather), nil
}

func (q wsq) GetByID(ctx context.Context) (domain.Weather, error) {
	if q.id == "" {
		return domain.Weather{}, fmt.Errorf("no id specified")
	}

	headers := map[string][]string{
		"Accept": {"application/ld+json"},
		"Link":   {entities.LinkHeader},
	}

	cbClient := contextbroker.NewContextBrokerClient(q.contextBrokerURL, contextbroker.Tenant(q.contextBrokerTenant))

	entity, err := cbClient.RetrieveEntity(ctx, q.id, headers)
	if err != nil {
		return domain.Weather{}, err
	}

	temporal, err := cbClient.RetrieveTemporalEvolutionOfEntity(ctx, entity.ID(), headers, contextbroker.Between(q.from, q.to))
	if err != nil {
		return domain.Weather{}, fmt.Errorf("invalid temperature service query: %s", err.Error())
	}

	dto := weatherObservedToWeatherDto(entity)
	dto.Temperatures = temporalPropertiesToTemperatureDto(temporal.Property("temperature"))

	if q.aggr != "" && q.aggr == "hour" || q.aggr == "day" || q.aggr == "month" || q.aggr == "year" {
		dto.Temperatures = groupByTime(dto.Temperatures, q.aggr)
	}

	return toWeather(dto), nil
}

func groupByTime(tempDto []TemperatureDTO, res string) []TemperatureDTO {
	grouped := make(map[string][]TemperatureDTO)

	for _, t := range tempDto {
		dateObserved := t.DateObserved.Format(time.RFC3339)
		switch res {
		case "hour":
			grouped[dateObserved[0:13]] = append(grouped[dateObserved[0:13]], t)
		case "day":
			grouped[dateObserved[0:10]] = append(grouped[dateObserved[0:10]], t)
		case "month":
			grouped[dateObserved[0:7]] = append(grouped[dateObserved[0:7]], t)
		case "year":
			grouped[dateObserved[0:4]] = append(grouped[dateObserved[0:4]], t)
		}
	}

	aggregated := make([]TemperatureDTO, 0)

	for _, t := range grouped {
		slices.SortFunc[[]TemperatureDTO](t, func(a, b TemperatureDTO) int {
			if a.DateObserved.Before(*b.DateObserved) {
				return -1
			}
			if a.DateObserved.After(*b.DateObserved) {
				return 1
			}
			return 0
		})
		aggregated = append(aggregated, aggregate(t))
	}

	return aggregated
}

func aggregate(temperatures []TemperatureDTO) TemperatureDTO {
	var min, max *float64
	var avg, total float64

	rnd := func(val float64) float64 {
		ratio := math.Pow(10, float64(2))
		return math.Round(val*ratio) / ratio
	}

	for _, t := range temperatures {
		if min == nil && t.Temperature != nil {
			min = t.Temperature
		} else if t.Temperature != nil && *t.Temperature < *min {
			min = t.Temperature
		}

		if max == nil && t.Temperature != nil {
			max = t.Temperature
		} else if t.Temperature != nil && *t.Temperature > *max {
			max = t.Temperature
		}

		if t.Temperature != nil {
			total = total + *t.Temperature
		}
	}

	avg = rnd(total / float64(len(temperatures)))
	return TemperatureDTO{
		DateObserved: temperatures[0].DateObserved,
		Temperature:  &avg,
		Max:          max,
		Min:          min,
		From:         temperatures[0].DateObserved,
		To:           temperatures[len(temperatures)-1].DateObserved,
	}
}

func calc(w *domain.Weather) domain.Weather {
	if w.Temperature.Values == nil || len(*w.Temperature.Values) == 0 {
		return *w
	}

	var min, max *float64
	var avg, total float64

	rnd := func(val float64) float64 {
		ratio := math.Pow(10, float64(2))
		return math.Round(val*ratio) / ratio
	}

	for _, t := range *w.Temperature.Values {
		if min == nil && t.Value != nil {
			min = t.Value
		} else if t.Value != nil && *t.Value < *min {
			min = t.Value
		}

		if max == nil && t.Value != nil {
			max = t.Value
		} else if t.Value != nil && *t.Value > *max {
			max = t.Value
		}

		if t.Value != nil {
			total = total + *t.Value
		}
	}

	temps := *w.Temperature.Values

	avg = rnd(total / float64(len(temps)))
	w.Temperature.Average = &avg
	w.Temperature.Max = max
	w.Temperature.Min = min
	w.Temperature.From = temps[0].When
	w.Temperature.To = temps[len(temps)-1].When

	return *w
}

func temporalPropertiesToTemperatureDto(props []types.TemporalProperty) []TemperatureDTO {
	temperatures := make([]TemperatureDTO, 0)

	for _, p := range props {
		v, ok := p.Value().(float64)
		if !ok {
			continue
		}

		tss := p.ObservedAt()
		ts, err := time.Parse(time.RFC3339, tss)
		if err != nil {
			continue
		}
		temperatures = append(temperatures, TemperatureDTO{
			DateObserved: &ts,
			Temperature:  &v,
		})
	}

	return temperatures
}

func weatherObservedToWeatherDto(e types.Entity) WeatherDTO {
	w := NewDTO(e.ID())

	e.ForEachAttribute(func(_, attributeName string, contents any) {
		switch attributeName {
		case "dateObserved":
			p := contents.(*properties.DateTimeProperty)
			w.DateObserved = &p.Val.Value
		case "temperature":
			p := contents.(*properties.NumberProperty)
			w.Temperature = &p.Val
		case "location":
			p := contents.(*geojson.GeoJSONProperty)
			point := p.GetAsPoint()
			w.Location = &struct {
				Lat float64
				Lon float64
			}{
				Lat: point.Latitude(),
				Lon: point.Longitude(),
			}
		}
	})

	return w
}

func toWeather(d WeatherDTO) domain.Weather {
	dateObserved, _ := time.Parse(time.RFC3339, *d.DateObserved)

	w := domain.Weather{
		ID: d.ID,
		Temperature: domain.Temperature{
			Value: d.Temperature,
			When:  &dateObserved,
		},
		DateObserved: dateObserved,
		Location:     domain.NewPoint(d.Location.Lat, d.Location.Lon),
	}

	if len(d.Temperatures) > 0 {
		temperatures := make([]domain.Temperature, 0)

		for _, t := range d.Temperatures {
			temperatures = append(temperatures, domain.Temperature{
				Value: t.Temperature,
				When:  t.DateObserved,
			})
		}

		w.Temperature.Value = nil
		w.Temperature.When = nil
		w.Temperature.Values = &temperatures
	}

	return calc(&w)
}

func toWeatherSlice(dto []WeatherDTO) []domain.Weather {
	weather := make([]domain.Weather, 0)

	for _, d := range dto {
		weather = append(weather, toWeather(d))
	}

	return weather
}
