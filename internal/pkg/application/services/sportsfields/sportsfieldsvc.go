package sportsfields

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

var tracer = otel.Tracer("api-opendata/svcs/sportsfields")

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
		return nil, fmt.Errorf("no such sports field")
	}

	return &svc.sportsfields[index], nil
}

func (svc *sportsfieldSvc) Start() {
	svc.log.Info().Msg("starting sports fields service")
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
			svc.log.Info().Msg("refreshing sports field info")
			count, err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh sports fields")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				svc.log.Info().Msgf("refreshed %d sports fields", count)
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		// TODO: Use blocking channels instead of sleeps
		time.Sleep(1 * time.Second)
	}

	svc.log.Info().Msg("sports fields service exiting")
}

func (svc *sportsfieldSvc) refresh() (count int, err error) {

	ctx, span := tracer.Start(svc.ctx, "refresh-sports-fields")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, _ = o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	sportsfields := []domain.SportsField{}

	count, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "SportsField", nil, func(sf sportsFieldDTO) {

		sportsfield := domain.SportsField{
			ID:          sf.ID,
			Name:        sf.Name,
			Description: sf.Description,
			Categories:  sf.Categories(),
			Location:    sf.Location,
			Source:      sf.Source,
		}

		if sf.DateCreated != nil {
			sportsfield.DateCreated = &sf.DateCreated.Value
		}
		if sf.DateModified != nil {
			sportsfield.DateModified = &sf.DateModified.Value
		}
		if sf.DateLastPreparation != nil {
			sportsfield.DateLastPreparation = &sf.DateLastPreparation.Value
		}

		sportsfields = append(sportsfields, sportsfield)
	})

	if err != nil {
		err = fmt.Errorf("failed to retrieve sports fields from context broker: %w", err)
		return
	}

	svc.storeSportsFieldList(sportsfields)

	return
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

type sportsFieldDTO struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	Category            json.RawMessage     `json:"category"`
	Location            domain.MultiPolygon `json:"location"`
	DateCreated         *domain.DateTime    `json:"dateCreated"`
	DateModified        *domain.DateTime    `json:"dateModified,omitempty"`
	DateLastPreparation *domain.DateTime    `json:"dateLastPreparation,omitempty"`
	Source              string              `json:"source"`
}

// Categories extracts the field categories as a string array, regardless
// of the format string vs []string of the response property
func (f *sportsFieldDTO) Categories() []string {
	valueAsArray := []string{}

	if len(f.Category) > 0 {
		if err := json.Unmarshal(f.Category, &valueAsArray); err != nil {
			var valueAsString string

			if err = json.Unmarshal(f.Category, &valueAsString); err != nil {
				return []string{err.Error()}
			}

			return []string{valueAsString}
		}
	}

	return valueAsArray
}
