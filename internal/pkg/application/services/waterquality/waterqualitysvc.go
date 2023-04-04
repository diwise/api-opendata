package waterquality

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/waterquality")

type WaterQualityService interface {
	Start(ctx context.Context)
	Refresh(ctx context.Context) (int, error)
	Shutdown(ctx context.Context)

	Tenant() string
	Broker() string

	GetAll(ctx context.Context) []domain.WaterQuality
	GetAllNearPoint(ctx context.Context, pt Point, distance int) ([]domain.WaterQuality, error)
	GetByID(ctx context.Context, id string, from, to time.Time) (*domain.WaterQualityTemporal, error)
}

func NewWaterQualityService(ctx context.Context, url, tenant string) WaterQualityService {
	return &wqsvc{
		contextBrokerURL: url,
		tenant:           tenant,

		waterQualities:   []domain.WaterQuality{},
		waterQualityByID: map[string]domain.WaterQuality{},

		queue:       make(chan func()),
		keepRunning: &atomic.Bool{},
	}
}

var ErrWQNotFound error = errors.New("not found")

type wqsvc struct {
	contextBrokerURL string
	tenant           string

	waterQualities   []domain.WaterQuality
	waterQualityByID map[string]domain.WaterQuality

	queue chan func()

	keepRunning *atomic.Bool
	wg          sync.WaitGroup
}

func (svc *wqsvc) Start(ctx context.Context) {
	// use atomic swap to avoid startup races
	alreadyStarted := svc.keepRunning.Swap(true)
	if alreadyStarted {
		logger := logging.GetFromContext(ctx)
		logger.Error().Msg("attempt to start the water quality service multiple times")
		return
	}

	go svc.run(ctx)
}

func (svc *wqsvc) Refresh(ctx context.Context) (int, error) {

	refreshDone := make(chan int)
	refreshFailed := make(chan error)

	svc.queue <- func() {
		count, err := svc.refresh(ctx)
		if err != nil {
			refreshFailed <- err
		} else {
			refreshDone <- count
		}
	}

	select {
	case c := <-refreshDone:
		return c, nil
	case e := <-refreshFailed:
		return 0, e
	}
}

func (svc *wqsvc) Shutdown(ctx context.Context) {
	if svc.keepRunning.Load() {
		svc.queue <- func() {
			svc.keepRunning.Store(false)
		}

		svc.wg.Wait()
	}
}

func (svc *wqsvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *wqsvc) Tenant() string {
	return svc.tenant
}

func (svc *wqsvc) GetAll(ctx context.Context) []domain.WaterQuality {
	result := make(chan []domain.WaterQuality)

	svc.queue <- func() {
		result <- svc.waterQualities
	}

	return <-result
}

func (svc *wqsvc) GetAllNearPoint(ctx context.Context, pt Point, maxDistance int) ([]domain.WaterQuality, error) {

	result := make(chan []domain.WaterQuality)

	svc.queue <- func() {
		waterQualitiesWithinDistance := make([]domain.WaterQuality, 0, len(svc.waterQualities))

		for _, storedWQ := range svc.waterQualities {
			wqPoint := NewPoint(storedWQ.Location.Coordinates[1], storedWQ.Location.Coordinates[0])
			distanceBetweenPoints := distance(wqPoint, pt)

			if distanceBetweenPoints < maxDistance {
				waterQualitiesWithinDistance = append(waterQualitiesWithinDistance, storedWQ)
			}
		}

		result <- waterQualitiesWithinDistance
	}

	return <-result, nil
}

func (svc *wqsvc) GetByID(ctx context.Context, id string, from, to time.Time) (*domain.WaterQualityTemporal, error) {

	result := make(chan *domain.WaterQualityTemporal)
	failure := make(chan error)

	svc.queue <- func() {
		_, ok := svc.waterQualityByID[id]
		if !ok {
			failure <- ErrWQNotFound
			return
		}

		if from.IsZero() && to.IsZero() {
			to = time.Now().UTC()
			from = to.Add(-24 * time.Hour)
		}

		wqoBytes, err := svc.requestTemporalDataForSingleEntity(ctx, svc.contextBrokerURL, id, from, to)
		if err != nil {
			failure <- fmt.Errorf("no temporal data found for water quality with id %s", id)
			return
		}

		wqo := domain.WaterQualityTemporal{}

		err = json.Unmarshal(wqoBytes, &wqo)
		if err != nil {
			failure <- fmt.Errorf("failed to unmarshal temporal water quality with id %s", id)
			return
		}

		temps := wqo.Temperature

		sort.Slice(temps, func(i, j int) bool {
			return strings.Compare(temps[i].ObservedAt, temps[j].ObservedAt) > 0
		})

		result <- &wqo
	}

	select {
	case err := <-failure:
		return nil, err
	case r := <-result:
		return r, nil
	}
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

func distance(point1, point2 Point) int {
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
	svc.wg.Add(1)
	defer svc.wg.Done()

	logger := logging.GetFromContext(ctx)
	logger.Info().Msg("starting water quality service")

	const RefreshIntervalOnFail time.Duration = 5 * time.Second
	const RefreshIntervalOnSuccess time.Duration = 30 * time.Second

	var refreshTimer *time.Timer
	count, err := svc.refresh(ctx)

	if err != nil {
		logger.Error().Err(err).Msg("failed to refresh water qualities")
		refreshTimer = time.NewTimer(RefreshIntervalOnFail)
	} else {
		logger.Info().Msgf("refreshed %d water quality instances", count)
		refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
	}

	for svc.keepRunning.Load() {
		select {
		case fn := <-svc.queue:
			fn()
		case <-refreshTimer.C:
			count, err := svc.refresh(ctx)
			if err != nil {
				logger.Error().Err(err).Msg("failed to refresh water quality info")
				refreshTimer = time.NewTimer(RefreshIntervalOnFail)
			} else {
				logger.Info().Msgf("refreshed %d water quality entities", count)
				refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
			}
		}
	}

	logger.Info().Msg("water quality service exiting")
}

func (svc *wqsvc) refresh(ctx context.Context) (count int, err error) {

	ctx, span := tracer.Start(ctx, "refresh-water-quality")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	logger := logging.GetFromContext(ctx)
	_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	logger.Info().Msg("refreshing water quality info")

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

		svc.waterQualityByID[w.ID] = waterquality

		waterqualities = append(waterqualities, waterquality)
	})

	if err != nil {
		err = fmt.Errorf("failed to retrieve water qualities from context broker: %w", err)
		return
	}

	svc.waterQualities = waterqualities

	return len(svc.waterQualities), nil
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
