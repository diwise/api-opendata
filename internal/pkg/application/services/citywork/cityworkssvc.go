package citywork

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/cityworks")

type CityworksService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start(ctx context.Context)
	Shutdown(ctx context.Context)
}

func NewCityworksService(ctx context.Context, contextBrokerUrl, tenant string) CityworksService {
	svc := &cityworksSvc{
		cityworks:        []byte("[]"),
		cityworksDetails: map[string][]byte{},
		contextBrokerURL: contextBrokerUrl,
		tenant:           tenant,

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

func (svc *cityworksSvc) Start(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info("starting cityworks service")
	go svc.run(ctx)
}

func (svc *cityworksSvc) Shutdown(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info("shutting down cityworks service")
	svc.keepRunning = false
}

func (svc *cityworksSvc) run(ctx context.Context) {
	nextRefreshTime := time.Now()
	logger := logging.GetFromContext(ctx)

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			logger.Info("refreshing cityworks info")
			count, err := svc.refresh(ctx)

			if err != nil {
				logger.Error("failed to refresh cityworks", slog.String("error", err.Error()))
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				logger.Info("refreshed cityworks", slog.Int("count", count))
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		time.Sleep(1 * time.Second)
	}

	logger.Info("cityworks service exiting")
}

func (svc *cityworksSvc) refresh(ctx context.Context) (count int, err error) {
	logger := logging.GetFromContext(ctx)

	ctx, span := tracer.Start(ctx, "refresh-cityworks")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, _ = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

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
