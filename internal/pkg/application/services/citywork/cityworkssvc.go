package citywork

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/cityworks")

type CityworksService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start()
	Shutdown()
}

func NewCityworksService(ctx context.Context, logger zerolog.Logger, contextBrokerUrl, tenant string) CityworksService {
	svc := &cityworksSvc{
		ctx: ctx,

		cityworks:        []byte("[]"),
		cityworksDetails: map[string][]byte{},
		contextBrokerURL: contextBrokerUrl,
		tenant:           tenant,
		log:              logger,

		keepRunning: true,
	}

	return svc
}

type cityworksSvc struct {
	contextBrokerURL string
	tenant           string

	cityworksMutex   sync.Mutex
	cityworks        []byte
	cityworksDetails map[string][]byte

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *cityworksSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *cityworksSvc) Tenant() string {
	return svc.tenant
}

func (svc *cityworksSvc) GetAll() []byte {
	svc.cityworksMutex.Lock()
	defer svc.cityworksMutex.Unlock()

	return svc.cityworks
}

func (svc *cityworksSvc) GetByID(id string) ([]byte, error) {
	svc.cityworksMutex.Lock()
	defer svc.cityworksMutex.Unlock()

	body, ok := svc.cityworksDetails[id]
	if !ok {
		return []byte{}, fmt.Errorf("no such cityworks")
	}

	return body, nil
}

func (svc *cityworksSvc) Start() {
	svc.log.Info().Msg("starting cityworks service")
	go svc.run()
}

func (svc *cityworksSvc) Shutdown() {
	svc.log.Info().Msg("shutting down cityworks service")
	svc.keepRunning = false
}

func (svc *cityworksSvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing cityworks info")
			count, err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh cityworks")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				svc.log.Info().Msgf("refreshed %d cityworks", count)
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		time.Sleep(1 * time.Second)
	}

	svc.log.Info().Msg("cityworks service exiting")
}

func (svc *cityworksSvc) refresh() (count int, err error) {

	ctx, span := tracer.Start(svc.ctx, "refresh-cityworks")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, _ = o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	cityworks := []domain.Cityworks{}

	count, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "CityWork", nil, func(c cityworksDTO) {
		location := *domain.NewPoint(c.Location.Coordinates[1], c.Location.Coordinates[0])

		details := domain.CityworksDetails{
			ID:           c.ID,
			Location:     location,
			Description:  c.Description,
			DateModified: c.DateModified.Value,
			StartDate:    c.StartDate.Value,
			EndDate:      c.EndDate.Value,
		}

		jsonBytes, err_ := json.MarshalIndent(details, "  ", "  ")
		if err_ != nil {
			err = fmt.Errorf("failed to marshal cityworks to json: %w", err_)
			return
		}

		svc.storeCityworksDetails(c.ID, jsonBytes)

		cw := domain.Cityworks{
			ID:        c.ID,
			Location:  location,
			StartDate: c.StartDate.Value,
			EndDate:   c.EndDate.Value,
		}

		cityworks = append(cityworks, cw)

	})
	if err != nil {
		err = fmt.Errorf("failed to retrieve cityworks from context broker: %w", err)
		return
	}

	jsonBytes, err_ := json.MarshalIndent(cityworks, "  ", "  ")
	if err_ != nil {
		err = fmt.Errorf("failed to marshal cityworks to json: %w", err_)
		return
	}

	svc.storeCityworksList(jsonBytes)

	return
}

func (svc *cityworksSvc) storeCityworksDetails(id string, body []byte) {
	svc.cityworksMutex.Lock()
	defer svc.cityworksMutex.Unlock()

	svc.cityworksDetails[id] = body
}

func (svc *cityworksSvc) storeCityworksList(body []byte) {
	svc.cityworksMutex.Lock()
	defer svc.cityworksMutex.Unlock()

	svc.cityworks = body
}

type cityworksDTO struct {
	ID       string `json:"id"`
	Location struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"location"`
	Description  string          `json:"description"`
	DateCreated  domain.DateTime `json:"dateCreated"`
	DateModified domain.DateTime `json:"dateModified"`
	StartDate    domain.DateTime `json:"startDate"`
	EndDate      domain.DateTime `json:"endDate"`
}
