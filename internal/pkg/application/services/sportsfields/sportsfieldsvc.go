package sportsfields

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

var tracer = otel.Tracer("api-opendata/svcs/sportsfields")

const (
	DefaultBrokerTenant string = "default"
)

type SportsFieldService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start()
	Shutdown()
}

func NewSportsFieldService(ctx context.Context, logger zerolog.Logger, contextBrokerURL, tenant string) SportsFieldService {
	svc := &sfsvc{
		ctx:                 ctx,
		sportsfields:        []byte{},
		sportsfieldsDetails: map[string][]byte{},
		contextBrokerURL:    contextBrokerURL,
		tenant:              tenant,
		log:                 logger,
		keepRunning:         true,
	}

	return svc
}

type sfsvc struct {
	ctx                 context.Context
	sportsfieldsMutex   sync.Mutex
	sportsfields        []byte
	sportsfieldsDetails map[string][]byte
	contextBrokerURL    string
	tenant              string
	log                 zerolog.Logger
	keepRunning         bool
}

func (s *sfsvc) Broker() string {
	return s.contextBrokerURL
}

func (s *sfsvc) Tenant() string {
	return s.tenant
}

func (s *sfsvc) GetAll() []byte {
	s.sportsfieldsMutex.Lock()
	defer s.sportsfieldsMutex.Unlock()

	return s.sportsfields
}

func (s *sfsvc) GetByID(id string) ([]byte, error) {
	s.sportsfieldsMutex.Lock()
	defer s.sportsfieldsMutex.Unlock()

	body, ok := s.sportsfieldsDetails[id]
	if !ok {
		return nil, fmt.Errorf("no such sports field")
	}

	return body, nil
}

func (s *sfsvc) Start() {
	s.log.Info().Msg("starting sports fields service")
	// TODO: Prevent multiple starts on the same service
	go s.run()
}

func (s *sfsvc) Shutdown() {
	s.log.Info().Msg("shutting down sports fields service")
	s.keepRunning = false
}

func (s *sfsvc) run() {
	nextRefreshTime := time.Now()

	for s.keepRunning {
		if time.Now().After(nextRefreshTime) {
			s.log.Info().Msg("refreshing sports field info")
			err := s.refresh()

			if err != nil {
				s.log.Error().Err(err).Msg("failed to refresh sports fields")
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

	s.log.Info().Msg("sports fields service exiting")
}

func (s *sfsvc) refresh() error {
	var err error
	ctx, span := tracer.Start(s.ctx, "refresh-sports-fields")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, s.log, ctx)

	sportsfields := []domain.SportsField{}

	err = s.getSportsFieldsFromContextBroker(ctx, func(sf sportsFieldsDTO) {

		details := domain.SportsFieldDetails{
			ID:               sf.ID,
			Name:             sf.Name,
			Description:      sf.Description,
			Category:         sf.Category,
			Geometry:         sf.Geometry,
			DateCreated:      sf.DateCreated,
			DateModified:     sf.DateModified,
			DateLastPrepared: sf.DateLastPrepared,
			Source:           sf.Source,
		}

		jsonBytes, err := json.Marshal(details)
		if err != nil {
			logger.Error().Err(err).Msg("failed to marshal sports field details to json")
			return
		}

		s.storeSportsFieldDetails(sf.ID, jsonBytes)

		sportfield := domain.SportsField{
			Name:             sf.Name,
			Category:         sf.Category,
			Geometry:         details.Geometry,
			DateLastPrepared: details.DateLastPrepared,
		}

		sportsfields = append(sportsfields, sportfield)
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve sports fields from context broker")
		return err
	}

	jsonBytes, err := json.Marshal(sportsfields)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal sports fields to json")
		return err
	}

	s.storeSportsFieldList(jsonBytes)

	return err
}

func (s *sfsvc) getSportsFieldsFromContextBroker(ctx context.Context, callback func(sf sportsFieldsDTO)) error {
	var err error

	logger := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.contextBrokerURL+"/ngsi-ld/v1/entities?type=SportsField&limit=1000&options=keyValues", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Add("Accept", "application/ld+json")
	linkHeaderURL := fmt.Sprintf("<%s>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"", entities.DefaultContextURL)
	req.Header.Add("Link", linkHeaderURL)

	if s.tenant != DefaultBrokerTenant {
		req.Header.Add("NGSILD-Tenant", s.tenant)
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

	var sportsfields []sportsFieldsDTO
	err = json.Unmarshal(respBody, &sportsfields)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	for _, sf := range sportsfields {
		callback(sf)
	}

	return nil
}

func (s *sfsvc) storeSportsFieldDetails(id string, body []byte) {
	s.sportsfieldsMutex.Lock()
	defer s.sportsfieldsMutex.Unlock()

	s.sportsfieldsDetails[id] = body
}

func (s *sfsvc) storeSportsFieldList(body []byte) {
	s.sportsfieldsMutex.Lock()
	defer s.sportsfieldsMutex.Unlock()

	s.sportsfields = body
}

type sportsFieldsDTO struct {
	ID               string
	Name             string
	Description      string
	Category         []string
	Geometry         domain.MultiPolygon
	DateCreated      domain.DateTime
	DateModified     domain.DateTime
	DateLastPrepared domain.DateTime
	Source           string
}
