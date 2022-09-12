package roadaccidents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/roadaccidents")

const (
	DefaultBrokerTenant string = "default"
)

type RoadAccidentService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start()
	Shutdown()
}

func NewRoadAccidentService(ctx context.Context, logger zerolog.Logger, contextBrokerURL, tenant string) RoadAccidentService {
	svc := &roadAccidentSvc{
		contextBrokerURL: contextBrokerURL,
		tenant:           tenant,

		roadAccidents:       []byte("[]"),
		roadAccidentDetails: map[string][]byte{},

		ctx: ctx,
		log: logger,

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

	ctx context.Context
	log zerolog.Logger

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

func (svc *roadAccidentSvc) Start() {
	svc.log.Info().Msg("starting road accident service")
	// TODO: Prevent multiple starts on the same service
	go svc.run()
}

func (svc *roadAccidentSvc) Shutdown() {
	svc.log.Info().Msg("shutting down road accident service")
	svc.keepRunning = false
}

func (svc *roadAccidentSvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing road accident info")
			err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh road accidents")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		// TODO: Use blocking channels instead of sleeps
		time.Sleep(1 * time.Second)
	}

	svc.log.Info().Msg("road accident service exiting")
}

func (svc *roadAccidentSvc) refresh() error {
	var err error
	ctx, span := tracer.Start(svc.ctx, "refresh-roadaccidents")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	roadAccidents := []domain.RoadAccident{}

	err = svc.getRoadAccidentsFromContextBroker(ctx, func(r roadAccidentDTO) {

		details := domain.RoadAccidentDetails{
			ID:           r.ID,
			Description:  r.Description,
			Location:     *domain.NewPoint(r.Location.Coordinates[1], r.Location.Coordinates[0]),
			DateCreated:  r.DateCreated,
			DateModified: r.DateModified,
			Status:       r.Status,
		}

		jsonBytes, err := json.MarshalIndent(details, "  ", "  ")
		if err != nil {
			logger.Error().Err(err).Msg("failed to marshal road accident to json")
			return
		}

		svc.storeRoadAccidentDetails(r.ID, jsonBytes)

		roadAccident := domain.RoadAccident{
			ID:           r.ID,
			Location:     details.Location,
			AccidentDate: r.AccidentDate,
		}

		roadAccidents = append(roadAccidents, roadAccident)
	})

	jsonBytes, err := json.MarshalIndent(roadAccidents, "  ", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal road accidents to json")
		return err
	}

	svc.storeRoadAccidentList(jsonBytes)

	return err
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

func (svc *roadAccidentSvc) getRoadAccidentsFromContextBroker(ctx context.Context, callback func(r roadAccidentDTO)) error {
	var err error

	logger := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.contextBrokerURL+"/ngsi-ld/v1/entities?type=RoadAccident&limit=100&options=keyValues", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Add("Accept", "application/ld+json")
	linkHeaderURL := fmt.Sprintf("<%s>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"", entities.DefaultContextURL)
	req.Header.Add("Link", linkHeaderURL)

	if svc.tenant != DefaultBrokerTenant {
		req.Header.Add("NGSILD-Tenant", svc.tenant)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %s", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %s", err.Error())
	}

	if resp.StatusCode >= http.StatusBadRequest {
		reqbytes, _ := httputil.DumpRequest(req, false)
		respbytes, _ := httputil.DumpResponse(resp, false)

		logger.Error().Str("request", string(reqbytes)).Str("response", string(respbytes)).Msg("request failed")
		return fmt.Errorf("request failed")
	}

	if resp.StatusCode != http.StatusOK {
		contentType := resp.Header.Get("Content-Type")
		return fmt.Errorf("context source returned status code %d (content-type: %s, body: %s)", resp.StatusCode, contentType, string(respBody))
	}

	var roadAccidents []roadAccidentDTO
	err = json.Unmarshal(respBody, &roadAccidents)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	for _, r := range roadAccidents {
		callback(r)
	}

	return err
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
