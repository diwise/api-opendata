package temperature

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
)

//go:generate moq -rm -out tempsvc_mock.go . TempService
type TempService interface {
	Query() TempServiceQuery
}

type aggrFunc func([]fiware.WeatherObserved, int, int, domain.Temperature) domain.Temperature

func average(data []fiware.WeatherObserved, from, to int, aggregate domain.Temperature) domain.Temperature {
	sum := 0.0
	for i := from; i < to; i++ {
		sum += data[i].Temperature.Value
	}

	avg := sum / float64(to-from)
	avg = float64(math.Round(avg*10) / 10)
	aggregate.Average = &avg

	return aggregate
}

func max(data []fiware.WeatherObserved, from, to int, aggregate domain.Temperature) domain.Temperature {
	max := data[from].Temperature.Value
	for i := from + 1; i < to; i++ {
		max = math.Max(max, data[i].Temperature.Value)
	}
	aggregate.Max = &max
	return aggregate
}

func min(data []fiware.WeatherObserved, from, to int, aggregate domain.Temperature) domain.Temperature {
	min := data[from].Temperature.Value
	for i := from + 1; i < to; i++ {
		min = math.Min(min, data[i].Temperature.Value)
	}
	aggregate.Min = &min
	return aggregate
}

//go:generate moq -rm -out tempsvcquery_mock.go . TempServiceQuery
type TempServiceQuery interface {
	Aggregate(period, aggregates string) TempServiceQuery
	BetweenTimes(from, to time.Time) TempServiceQuery
	Sensor(sensor string) TempServiceQuery
	Get(ctx context.Context) ([]domain.Sensor, error)
}

func NewTempService(contextBrokerURL string) TempService {
	return &ts{contextBrokerURL: contextBrokerURL}
}

type ts struct {
	contextBrokerURL    string
	contextBrokerTenant string
}

type tsq struct {
	ts
	sensor              string
	from                time.Time
	to                  time.Time
	aggregations        []aggrFunc
	aggregationDuration time.Duration
	err                 error
}

func (svc ts) Query() TempServiceQuery {
	return &tsq{ts: svc}
}

func parseAggregationPeriod(period string) (time.Duration, error) {
	supportedPeriods := map[string]time.Duration{
		"PT15M": 15 * time.Minute,
		"PT1H":  1 * time.Hour,
		"PT24H": 24 * time.Hour,
		"P7D":   7 * 24 * time.Hour,
	}

	if dur, ok := supportedPeriods[period]; ok {
		return dur, nil
	}

	return 0 * time.Millisecond, fmt.Errorf("aggregation period %s not in supported set [PT1H, PT24H, P7D]", period)
}

func (q tsq) Aggregate(period, aggregates string) TempServiceQuery {
	supportedAggregates := map[string]aggrFunc{
		"avg": average,
		"max": max,
		"min": min,
	}

	for _, aggrName := range strings.Split(aggregates, ",") {
		aggrFn, ok := supportedAggregates[aggrName]
		if ok {
			q.aggregations = append(q.aggregations, aggrFn)
		} else {
			q.err = fmt.Errorf("aggregation method %s not in supported set [avg, max, min]", aggrName)
			return q
		}
	}

	q.aggregationDuration, q.err = parseAggregationPeriod(period)

	return q
}

func (q tsq) BetweenTimes(from, to time.Time) TempServiceQuery {
	q.from = from
	q.to = to
	return q
}

func (q tsq) Sensor(sensor string) TempServiceQuery {
	q.sensor = sensor
	return q
}

func (q tsq) Get(ctx context.Context) ([]domain.Sensor, error) {
	if q.err == nil && q.sensor == "" {
		q.err = fmt.Errorf("a specific sensor must be specified")
	}

	headers := map[string][]string{
		"Accept": {"application/ld+json"},
		"Link":   {entities.LinkHeader},
	}

	// query all WeatherObserved for specified (refDevice) sensor
	cbClient := contextbroker.NewContextBrokerClient(q.contextBrokerURL, contextbroker.Tenant(q.contextBrokerTenant))
	result, err := cbClient.QueryEntities(ctx, []string{fiware.WeatherObservedTypeName}, nil, fmt.Sprintf("refDevice=%s", q.sensor), headers)
	if err != nil {
		return nil, fmt.Errorf("invalid temperature service query: %s", q.err.Error())
	}

	sensors := make([]domain.Sensor, 0)
	temperatures := make([]domain.Temperature, 0)

	for e := range result.Found {
		temporal, err := cbClient.RetrieveTemporalEvolutionOfEntity(ctx, e.ID(), nil, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid temperature service query: %s", err.Error())
		}

		props := temporal.Property("temperature")

		for i, t := range props {
			v := t.Value().(float64)
			ts, _ := time.Parse(time.RFC3339, t.ObservedAt())
			temperatures = append(temperatures, domain.Temperature{
				Id:    fmt.Sprintf("%s:temperature:%d", q.sensor, i),
				Value: &v,
				When:  &ts,
			})
		}

		sensors = append(sensors, domain.Sensor{Id: q.sensor, Temperatures: temperatures})
	}

	return sensors, nil
}
