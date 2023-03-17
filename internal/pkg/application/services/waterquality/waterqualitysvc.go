package waterquality

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/waterquality")

type WaterQualityService interface {
	Start(ctx context.Context)
	Shutdown(ctx context.Context)

	Tenant() string
	Broker() string

	GetAll(ctx context.Context) []domain.WaterQuality
	GetAllNearPoint(ctx context.Context, pt Point, distance int) ([]domain.WaterQuality, error)
	GetByID(ctx context.Context, id string, from, to time.Time) (*domain.WaterQualityTemporal, error)
}

func NewWaterQualityService(ctx context.Context, log zerolog.Logger, url, tenant string) WaterQualityService {
	return &wqsvc{
		contextBrokerURL: url,
		tenant:           tenant,

		waterQualities:   []domain.WaterQuality{},
		waterQualityByID: map[string]domain.WaterQuality{},
		keepRunning:      true,
	}
}

type wqsvc struct {
	contextBrokerURL string
	tenant           string

	wqoMutex         sync.Mutex
	waterQualities   []domain.WaterQuality
	waterQualityByID map[string]domain.WaterQuality

	keepRunning bool
}

func (svc *wqsvc) Start(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info().Msg("starting water quality service")
	go svc.run(ctx)
}

func (svc *wqsvc) Shutdown(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info().Msg("shutting down water quality service")
	svc.keepRunning = false
}

func (svc *wqsvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *wqsvc) Tenant() string {
	return svc.tenant
}

func (svc *wqsvc) GetAll(ctx context.Context) []domain.WaterQuality {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	return svc.waterQualities
}

func (svc *wqsvc) GetAllNearPoint(ctx context.Context, pt Point, maxDistance int) ([]domain.WaterQuality, error) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	waterQualitiesWithinDistance := []domain.WaterQuality{}

	for _, storedWQ := range svc.waterQualities {
		wqPoint := NewPoint(storedWQ.Location.Coordinates[1], storedWQ.Location.Coordinates[0])
		distanceBetweenPoints := Distance(wqPoint, pt)

		if distanceBetweenPoints < maxDistance {
			waterQualitiesWithinDistance = append(waterQualitiesWithinDistance, storedWQ)
		}
	}

	if len(waterQualitiesWithinDistance) == 0 {
		return []domain.WaterQuality{}, fmt.Errorf("no stored water qualities exist within %d meters of point %f,%f", maxDistance, pt.Longitude, pt.Latitude)
	}

	return waterQualitiesWithinDistance, nil
}

func (svc *wqsvc) GetByID(ctx context.Context, id string, from, to time.Time) (*domain.WaterQualityTemporal, error) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	_, ok := svc.waterQualityByID[id]
	if !ok {
		return nil, fmt.Errorf("no stored water quality found with id %s", id)
	}

	wqo := domain.WaterQualityTemporal{}

	if from.IsZero() && to.IsZero() {
		to = time.Now().UTC()
		from = to.Add(-24 * time.Hour)
	}

	wqoBytes, err := svc.requestTemporalDataForSingleEntity(ctx, svc.contextBrokerURL, id, from, to)
	if err != nil {
		return nil, fmt.Errorf("no temporal data found for water quality with id %s", id)
	}

	err = json.Unmarshal(wqoBytes, &wqo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal temporal water quality with id %s", id)
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

func (svc *wqsvc) run(ctx context.Context) {
	nextRefreshTime := time.Now()

	logger := logging.GetFromContext(ctx)

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			logger.Info().Msg("refreshing water quality info")

			err := svc.refresh(context.Background())
			if err != nil {
				logger.Error().Err(err).Msg("failed to refresh water quality info")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				logger.Info().Msgf("refreshed water qualities")
				// Refresh every 5 minutes of success
				nextRefreshTime = time.Now().Add(30 * time.Second)
			}
		}

		time.Sleep(1 * time.Second)
	}

	logger.Info().Msg("water quality service exiting")
}

func (svc *wqsvc) refresh(ctx context.Context) (err error) {

	ctx, span := tracer.Start(ctx, "refresh-water-quality")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	logger := logging.GetFromContext(ctx)
	_, ctx, _ = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	waterqualities := []domain.WaterQuality{}

	_, err = contextbroker.QueryEntities(ctx, svc.Broker(), svc.Tenant(), "WaterQualityObserved", nil, func(w WaterQualityDTO) {

		waterquality := domain.WaterQuality{
			ID:           w.ID,
			Temperature:  w.Temperature,
			DateObserved: w.DateObserved.Value,
		}

		if w.Location != nil {
			waterquality.Location = domain.NewPoint(w.Location.Coordinates[1], w.Location.Coordinates[0])
		}

		if w.Source != nil {
			waterquality.Source = w.Source
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

func (q *wqsvc) requestTemporalDataForSingleEntity(ctx context.Context, ctxBrokerURL, id string, from, to time.Time) ([]byte, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	params := url.Values{}

	if from.IsZero() && to.IsZero() {
		from = time.Now().UTC().Add(-24 * time.Hour)
		to = time.Now().UTC()
	}

	params.Add("timerel", "between")
	params.Add("timeAt", from.Format(time.RFC3339))
	params.Add("endTimeAt", to.Format(time.RFC3339))

	requestURL := fmt.Sprintf(
		"%s/ngsi-ld/v1/temporal/entities/%s?%s",
		ctxBrokerURL, id, params.Encode(),
	)

	log.Debug().Msgf("request url for retrieving single entity: %s", requestURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		logger := logging.GetFromContext(ctx)
		logger.Error().Err(err).Msg("failed to create http request")
		return nil, err
	}

	req.Header.Add("Accept", "application/ld+json")
	req.Header.Add("Link", entities.LinkHeader)

	response, err := httpClient.Do(req)
	if err != nil {
		logger := logging.GetFromContext(ctx)
		logger.Error().Err(err).Msg("request failed")
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		logger := logging.GetFromContext(ctx)
		logger.Error().Err(err).Msg("request failed, status code not ok")
		return nil, err
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		logger := logging.GetFromContext(ctx)
		logger.Error().Err(err).Msg("failed to read response body")
		return nil, err
	}

	return b, nil
}

type WaterQualityTemporalDTO struct {
	ID           string          `json:"id"`
	Temperature  []domain.Value  `json:"temperature"`
	Source       *string         `json:"source,omitempty"`
	DateObserved domain.DateTime `json:"dateObserved,omitempty"`
}

type WaterQualityDTO struct {
	ID           string          `json:"id"`
	Location     *domain.Point   `json:"location,omitempty"`
	Temperature  float64         `json:"temperature"`
	Source       *string         `json:"source,omitempty"`
	DateObserved domain.DateTime `json:"dateObserved,omitempty"`
}
