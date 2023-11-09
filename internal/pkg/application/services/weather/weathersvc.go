package weather

import (
	"context"
	"fmt"
	"math"
	"net/url"
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
	DateObserved *string
	Temperature  *float64
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

		weather = append(weather, toDTO(entity))
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

	dto := toDTO(entity)
	props := temporal.Property("temperature")

	for _, t := range props {
		v, ok := t.Value().(float64)
		if !ok {
			continue
		}
		ts := t.ObservedAt()
		dto.Temperatures = append(dto.Temperatures, TemperatureDTO{
			DateObserved: &ts,
			Temperature:  &v,
		})
	}

	return toWeather(dto), nil
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

func toDTO(e types.Entity) WeatherDTO {
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
			dateObserved, _ := time.Parse(time.RFC3339, *t.DateObserved)
			temperatures = append(temperatures, domain.Temperature{
				Value: t.Temperature,
				When:  &dateObserved,
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
