package waterquality

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/waterquality")

type WaterQualityService interface {
	Start()
	Shutdown()

	Tenant() string
	Broker() string

	BetweenTimes(from, to time.Time)
	Distance(distance int)
	Location(latitude, longitude float64)

	GetAll() []domain.WaterQuality
	GetAllNearPoint(pt Point, distance int) (*[]domain.WaterQuality, error)
	GetByID(id string) (*domain.WaterQualityTemporal, error)
}

func NewWaterQualityService(ctx context.Context, log zerolog.Logger, url, tenant string) WaterQualityService {
	return &wqsvc{
		contextBrokerURL: url,
		tenant:           tenant,

		waterQualities:   []domain.WaterQuality{},
		waterQualityByID: map[string]domain.WaterQuality{},
		keepRunning:      true,

		ctx: ctx,
		log: log,
	}
}

type wqsvc struct {
	contextBrokerURL string
	tenant           string

	from      time.Time
	to        time.Time
	latitude  float64
	longitude float64
	distance  int

	wqoMutex         sync.Mutex
	waterQualities   []domain.WaterQuality
	waterQualityByID map[string]domain.WaterQuality

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *wqsvc) BetweenTimes(from, to time.Time) {
	svc.from = from
	svc.to = to
}

func (svc *wqsvc) Distance(distance int) {
	svc.distance = distance
}

func (svc *wqsvc) Location(latitude, longitude float64) {
	svc.latitude = latitude
	svc.longitude = longitude
}

func (svc *wqsvc) Start() {
	svc.log.Info().Msg("starting water quality service")
	go svc.run()
}

func (svc *wqsvc) Shutdown() {
	svc.log.Info().Msg("shutting down water quality service")
	svc.keepRunning = false
}

func (svc *wqsvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *wqsvc) Tenant() string {
	return svc.tenant
}

func (svc *wqsvc) GetAll() []domain.WaterQuality {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	return svc.waterQualities
}

func (svc *wqsvc) GetAllNearPoint(pt Point, maxDistance int) (*[]domain.WaterQuality, error) {
	waterQualitiesWithinDistance := []domain.WaterQuality{}

	for _, storedWQ := range svc.waterQualities {
		wqPoint := NewPoint(storedWQ.Location.Coordinates[1], storedWQ.Location.Coordinates[0])
		distanceBetweenPoints := Distance(wqPoint, pt)

		if distanceBetweenPoints < maxDistance {
			waterQualitiesWithinDistance = append(waterQualitiesWithinDistance, storedWQ)
		}
	}
	if len(waterQualitiesWithinDistance) == 0 {
		return &[]domain.WaterQuality{}, fmt.Errorf("no stored water qualities exist within %d meters of point %f,%f", maxDistance, pt.Longitude, pt.Latitude)
	}

	return &waterQualitiesWithinDistance, nil
}

func (svc *wqsvc) GetByID(id string) (*domain.WaterQualityTemporal, error) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	_, ok := svc.waterQualityByID[id]
	if !ok {
		return nil, fmt.Errorf("no water quality found with id %s", id)
	}

	wqo := domain.WaterQualityTemporal{}

	wqoBytes, err := svc.requestTemporalDataForSingleEntity(svc.ctx, svc.log, svc.contextBrokerURL, id)
	if err != nil {
		return nil, fmt.Errorf("no water quality found with id %s: %s", id, err.Error())
	}

	err = json.Unmarshal(wqoBytes, &wqo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal water quality with id %s: %s", id, err.Error())
	}

	return &wqo, nil
}

type Point struct {
	Latitude  float64
	Longitude float64
}

func NewPoint(lat, lon float64) Point {
	return Point{
		Latitude:  lat,
		Longitude: lon,
	}
}

func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

func Distance(point1, point2 Point) int {
	earthRadiusKm := 6371

	lat1 := degreesToRadians(point1.Latitude)
	lon1 := degreesToRadians(point1.Longitude)
	lat2 := degreesToRadians(point2.Latitude)
	lon2 := degreesToRadians(point2.Longitude)

	diffLat := lat2 - lat1
	diffLon := lon2 - lon1

	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*
		math.Pow(math.Sin(diffLon/2), 2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distanceInKm := c * float64(earthRadiusKm)
	distanceInM := distanceInKm * 1000

	return int(distanceInM)
}

func (svc *wqsvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing water quality info")
			err := svc.refresh()
			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh water quality info")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				svc.log.Info().Msgf("refreshed water qualities")
				// Refresh every 5 minutes of success
				nextRefreshTime = time.Now().Add(30 * time.Second)
			}
		}

		time.Sleep(1 * time.Second)
	}
	svc.log.Info().Msg("water quality service exiting")
}

func (svc *wqsvc) refresh() (err error) {

	ctx, span := tracer.Start(svc.ctx, "refresh-water-quality")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, _ = o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	waterqualities := []domain.WaterQuality{}

	_, err = contextbroker.QueryEntities(ctx, svc.Broker(), svc.Tenant(), "WaterQualityObserved", nil, func(w WaterQualityDTO) {
		waterquality := domain.WaterQuality{
			ID:           w.ID,
			Temperature:  w.Temperature,
			DateObserved: w.DateObserved.Value,
			Location:     *domain.NewPoint(w.Location.Value.Coordinates[1], w.Location.Value.Coordinates[0]),
		}

		if w.Source != "" {
			waterquality.Source = &w.Source
		}

		svc.storeWaterQuality(w.ID, waterquality)

		waterqualities = append(waterqualities, waterquality)
	})
	if err != nil {
		err = fmt.Errorf("failed to retrieve water qualities from context broker: %w", err)
		return
	}

	svc.storeWaterQualityList(waterqualities)

	return nil
}

func (svc *wqsvc) storeWaterQuality(id string, body domain.WaterQuality) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	svc.waterQualityByID[id] = body
}

func (svc *wqsvc) storeWaterQualityList(wqs []domain.WaterQuality) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	svc.waterQualities = wqs
}

func (q *wqsvc) requestTemporalDataForSingleEntity(ctx context.Context, log zerolog.Logger, ctxBrokerURL, id string) ([]byte, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	url := fmt.Sprintf(
		"%s/ngsi-ld/v1/temporal/entities/%s?attrs=temperature",
		ctxBrokerURL, id,
	)

	if !q.from.IsZero() && !q.to.IsZero() {
		url = fmt.Sprintf("%s&timerel=between&timeAt=%s&endTimeAt=%s", url, q.from.Format(time.RFC3339), q.to.Format(time.RFC3339))
	} else {
		q.from = time.Now().UTC().Add(-24 * time.Hour)
		q.to = time.Now().UTC()
		url = fmt.Sprintf("%s&timerel=between&timeAt=%s&endTimeAt=%s", url, q.from.Format(time.RFC3339), q.to.Format(time.RFC3339))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		q.log.Error().Err(err).Msg("failed to create http request")
		return nil, err
	}

	response, err := httpClient.Do(req)
	if err != nil {
		q.log.Error().Err(err).Msg("request failed")
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		q.log.Error().Err(err).Msg("request failed, status code not ok")
		return nil, err
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		q.log.Error().Err(err).Msg("failed to read response body")
		return nil, err
	}

	return b, nil
}

type WaterQualityTemporalDTO struct {
	ID       string `json:"id"`
	Location struct {
		Type  string       `json:"type"`
		Value domain.Point `json:"value"`
	} `json:"location"`
	Temperature  []domain.Value `json:"temperature"`
	Source       string         `json:"source,omitempty"`
	DateObserved struct {
		Type            string `json:"type"`
		domain.DateTime `json:"value"`
	} `json:"dateObserved,omitempty"`
}

type WaterQualityDTO struct {
	ID       string `json:"id"`
	Location struct {
		Type  string       `json:"type"`
		Value domain.Point `json:"value"`
	} `json:"location"`
	Temperature  float64 `json:"temperature"`
	Source       string  `json:"source,omitempty"`
	DateObserved struct {
		Type            string `json:"type"`
		domain.DateTime `json:"value"`
	} `json:"dateObserved,omitempty"`
}
