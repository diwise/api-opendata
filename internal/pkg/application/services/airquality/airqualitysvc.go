package airquality

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/airquality")

const (
	DefaultBrokerTenant string = "default"
)

type AirQualityService interface {
	Refresh(ctx context.Context) (int, error)
	Shutdown(ctx context.Context)
	Start(ctx context.Context)

	Broker() string
	Tenant() string

	GetAll(ctx context.Context) []domain.AirQuality
	GetByID(ctx context.Context, id string) (*domain.AirQualityDetails, error)
}

var ErrNoSuchAirQuality error = errors.New("no such air quality")

func NewAirQualityService(ctx context.Context, ctxBrokerURL, ctxBrokerTenant string) AirQualityService {
	return &aqsvc{
		contextBrokerURL: ctxBrokerURL,
		tenant:           ctxBrokerTenant,

		airQualities:   []domain.AirQuality{},
		airQualityByID: map[string]domain.AirQualityDetails{},

		queue:       make(chan func()),
		keepRunning: &atomic.Bool{},
	}
}

type aqsvc struct {
	contextBrokerURL string
	tenant           string

	airQualities   []domain.AirQuality
	airQualityByID map[string]domain.AirQualityDetails

	queue chan func()

	keepRunning *atomic.Bool
	wg          sync.WaitGroup
}

func (svc *aqsvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *aqsvc) Tenant() string {
	return svc.tenant
}

func (svc *aqsvc) GetAll(ctx context.Context) []domain.AirQuality {
	result := make(chan []domain.AirQuality)

	svc.queue <- func() {
		result <- svc.airQualities
	}

	return <-result
}

func (svc *aqsvc) GetByID(ctx context.Context, id string) (*domain.AirQualityDetails, error) {
	result := make(chan domain.AirQualityDetails)
	err := make(chan error)

	svc.queue <- func() {
		body, ok := svc.airQualityByID[id]
		if !ok {
			err <- ErrNoSuchAirQuality
		} else {
			result <- body
		}
	}

	select {
	case r := <-result:
		return &r, nil
	case e := <-err:
		return nil, e
	}
}

func (svc *aqsvc) Refresh(ctx context.Context) (int, error) {
	logger := logging.GetFromContext(ctx)

	refreshDone := make(chan int)
	refreshFailed := make(chan error)

	svc.queue <- func() {
		count, err := svc.refresh(ctx, logger)
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

func (svc *aqsvc) Start(ctx context.Context) {
	go svc.run(ctx)
}

func (svc *aqsvc) Shutdown(ctx context.Context) {
	svc.queue <- func() {
		svc.keepRunning.Store(false)
	}

	svc.wg.Wait()
}

func (svc *aqsvc) run(ctx context.Context) {
	svc.wg.Add(1)
	defer svc.wg.Done()

	logger := logging.GetFromContext(ctx)
	logger.Info().Msg("starting up air quality service")

	// use atomic swap to avoid startup races
	alreadyStarted := svc.keepRunning.Swap(true)
	if alreadyStarted {
		logger.Error().Msg("attempt to start the air quality service multiple times")
		return
	}

	const RefreshIntervalOnFail time.Duration = 5 * time.Second
	const RefreshIntervalOnSuccess time.Duration = 5 * time.Minute

	var refreshTimer *time.Timer
	count, err := svc.refresh(ctx, logger)

	if err != nil {
		logger.Error().Err(err).Msg("failed to refresh air qualities")
		refreshTimer = time.NewTimer(RefreshIntervalOnFail)
	} else {
		logger.Info().Msgf("refreshed %d air qualities", count)
		refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
	}

	for svc.keepRunning.Load() {
		select {
		case fn := <-svc.queue:
			fn()
		case <-refreshTimer.C:
			count, err := svc.refresh(ctx, logger)
			if err != nil {
				logger.Error().Err(err).Msg("failed to refresh air qualities")
				refreshTimer = time.NewTimer(RefreshIntervalOnFail)
			} else {
				logger.Info().Msgf("refreshed %d air qualities", count)
				refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
			}
		}
	}

	logger.Info().Msg("air quality service exiting")
}

func (svc *aqsvc) refresh(ctx context.Context, log zerolog.Logger) (count int, err error) {
	ctx, span := tracer.Start(ctx, "refresh-air-quality")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, log, ctx)

	logger.Info().Msg("refreshing air quality info")

	airqualities := []domain.AirQuality{}

	_, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "AirQualityObserved", nil, func(a airqualityDTO) {

		aBytes, _ := json.Marshal(a)
		fmt.Printf("air quality dto: %s\n", aBytes)

		details := domain.AirQualityDetails{
			ID: a.ID,
		}

		if a.Location != nil {
			details.Location = *a.Location
		}

		if a.DateObserved != nil {
			details.DateObserved = *a.DateObserved
		}

		svc.airQualityByID[details.ID] = details

		aq := domain.AirQuality{
			ID:           a.ID,
			Location:     details.Location,
			DateObserved: details.DateObserved,
		}

		airqualities = append(airqualities, aq)
	})

	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve air qualities from context broker")
		return
	}

	svc.airQualities = airqualities

	return len(svc.airQualities), nil
}

type airqualityDTO struct {
	ID                        string           `json:"id"`
	Location                  *domain.Point    `json:"location"`
	DateObserved              *domain.DateTime `json:"dateObserved"`
	AtmosphericPressure       *float64         `json:"atmosphericPressure,omitempty"`
	Temperature               *float64         `json:"temperature,omitempty"`
	RelativeHumidity          *float64         `json:"relativeHumidity,omitempty"`
	ParticleCount             *float64         `json:"particleCount,omitempty"`
	PM1                       *float64         `json:"PM1,omitempty"`
	PM4                       *float64         `json:"PM4,omitempty"`
	PM10                      *float64         `json:"PM10,omitempty"`
	PM25                      *float64         `json:"PM25,omitempty"`
	TotalSuspendedParticulate *float64         `json:"totalSuspendedParticulate,omitempty"`
	CO2                       *float64         `json:"CO2,omitempty"`
	NO                        *float64         `json:"NO,omitempty"`
	NO2                       *float64         `json:"NO2,omitempty"`
	NOx                       *float64         `json:"NOx,omitempty"`
	Voltage                   *float64         `json:"voltage,omitempty"`
	WindDirection             *float64         `json:"windDirection"`
	WindSpeed                 *float64         `json:"windSpeed"`
}
