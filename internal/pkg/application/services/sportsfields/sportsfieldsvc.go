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

	GetAll() []domain.SportsField
	GetByID(id string) (*domain.SportsField, error)

	Start()
	Shutdown()
}

func NewSportsFieldService(ctx context.Context, logger zerolog.Logger, contextBrokerURL, tenant string) SportsFieldService {
	svc := &sportsfieldSvc{
		ctx:                 ctx,
		sportsfields:        []domain.SportsField{},
		sportsfieldsDetails: map[string]int{},
		contextBrokerURL:    contextBrokerURL,
		tenant:              tenant,
		log:                 logger,
		keepRunning:         true,
	}

	return svc
}

type sportsfieldSvc struct {
	ctx                 context.Context
	sportsfieldsMutex   sync.Mutex
	sportsfields        []domain.SportsField
	sportsfieldsDetails map[string]int
	contextBrokerURL    string
	tenant              string
	log                 zerolog.Logger
	keepRunning         bool
}

func (svc *sportsfieldSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *sportsfieldSvc) Tenant() string {
	return svc.tenant
}

func (svc *sportsfieldSvc) GetAll() []domain.SportsField {
	svc.sportsfieldsMutex.Lock()
	defer svc.sportsfieldsMutex.Unlock()

	return svc.sportsfields
}

func (svc *sportsfieldSvc) GetByID(id string) (*domain.SportsField, error) {
	svc.sportsfieldsMutex.Lock()
	defer svc.sportsfieldsMutex.Unlock()

	index, ok := svc.sportsfieldsDetails[id]
	if !ok {
		return nil, fmt.Errorf("no such sportsfield")
	}

	return &svc.sportsfields[index], nil
}

func (svc *sportsfieldSvc) Start() {
	svc.log.Info().Msg("starting sportsfields service")
	// TODO: Prevent multiple starts on the same service
	go svc.run()
}

func (svc *sportsfieldSvc) Shutdown() {
	svc.log.Info().Msg("shutting down sports fields service")
	svc.keepRunning = false
}

func (svc *sportsfieldSvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing sportsfield info")
			err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh sportsfields")
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

	svc.log.Info().Msg("sportsfields service exiting")
}

func (svc *sportsfieldSvc) refresh() error {
	var err error
	ctx, span := tracer.Start(svc.ctx, "refresh-sports-fields")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	sportsfields := []domain.SportsField{}

	err = svc.getSportsFieldsFromContextBroker(ctx, func(sf sportsFieldsDTO) {

		sportsfield := domain.SportsField{
			ID:          sf.ID,
			Name:        sf.Name.Value,
			Description: sf.Description.Value,
			Categories:  sf.Category.Value,
			Location:    sf.Location.Value,
			Source:      sf.Source,
		}

		if sf.DateCreated != nil {
			sportsfield.DateCreated = &sf.DateCreated.Value.Value
		}
		if sf.DateModified != nil {
			sportsfield.DateModified = &sf.DateModified.Value.Value
		}
		if sf.DateLastPreparation != nil {
			sportsfield.DateLastPreparation = &sf.DateLastPreparation.Value.Value
		}

		sportsfields = append(sportsfields, sportsfield)
	})

	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve sports fields from context broker")
		return err
	}

	svc.storeSportsFieldList(sportsfields)

	return err
}

func (svc *sportsfieldSvc) getSportsFieldsFromContextBroker(ctx context.Context, callback func(sf sportsFieldsDTO)) error {
	var err error

	logger := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.contextBrokerURL+"/ngsi-ld/v1/entities?type=SportsField&limit=1000&options=keyValues", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Add("Link", entities.LinkHeader)

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

func (svc *sportsfieldSvc) storeSportsFieldList(list []domain.SportsField) {
	svc.sportsfieldsMutex.Lock()
	defer svc.sportsfieldsMutex.Unlock()

	svc.sportsfields = list
	svc.sportsfieldsDetails = map[string]int{}

	for index := range list {
		svc.sportsfieldsDetails[list[index].ID] = index
	}
}

type sportsFieldsDTO struct {
	ID                  string          `json:"id"`
	Name                domain.Text     `json:"name"`
	Description         domain.Text     `json:"description"`
	Category            domain.TextList `json:"category"`
	Location            Location        `json:"location"`
	DateCreated         *DateTime       `json:"dateCreated"`
	DateModified        *DateTime       `json:"dateModified,omitempty"`
	DateLastPreparation *DateTime       `json:"dateLastPreparation,omitempty"`
	Source              domain.Text     `json:"source"`
}

type Location struct {
	Type  string              `json:"type"`
	Value domain.MultiPolygon `json:"value"`
}

type DateTime struct {
	Type  string `json:"type"`
	Value struct {
		Type  string `json:"@type"`
		Value string `json:"@value"`
	} `json:"value"`
}
