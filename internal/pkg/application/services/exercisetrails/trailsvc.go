package exercisetrails

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/application/services/organisations"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/exercisetrails")

type ExerciseTrailService interface {
	Broker() string
	Tenant() string

	GetAll(requiredCategories []string) []domain.ExerciseTrail
	GetByID(id string) (*domain.ExerciseTrail, error)

	Start(ctx context.Context)
	Shutdown(ctx context.Context)
}

func NewExerciseTrailService(ctx context.Context, contextBrokerURL, tenant string, orgreg organisations.Registry) ExerciseTrailService {
	svc := &exerciseTrailSvc{
		trails:           []domain.ExerciseTrail{},
		trailDetails:     map[string]int{},
		orgRegistry:      orgreg,
		contextBrokerURL: contextBrokerURL,
		tenant:           tenant,
		keepRunning:      true,
	}

	return svc
}

type exerciseTrailSvc struct {
	contextBrokerURL string
	tenant           string

	orgRegistry organisations.Registry

	trailMutex   sync.Mutex
	trails       []domain.ExerciseTrail
	trailDetails map[string]int

	keepRunning bool
}

func (svc *exerciseTrailSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *exerciseTrailSvc) Tenant() string {
	return svc.tenant
}

func (svc *exerciseTrailSvc) GetAll(requiredCategories []string) []domain.ExerciseTrail {
	svc.trailMutex.Lock()
	defer svc.trailMutex.Unlock()

	if len(requiredCategories) == 0 {
		return svc.trails
	}

	result := make([]domain.ExerciseTrail, 0, len(svc.trails))

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

	for idx := range svc.trails {
		if anyCategoryMatches(svc.trails[idx].Categories) {
			result = append(result, svc.trails[idx])
		}
	}

	return result
}

func (svc *exerciseTrailSvc) GetByID(id string) (*domain.ExerciseTrail, error) {
	svc.trailMutex.Lock()
	defer svc.trailMutex.Unlock()

	index, ok := svc.trailDetails[id]
	if !ok {
		return nil, fmt.Errorf("no such exercisetrail")
	}

	return &svc.trails[index], nil
}

func (svc *exerciseTrailSvc) Start(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info("starting exercise trail service")
	// TODO: Prevent multiple starts on the same service
	go svc.run(ctx)
}

func (svc *exerciseTrailSvc) Shutdown(ctx context.Context) {
	logger := logging.GetFromContext(ctx)
	logger.Info("shutting down exercise trail service")
	svc.keepRunning = false
}

func (svc *exerciseTrailSvc) run(ctx context.Context) {
	nextRefreshTime := time.Now()
	logger := logging.GetFromContext(ctx)

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			logger.Info("refreshing exercise trail info")
			count, err := svc.refresh(ctx)

			if err != nil {
				logger.Error("failed to refresh exercise trails", slog.String("err", err.Error()))
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				logger.Info("refreshed exercise trails", slog.Int("count", count))

				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		// TODO: Use blocking channels instead of sleeps
		time.Sleep(1 * time.Second)
	}

	logger.Info("exercise trail service exiting")
}

func (svc *exerciseTrailSvc) refresh(ctx context.Context) (count int, err error) {
	log := logging.GetFromContext(ctx)

	ctx, span := tracer.Start(ctx, "refresh-trails")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, log, ctx)

	trails := []domain.ExerciseTrail{}

	count, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "ExerciseTrail", nil, func(t trailDTO) {

		trail := domain.ExerciseTrail{
			ID:                  t.ID,
			Name:                t.Name,
			Description:         t.Description,
			Categories:          t.Categories(),
			PublicAccess:        t.PublicAccess,
			Location:            *domain.NewLineString(t.Location.Coordinates),
			Length:              math.Round(t.Length*10) / 10,
			Difficulty:          math.Round(t.Difficulty*100) / 100,
			PaymentRequired:     t.PaymentRequired == "yes",
			Status:              t.Status,
			DateLastPreparation: t.DateLastPreparation.Value,
			Source:              t.Source,
			AreaServed:          t.AreaServed,
			SeeAlso:             t.SeeAlso(),
		}

		if len(t.ManagedBy) > 0 {
			trail.ManagedBy, err = svc.orgRegistry.Get(t.ManagedBy)
			if err != nil {
				logger.Error("failed to resolve organisation", slog.String("err", err.Error()))
			}
		}

		if len(t.Owner) > 0 {
			trail.Owner, err = svc.orgRegistry.Get(t.Owner)
			if err != nil {
				logger.Error("failed to resolve organisation", slog.String("err", err.Error()))
			}
		}

		trails = append(trails, trail)
	})

	if err != nil {
		return
	}

	svc.storeExerciseTrailList(trails)

	return
}

func (svc *exerciseTrailSvc) storeExerciseTrailList(list []domain.ExerciseTrail) {
	svc.trailMutex.Lock()
	defer svc.trailMutex.Unlock()

	svc.trails = list
	svc.trailDetails = map[string]int{}

	for index := range list {
		svc.trailDetails[list[index].ID] = index
	}
}

type trailDTO struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Category     json.RawMessage `json:"category"`
	PublicAccess string          `json:"publicAccess"`
	Location     struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates"`
	} `json:"location"`
	Length              float64         `json:"length"`
	Difficulty          float64         `json:"difficulty"`
	PaymentRequired     string          `json:"paymentRequired"`
	Source              string          `json:"source"`
	Status              string          `json:"status"`
	AreaServed          string          `json:"areaServed"`
	DateModified        domain.DateTime `json:"dateModified"`
	DateLastPreparation domain.DateTime `json:"dateLastPreparation"`
	ManagedBy           string          `json:"managedBy"`
	Owner               string          `json:"owner"`
	See                 json.RawMessage `json:"seeAlso"`
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

// SeeAlso extracts the trail reference links as a string array, regardless
// of the format string vs []string of the response property
func (t *trailDTO) SeeAlso() []string {
	return rawJSONToSliceOfStrings(t.See)
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
