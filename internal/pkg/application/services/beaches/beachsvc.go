package beaches

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/beaches")

type BeachService interface {
	Broker() string
	Tenant() string

	GetAll(ctx context.Context) []Beach
	GetByID(ctx context.Context, id string) (*BeachDetails, error)

	Start(context.Context)
	Refresh(context.Context) (int, error)
	Shutdown(context.Context)
}

func NewBeachService(ctx context.Context, contextBrokerURL, tenant string, maxWQODistance int, wqsvc waterquality.WaterQualityService) BeachService {
	svc := &beachSvc{
		wqsvc:               wqsvc,
		beaches:             []Beach{},
		beachDetails:        map[string]BeachDetails{},
		beachMaxWQODistance: maxWQODistance,
		contextBrokerURL:    contextBrokerURL,
		tenant:              tenant,
		queue:               make(chan func()),
		keepRunning:         &atomic.Bool{},
	}

	return svc
}

type beachSvc struct {
	wqsvc waterquality.WaterQualityService

	contextBrokerURL string
	tenant           string

	beaches             []Beach
	beachDetails        map[string]BeachDetails
	beachMaxWQODistance int

	queue chan func()

	keepRunning *atomic.Bool
	wg          sync.WaitGroup
}

func (svc *beachSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *beachSvc) Tenant() string {
	return svc.tenant
}

func (svc *beachSvc) GetAll(ctx context.Context) []Beach {
	result := make(chan []Beach)

	svc.queue <- func() {
		result <- svc.beaches
	}

	return <-result
}

func (svc *beachSvc) GetByID(ctx context.Context, beachID string) (*BeachDetails, error) {
	result := make(chan BeachDetails)
	err := make(chan error)

	svc.queue <- func() {
		body, ok := svc.beachDetails[beachID]
		if !ok {
			err <- fmt.Errorf("no such beach")
		} else {
			result <- body
		}
	}

	select {
	case r := <-result:
		return &r, nil
	case e := <-err:
		return nil, e
	}
}

func (svc *beachSvc) Start(ctx context.Context) {
	go svc.run(ctx)
}

func (svc *beachSvc) Refresh(ctx context.Context) (int, error) {
	logger := logging.GetFromContext(ctx)

	refreshDone := make(chan int)
	refreshFailed := make(chan error)

	svc.queue <- func() {
		count, err := svc.refresh(ctx, logger)
		if err != nil {
			refreshFailed <- err
		} else {
			refreshDone <- count
		}
	}

	select {
	case c := <-refreshDone:
		return c, nil
	case e := <-refreshFailed:
		return 0, e
	}
}

func (svc *beachSvc) Shutdown(ctx context.Context) {
	svc.queue <- func() {
		svc.keepRunning.Store(false)
	}

	svc.wg.Wait()
}

func (svc *beachSvc) run(ctx context.Context) {
	svc.wg.Add(1)
	defer svc.wg.Done()

	logger := logging.GetFromContext(ctx)
	logger.Info().Msg("starting up beach service")

	// use atomic swap to avoid startup races
	alreadyStarted := svc.keepRunning.Swap(true)
	if alreadyStarted {
		logger.Error().Msg("attempt to start the beach service multiple times")
		return
	}

	const RefreshIntervalOnFail time.Duration = 5 * time.Second
	const RefreshIntervalOnSuccess time.Duration = 5 * time.Minute

	var refreshTimer *time.Timer
	count, err := svc.refresh(ctx, logger)

	if err != nil {
		logger.Error().Err(err).Msg("failed to refresh beaches")
		refreshTimer = time.NewTimer(RefreshIntervalOnFail)
	} else {
		logger.Info().Msgf("refreshed %d beaches", count)
		refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
	}

	for svc.keepRunning.Load() {
		select {
		case fn := <-svc.queue:
			fn()
		case <-refreshTimer.C:
			count, err := svc.refresh(ctx, logger)
			if err != nil {
				logger.Error().Err(err).Msg("failed to refresh beaches")
				refreshTimer = time.NewTimer(RefreshIntervalOnFail)
			} else {
				logger.Info().Msgf("refreshed %d beaches", count)
				refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
			}
		}
	}

	logger.Info().Msg("beach service exiting")
}

func (svc *beachSvc) refresh(ctx context.Context, log zerolog.Logger) (count int, err error) {

	ctx, span := tracer.Start(ctx, "refresh-beaches")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, log, ctx)

	logger.Info().Msg("refreshing beach info")

	beaches := []Beach{}

	_, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "Beach", nil, func(b beachDTO) {
		latitude, longitude := b.LatLon()

		details := BeachDetails{
			ID:          b.ID,
			Name:        b.Name,
			Description: &b.Description,
			Location:    *domain.NewPoint(latitude, longitude),
		}

		seeAlso := b.SeeAlso()
		if len(seeAlso) > 0 {
			details.SeeAlso = &seeAlso
		}

		pt := waterquality.NewPoint(latitude, longitude)
		wqots, err_ := svc.wqsvc.GetAllNearPoint(ctx, pt, svc.beachMaxWQODistance)
		if err_ != nil {
			logger.Error().Err(err_).Msgf("failed to get water qualities near %s (%s)", b.Name, b.ID)
		} else {
			wq := []WaterQuality{}

			for _, t := range wqots {
				newWQ := WaterQuality{}

				if t.Temperature > 0 {
					newWQ.Temperature = t.Temperature
				}

				if t.Source != nil {
					newWQ.Source = t.Source
				}

				if t.DateObserved != "" {
					newWQ.DateObserved = t.DateObserved
				}

				wq = append(wq, newWQ)
			}

			details.WaterQuality = &wq
		}

		svc.beachDetails[b.ID] = details

		beach := Beach{
			ID:       b.ID,
			Name:     b.Name,
			Location: details.Location,
		}

		if details.WaterQuality != nil && len(*details.WaterQuality) > 0 {
			mostRecentWQ := (*details.WaterQuality)[0]
			if mostRecentWQ.Age() < 24*time.Hour {
				beach.WaterQuality = &WaterQuality{
					Temperature:  mostRecentWQ.Temperature,
					DateObserved: mostRecentWQ.DateObserved,
					Source:       mostRecentWQ.Source,
				}
			}
		}

		beaches = append(beaches, beach)
	})
	if err != nil {
		err = fmt.Errorf("failed to retrieve beaches from context broker: %w", err)
		return
	}

	svc.beaches = beaches

	return
}

type beachDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    struct {
		Type        string          `json:"type"`
		Coordinates [][][][]float64 `json:"coordinates"`
	} `json:"location"`
	See          json.RawMessage `json:"seeAlso"`
	DateModified json.RawMessage `json:"dateModified"`
}

func round(v float64) float64 {
	return math.Round(v*1000000) / 1000000
}

func (b *beachDTO) LatLon() (float64, float64) {
	latSum := 0.0
	lonSum := 0.0

	for idx, pair := range b.Location.Coordinates[0][0] {
		if idx > 0 {
			lonSum = lonSum + pair[0]
			latSum = latSum + pair[1]
		}
	}

	numPairs := len(b.Location.Coordinates[0][0])
	return round(latSum / (float64(numPairs - 1))), round(lonSum / (float64(numPairs - 1)))
}

func (b *beachDTO) SeeAlso() []string {
	refsAsArray := []string{}

	if len(b.See) > 0 {
		if err := json.Unmarshal(b.See, &refsAsArray); err != nil {
			var refsAsString string

			if err = json.Unmarshal(b.See, &refsAsString); err != nil {
				return []string{err.Error()}
			}

			return []string{refsAsString}
		}
	}

	return refsAsArray
}

type WaterQuality struct {
	Temperature  float64 `json:"temperature"`
	DateObserved string  `json:"dateObserved"`
	Source       *string `json:"source,omitempty"`
}

func (w WaterQuality) Age() time.Duration {
	observedAt, err := time.Parse(time.RFC3339, w.DateObserved)
	if err != nil {
		// Pretend it was almost 100 years ago
		return 100 * 365 * 24 * time.Hour
	}

	return time.Since(observedAt)
}

type Beach struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Location     domain.Point  `json:"location"`
	WaterQuality *WaterQuality `json:"waterquality,omitempty"`
}

type BeachDetails struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	Location     domain.Point    `json:"location"`
	WaterQuality *[]WaterQuality `json:"waterquality"`
	SeeAlso      *[]string       `json:"seeAlso,omitempty"`
}
