package exercisetrails

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

var tracer = otel.Tracer("api-opendata/svcs/exercisetrails")

const (
	DefaultBrokerTenant string = "default"
)

type ExerciseTrailService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start()
	Shutdown()
}

func NewExerciseTrailService(ctx context.Context, logger zerolog.Logger, contextBrokerURL, tenant string) ExerciseTrailService {
	svc := &exerciseTrailSvc{
		ctx:              ctx,
		trails:           []byte("[]"),
		trailDetails:     map[string][]byte{},
		contextBrokerURL: contextBrokerURL,
		tenant:           tenant,
		log:              logger,
		keepRunning:      true,
	}

	return svc
}

type exerciseTrailSvc struct {
	contextBrokerURL string
	tenant           string

	trailMutex   sync.Mutex
	trails       []byte
	trailDetails map[string][]byte

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *exerciseTrailSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *exerciseTrailSvc) Tenant() string {
	return svc.tenant
}

func (svc *exerciseTrailSvc) GetAll() []byte {
	svc.trailMutex.Lock()
	defer svc.trailMutex.Unlock()

	return svc.trails
}

func (svc *exerciseTrailSvc) GetByID(id string) ([]byte, error) {
	svc.trailMutex.Lock()
	defer svc.trailMutex.Unlock()

	body, ok := svc.trailDetails[id]
	if !ok {
		return []byte{}, fmt.Errorf("no such exercisetrail")
	}

	return body, nil
}

func (svc *exerciseTrailSvc) Start() {
	svc.log.Info().Msg("starting exercise trail service")
	// TODO: Prevent multiple starts on the same service
	go svc.run()
}

func (svc *exerciseTrailSvc) Shutdown() {
	svc.log.Info().Msg("shutting down exercise trail service")
	svc.keepRunning = false
}

func (svc *exerciseTrailSvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing execise trail info")
			err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh exercise trails")
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

	svc.log.Info().Msg("exercise trail service exiting")
}

func (svc *exerciseTrailSvc) refresh() error {
	var err error
	ctx, span := tracer.Start(svc.ctx, "refresh-trails")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	trails := []domain.ExerciseTrail{}

	err = svc.getExerciseTrailsFromContextBroker(ctx, func(t trailDTO) {
		details := domain.ExerciseTrailDetails{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Categories:  t.Categories(),
			Location:    *domain.NewLineString(t.Location.Coordinates),
			Length:      t.Length,
			Status:      t.Status,
			Source:      t.Source,
			AreaServed:  t.AreaServed,
		}

		jsonBytes, err := json.MarshalIndent(details, "  ", "  ")
		if err != nil {
			logger.Error().Err(err).Msg("failed to marshal exercise trail to json")
			return
		}

		svc.storeExerciseTrailDetails(t.ID, jsonBytes)

		lat, lon := t.LatLon()
		trail := domain.ExerciseTrail{
			ID:         t.ID,
			Name:       t.Name,
			Categories: t.Categories(),
			Location:   *domain.NewPoint(lat, lon),
			Length:     t.Length,
			Status:     t.Status,
		}

		trails = append(trails, trail)
	})

	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(trails, "  ", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal exercise trails to json")
		return err
	}

	svc.storeExerciseTrailList(jsonBytes)

	return err
}

func (svc *exerciseTrailSvc) storeExerciseTrailDetails(id string, body []byte) {
	svc.trailMutex.Lock()
	defer svc.trailMutex.Unlock()

	svc.trailDetails[id] = body
}

func (svc *exerciseTrailSvc) storeExerciseTrailList(body []byte) {
	svc.trailMutex.Lock()
	defer svc.trailMutex.Unlock()

	svc.trails = body
}

func (svc *exerciseTrailSvc) getExerciseTrailsFromContextBroker(ctx context.Context, callback func(b trailDTO)) error {
	var err error

	logger := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.contextBrokerURL+"/ngsi-ld/v1/entities?type=ExerciseTrail&limit=1000&options=keyValues", nil)
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

	var trails []trailDTO
	err = json.Unmarshal(respBody, &trails)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	for _, t := range trails {
		callback(t)
	}

	return nil
}

type trailDTO struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    json.RawMessage `json:"category"`
	Location    struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates"`
	} `json:"location"`
	Length       float64         `json:"length"`
	Source       string          `json:"source"`
	Status       string          `json:"status"`
	AreaServed   string          `json:"areaServed"`
	DateModified domain.DateTime `json:"dateModified"`
}

// LatLon tries to guess a suitable location point by assuming that the
// first coordinate is the start of the trail (not always true ofcourse)
func (t *trailDTO) LatLon() (float64, float64) {
	lat := 0.0
	lon := 0.0

	if len(t.Location.Coordinates) > 0 {
		start := t.Location.Coordinates[0]
		return start[1], start[0]
	}

	return lat, lon
}

// Categories extracts the trail categories as a string array, regardless
// of the format string vs []string of the response property
func (t *trailDTO) Categories() []string {
	catsAsArray := []string{}

	if len(t.Category) > 0 {
		if err := json.Unmarshal(t.Category, &catsAsArray); err != nil {
			var categoryAsString string

			if err = json.Unmarshal(t.Category, &categoryAsString); err != nil {
				return []string{err.Error()}
			}

			return []string{categoryAsString}
		}
	}

	return catsAsArray
}