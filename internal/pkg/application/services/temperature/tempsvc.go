package temperature

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/services")

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

type TempServiceQuery interface {
	Aggregate(period, aggregates string) TempServiceQuery
	BetweenTimes(from, to time.Time) TempServiceQuery
	Sensor(sensor string) TempServiceQuery
	Get(r *http.Request, log zerolog.Logger) ([]domain.Sensor, error)
}

func NewTempService(contextBrokerURL string) TempService {
	return &ts{contextBrokerURL: contextBrokerURL}
}

type ts struct {
	contextBrokerURL string
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

func (q tsq) Get(r *http.Request, log zerolog.Logger) ([]domain.Sensor, error) {

	if q.err == nil && q.sensor == "" {
		q.err = fmt.Errorf("a specific sensor must be specified")
	}

	if q.err != nil {
		return nil, fmt.Errorf("invalid temperature service query: %s", q.err.Error())
	}

	pageSize := uint64(1000)
	maxResultSize := uint64(50000)
	wos, err := requestData(r, log, q, 0, pageSize)
	if err != nil {
		return nil, err
	}

	if len(wos) == int(pageSize) {
		// We need to request more data page by page
		for offset := pageSize; offset < maxResultSize; offset += pageSize {
			page, err := requestData(r, log, q, offset, pageSize)
			if err != nil {
				return nil, err
			}

			wos = append(wos, page...)

			if len(page) < int(pageSize) {
				break
			}
		}
	}

	temps := []domain.Temperature{}

	if len(wos) > 0 {

		if len(q.aggregations) == 0 {
			for _, wo := range wos {
				dateObserved, _ := time.Parse(time.RFC3339, wo.DateObserved.Value.Value)

				t := domain.Temperature{
					Id:    wo.RefDevice.Object,
					Value: &wo.Temperature.Value,
					When:  &dateObserved,
				}
				temps = append(temps, t)
			}
		} else {
			dateOfFirstObservation, _ := time.Parse(time.RFC3339, wos[0].DateObserved.Value.Value)
			periodStart := dateOfFirstObservation
			if !q.from.IsZero() {
				periodStart = q.from
			}

			periodEnd := periodStart.Add(q.aggregationDuration).Add(-1 * time.Millisecond)
			for periodEnd.Before(dateOfFirstObservation) {
				periodStart = periodStart.Add(q.aggregationDuration)
				periodEnd = periodEnd.Add(q.aggregationDuration)
			}

			aggregationStartIndex := 0

			for idx := range wos {
				dateObserved, _ := time.Parse(time.RFC3339, wos[idx].DateObserved.Value.Value)
				if dateObserved.After(periodEnd) {
					ps := periodStart
					pe := periodEnd

					aggr := domain.Temperature{
						Id:   wos[0].RefDevice.Object,
						From: &ps,
						To:   &pe,
					}
					for _, aggrF := range q.aggregations {
						aggr = aggrF(wos, aggregationStartIndex, idx, aggr)
					}

					temps = append(temps, aggr)

					periodStart = periodStart.Add(q.aggregationDuration)
					periodEnd = periodEnd.Add(q.aggregationDuration)

					aggregationStartIndex = idx
				}
			}

			aggr := domain.Temperature{
				Id:   wos[0].RefDevice.Object,
				From: &periodStart,
				To:   &periodEnd,
			}
			for _, aggrF := range q.aggregations {
				aggr = aggrF(wos, aggregationStartIndex, len(wos), aggr)
			}

			temps = append(temps, aggr)
		}
	}

	sensors := []domain.Sensor{{Id: q.sensor, Temperatures: temps}}

	return sensors, nil
}

func requestData(r *http.Request, log zerolog.Logger, q tsq, offset, limit uint64) ([]fiware.WeatherObserved, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	ctx, span := tracer.Start(r.Context(), "incoming-message")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	traceID := span.SpanContext().TraceID()
	if traceID.IsValid() {
		log = log.With().Str("traceID", traceID.String()).Logger()
	}

	ctx = logging.NewContextWithLogger(ctx, log)

	url := fmt.Sprintf(
		"%s/ngsi-ld/v1/entities?type=WeatherObserved&attrs=temperature&q=refDevice==\"%s\"",
		q.ts.contextBrokerURL,
		q.sensor,
	)

	if !q.from.IsZero() && !q.to.IsZero() {
		timeAt := q.from.Format(time.RFC3339)
		endTimeAt := q.to.Format(time.RFC3339)
		url = url + fmt.Sprintf("&timerel=between&timeAt=%s&endTimeAt=%s", timeAt, endTimeAt)
	}

	if limit > 0 {
		url = url + fmt.Sprintf("&limit=%d", limit)
	}

	if offset > 0 {
		url = url + fmt.Sprintf("&offset=%d", offset)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to create http request")
		return nil, err
	}

	response, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get water quality observed from context broker")
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed, status code not ok: %d", response.StatusCode)
	}

	wos := []fiware.WeatherObserved{}
	b, _ := io.ReadAll(response.Body)

	err = json.Unmarshal(b, &wos)
	if err != nil {
		return nil, err
	}

	return wos, nil
}
