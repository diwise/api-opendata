package sportsfields

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/application/services/organisations"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/sportsfields")

type SportsFieldService interface {
	Broker() string
	Tenant() string

	GetAll(requiredCategories []string) []domain.SportsField
	GetByID(id string) (*domain.SportsField, error)

	Start(ctx context.Context)
	Shutdown(ctx context.Context)
}

func NewSportsFieldService(ctx context.Context, contextBrokerURL, tenant string, orgreg organisations.Registry) SportsFieldService {
	svc := &sportsfieldSvc{
		sportsfields:        []domain.SportsField{},
		sportsfieldsDetails: map[string]int{},
		orgRegistry:         orgreg,
		contextBrokerURL:    contextBrokerURL,
		tenant:              tenant,
		keepRunning:         true,
	}

	return svc
}

type sportsfieldSvc struct {
	sportsfieldsMutex   sync.Mutex
	sportsfields        []domain.SportsField
	sportsfieldsDetails map[string]int
	orgRegistry         organisations.Registry
	contextBrokerURL    string
	tenant              string
	keepRunning         bool
}

func (svc *sportsfieldSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *sportsfieldSvc) Tenant() string {
	return svc.tenant
}

func (svc *sportsfieldSvc) GetAll(requiredCategories []string) []domain.SportsField {
	svc.sportsfieldsMutex.Lock()
	defer svc.sportsfieldsMutex.Unlock()

	if len(requiredCategories) == 0 {
		return svc.sportsfields
	}

	result := make([]domain.SportsField, 0, len(svc.sportsfields))

	anyCategoryMatches := func(categories []string) bool {
		for _, category := range categories {
			for _, requiredCategory := range requiredCategories {
				if category == requiredCategory {
					return true
				}
			}
		}

		return false
	}

	for idx := range svc.sportsfields {
		if anyCategoryMatches(svc.sportsfields[idx].Categories) {
			result = append(result, svc.sportsfields[idx])
		}
	}

	return result
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

func (svc *sportsfieldSvc) Start(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info().Msg("starting sports fields service")
	// TODO: Prevent multiple starts on the same service
	go svc.run(ctx)
}

func (svc *sportsfieldSvc) Shutdown(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info().Msg("shutting down sports fields service")
	svc.keepRunning = false
}

func (svc *sportsfieldSvc) run(ctx context.Context) {
	nextRefreshTime := time.Now()
	logger := logging.GetFromContext(ctx)

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			logger.Info().Msg("refreshing sports field info")
			count, err := svc.refresh(ctx, logger)

			if err != nil {
				logger.Error().Err(err).Msg("failed to refresh sports fields")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				logger.Info().Msgf("refreshed %d sports fields", count)
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		// TODO: Use blocking channels instead of sleeps
		time.Sleep(1 * time.Second)
	}

	logger.Info().Msg("sports fields service exiting")
}

func (svc *sportsfieldSvc) refresh(ctx context.Context, logger zerolog.Logger) (count int, err error) {

	ctx, span := tracer.Start(ctx, "refresh-sports-fields")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, _ = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	sportsfields := []domain.SportsField{}

	count, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "SportsField", nil, func(sf sportsFieldDTO) {

		sportsfield := domain.SportsField{
			ID:           sf.ID,
			Name:         sf.Name,
			Description:  sf.Description,
			Categories:   sf.Categories(),
			PublicAccess: sf.PublicAccess,
			Location:     sf.Location,
			Source:       sf.Source,
			Status:       sf.Status,
		}

		if len(sf.ManagedBy) > 0 {
			sportsfield.ManagedBy, err = svc.orgRegistry.Get(sf.ManagedBy)
			if err != nil {
				logger.Error().Err(err).Msg("failed to resolve organisation")
			}
		}

		if len(sf.Owner) > 0 {
			sportsfield.Owner, err = svc.orgRegistry.Get(sf.Owner)
			if err != nil {
				logger.Error().Err(err).Msg("failed to resolve organisation")
			}
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
	PublicAccess        string              `json:"publicAccess"`
	Location            domain.MultiPolygon `json:"location"`
	DateCreated         *domain.DateTime    `json:"dateCreated"`
	DateModified        *domain.DateTime    `json:"dateModified,omitempty"`
	DateLastPreparation *domain.DateTime    `json:"dateLastPreparation,omitempty"`
	Source              string              `json:"source"`
	ManagedBy           string              `json:"managedBy"`
	Owner               string              `json:"owner"`
	Status              string              `json:"status"`
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
