package sportsvenues

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
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/sportsvenues")

type SportsVenueService interface {
	Broker() string
	Tenant() string

	GetAll(requiredCategories []string) []domain.SportsVenue
	GetByID(id string) (*domain.SportsVenue, error)

	Start(ctx context.Context)
	Shutdown(ctx context.Context)
}

func NewSportsVenueService(ctx context.Context, contextBrokerURL, tenant string, orgreg organisations.Registry) SportsVenueService {
	svc := &sportsvenueSvc{
		sportsvenues:        []domain.SportsVenue{},
		sportsvenuesDetails: map[string]int{},
		contextBrokerURL:    contextBrokerURL,
		orgRegistry:         orgreg,
		tenant:              tenant,
		keepRunning:         true,
	}

	return svc
}

type sportsvenueSvc struct {
	sportsvenuesMutex   sync.Mutex
	sportsvenues        []domain.SportsVenue
	sportsvenuesDetails map[string]int
	orgRegistry         organisations.Registry
	contextBrokerURL    string
	tenant              string
	keepRunning         bool
}

func (svc *sportsvenueSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *sportsvenueSvc) Tenant() string {
	return svc.tenant
}

func (svc *sportsvenueSvc) GetAll(requiredCategories []string) []domain.SportsVenue {
	svc.sportsvenuesMutex.Lock()
	defer svc.sportsvenuesMutex.Unlock()

	if len(requiredCategories) == 0 {
		return svc.sportsvenues
	}

	result := make([]domain.SportsVenue, 0, len(svc.sportsvenues))

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

	for idx := range svc.sportsvenues {
		if anyCategoryMatches(svc.sportsvenues[idx].Categories) {
			result = append(result, svc.sportsvenues[idx])
		}
	}

	return result
}

func (svc *sportsvenueSvc) GetByID(id string) (*domain.SportsVenue, error) {
	svc.sportsvenuesMutex.Lock()
	defer svc.sportsvenuesMutex.Unlock()

	index, ok := svc.sportsvenuesDetails[id]
	if !ok {
		return nil, fmt.Errorf("no such sports venue")
	}

	return &svc.sportsvenues[index], nil
}

func (svc *sportsvenueSvc) Start(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info("starting sports venues service")
	// TODO: Prevent multiple starts on the same service
	go svc.run(ctx)
}

func (svc *sportsvenueSvc) Shutdown(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info("shutting down sports venues service")
	svc.keepRunning = false
}

func (svc *sportsvenueSvc) run(ctx context.Context) {
	nextRefreshTime := time.Now()
	logger := logging.GetFromContext(ctx)

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			logger.Info("refreshing sports venue info")
			count, err := svc.refresh(ctx)

			if err != nil {
				logger.Error("failed to refresh sports venues", "error", err)
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				logger.Info("refreshed sports venues", "count", count)
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		// TODO: Use blocking channels instead of sleeps
		time.Sleep(1 * time.Second)
	}

	logger.Info("sports venues service exiting")
}

func (svc *sportsvenueSvc) refresh(ctx context.Context) (count int, err error) {

	ctx, span := tracer.Start(ctx, "refresh-sports-venues")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

	sportsvenues := []domain.SportsVenue{}

	count, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "SportsVenue", nil, func(sv sportsVenueDTO) {

		venue := domain.SportsVenue{
			ID:           sv.ID,
			Name:         sv.Name,
			Description:  sv.Description,
			Categories:   sv.Categories(),
			PublicAccess: sv.PublicAccess,
			Location:     sv.Location,
			Source:       sv.Source,
			SeeAlso:      sv.SeeAlso(),
		}

		if len(sv.ManagedBy) > 0 {
			venue.ManagedBy, err = svc.orgRegistry.Get(sv.ManagedBy)
			if err != nil {
				logger.Error("failed to resolve organisation", "error", err)
			}
		}

		if len(sv.Owner) > 0 {
			venue.Owner, err = svc.orgRegistry.Get(sv.Owner)
			if err != nil {
				logger.Error("failed to resolve organisation", "error", err)
			}
		}

		if sv.DateCreated != nil {
			venue.DateCreated = &sv.DateCreated.Value
		}
		if sv.DateModified != nil {
			venue.DateModified = &sv.DateModified.Value
		}

		sportsvenues = append(sportsvenues, venue)
	})

	if err != nil {
		err = fmt.Errorf("failed to retrieve sports venues from context broker: %w", err)
		return
	}

	svc.storeSportsVenueList(sportsvenues)

	return
}

func (svc *sportsvenueSvc) storeSportsVenueList(list []domain.SportsVenue) {
	svc.sportsvenuesMutex.Lock()
	defer svc.sportsvenuesMutex.Unlock()

	svc.sportsvenues = list
	svc.sportsvenuesDetails = map[string]int{}

	for index := range list {
		svc.sportsvenuesDetails[list[index].ID] = index
	}
}

type sportsVenueDTO struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	Category     json.RawMessage     `json:"category"`
	PublicAccess string              `json:"publicAccess"`
	Location     domain.MultiPolygon `json:"location"`
	DateCreated  *domain.DateTime    `json:"dateCreated"`
	DateModified *domain.DateTime    `json:"dateModified,omitempty"`
	See          json.RawMessage     `json:"seeAlso"`
	Source       string              `json:"source"`
	ManagedBy    string              `json:"managedBy"`
	Owner        string              `json:"owner"`
}

// Categories extracts the venue categories as a string array, regardless
// of the format string vs []string of the response property
func (v *sportsVenueDTO) Categories() []string {
	return rawJSONToSliceOfStrings(v.Category)
}

// SeeAlso extracts the venue reference links as a string array, regardless
// of the format string vs []string of the response property
func (v *sportsVenueDTO) SeeAlso() []string {
	return rawJSONToSliceOfStrings(v.See)
}

func rawJSONToSliceOfStrings(rm json.RawMessage) []string {
	valueAsArray := []string{}

	if len(rm) > 0 {
		if err := json.Unmarshal(rm, &valueAsArray); err != nil {
			var valueAsString string

			if err = json.Unmarshal(rm, &valueAsString); err != nil {
				return []string{err.Error()}
			}

			return []string{valueAsString}
		}
	}

	return valueAsArray
}
