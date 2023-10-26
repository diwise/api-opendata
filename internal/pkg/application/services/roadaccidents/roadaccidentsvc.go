package roadaccidents

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

var tracer = otel.Tracer("api-opendata/svcs/roadaccidents")

type RoadAccidentService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start(ctx context.Context)
	Shutdown(ctx context.Context)
}

func NewRoadAccidentService(ctx context.Context, contextBrokerURL, tenant string) RoadAccidentService {
	svc := &roadAccidentSvc{
		contextBrokerURL: contextBrokerURL,
		tenant:           tenant,

		roadAccidents:       []byte("[]"),
		roadAccidentDetails: map[string][]byte{},

		keepRunning: true,
	}

	return svc
}

type roadAccidentSvc struct {
	contextBrokerURL string
	tenant           string

	roadAccidentMutex   sync.Mutex
	roadAccidents       []byte
	roadAccidentDetails map[string][]byte

	keepRunning bool
}

func (svc *roadAccidentSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *roadAccidentSvc) Tenant() string {
	return svc.tenant
}

func (svc *roadAccidentSvc) GetAll() []byte {
	svc.roadAccidentMutex.Lock()
	defer svc.roadAccidentMutex.Unlock()

	return svc.roadAccidents
}

func (svc *roadAccidentSvc) GetByID(id string) ([]byte, error) {
	svc.roadAccidentMutex.Lock()
	defer svc.roadAccidentMutex.Unlock()

	body, ok := svc.roadAccidentDetails[id]
	if !ok {
		return []byte{}, fmt.Errorf("no such road accident")
	}

	return body, nil
}

func (svc *roadAccidentSvc) Start(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info("starting road accident service")
	// TODO: Prevent multiple starts on the same service
	go svc.run(ctx)
}

func (svc *roadAccidentSvc) Shutdown(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info("shutting down road accident service")
	svc.keepRunning = false
}

func (svc *roadAccidentSvc) run(ctx context.Context) {
	nextRefreshTime := time.Now()
	logger := logging.GetFromContext(ctx)

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			logger.Info("refreshing road accident info")
			count, err := svc.refresh(ctx)

			if err != nil {
				logger.Error("failed to refresh road accidents", slog.String("error", err.Error()))
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				logger.Info("refreshed road accidents", slog.Int("count", count))
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		// TODO: Use blocking channels instead of sleeps
		time.Sleep(1 * time.Second)
	}

	logger.Info("road accident service exiting")
}

func (svc *roadAccidentSvc) refresh(ctx context.Context) (count int, err error) {
	log := logging.GetFromContext(ctx)

	ctx, span := tracer.Start(ctx, "refresh-roadaccidents")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, _ = o11y.AddTraceIDToLoggerAndStoreInContext(span, log, ctx)

	roadAccidents := []domain.RoadAccident{}

	count, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "RoadAccident", nil, func(r roadAccidentDTO) {

		details := domain.RoadAccidentDetails{
			ID:           r.ID,
			Description:  r.Description,
			Location:     *domain.NewPoint(r.Location.Coordinates[1], r.Location.Coordinates[0]),
			DateCreated:  r.DateCreated.Value,
			AccidentDate: r.AccidentDate.Value,
			DateModified: r.DateModified.Value,
			Status:       r.Status,
		}

		jsonBytes, err_ := json.MarshalIndent(details, "  ", "  ")
		if err_ != nil {
			err = fmt.Errorf("failed to marshal road accident to json: %w", err_)
			return
		}

		svc.storeRoadAccidentDetails(r.ID, jsonBytes)

		roadAccident := domain.RoadAccident{
			ID:           r.ID,
			Location:     details.Location,
			AccidentDate: r.AccidentDate.Value,
		}

		roadAccidents = append(roadAccidents, roadAccident)
	})
	if err != nil {
		err = fmt.Errorf("failed to retrieve road accidents from context broker: %w", err)
		return
	}

	jsonBytes, err_ := json.MarshalIndent(roadAccidents, "  ", "  ")
	if err_ != nil {
		err = fmt.Errorf("failed to marshal road accidents to json: %w", err_)
		return
	}

	svc.storeRoadAccidentList(jsonBytes)

	return
}

func (svc *roadAccidentSvc) storeRoadAccidentDetails(id string, body []byte) {
	svc.roadAccidentMutex.Lock()
	defer svc.roadAccidentMutex.Unlock()

	svc.roadAccidentDetails[id] = body

}

func (svc *roadAccidentSvc) storeRoadAccidentList(body []byte) {
	svc.roadAccidentMutex.Lock()
	defer svc.roadAccidentMutex.Unlock()

	svc.roadAccidents = body

}

type roadAccidentDTO struct {
	ID           string          `json:"id"`
	AccidentDate domain.DateTime `json:"accidentDate"`
	Description  string          `json:"description"`
	Location     struct {
		Type        string     `json:"type"`
		Coordinates [2]float64 `json:"coordinates"`
	} `json:"location"`
	DateCreated  domain.DateTime `json:"dateCreated"`
	DateModified domain.DateTime `json:"dateModified"`
	Status       string          `json:"status"`
}
