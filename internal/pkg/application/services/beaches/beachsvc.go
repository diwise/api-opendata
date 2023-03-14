package beaches

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/beaches")

type BeachService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start()
	Shutdown()
}

func NewBeachService(ctx context.Context, logger zerolog.Logger, contextBrokerURL, tenant string, maxWQODistance int, wqsvc waterquality.WaterQualityService) BeachService {
	svc := &beachSvc{
		wqsvc:               wqsvc,
		ctx:                 ctx,
		beaches:             []byte("[]"),
		beachDetails:        map[string][]byte{},
		beachMaxWQODistance: maxWQODistance,
		contextBrokerURL:    contextBrokerURL,
		tenant:              tenant,
		log:                 logger,
		keepRunning:         true,
	}

	return svc
}

type beachSvc struct {
	wqsvc waterquality.WaterQualityService

	contextBrokerURL string
	tenant           string

	beachMutex          sync.Mutex
	beaches             []byte
	beachDetails        map[string][]byte
	beachMaxWQODistance int

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *beachSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *beachSvc) Tenant() string {
	return svc.tenant
}

func (svc *beachSvc) GetAll() []byte {
	svc.beachMutex.Lock()
	defer svc.beachMutex.Unlock()

	return svc.beaches
}

func (svc *beachSvc) GetByID(id string) ([]byte, error) {
	svc.beachMutex.Lock()
	defer svc.beachMutex.Unlock()

	body, ok := svc.beachDetails[id]
	if !ok {
		return []byte{}, fmt.Errorf("no such beach")
	}

	return body, nil
}

func (svc *beachSvc) Start() {
	svc.log.Info().Msg("starting beach service")
	// TODO: Prevent multiple starts on the same service
	go svc.run()
}

func (svc *beachSvc) Shutdown() {
	svc.log.Info().Msg("shutting down beach service")
	svc.keepRunning = false
}

func (svc *beachSvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing beach info")
			count, err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh beaches")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				svc.log.Info().Msgf("refreshed %d beaches", count)
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		// TODO: Use blocking channels instead of sleeps
		time.Sleep(1 * time.Second)
	}

	svc.log.Info().Msg("beach service exiting")
}

func (svc *beachSvc) refresh() (count int, err error) {

	ctx, span := tracer.Start(svc.ctx, "refresh-beaches")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

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
		wqots, err_ := svc.wqsvc.GetAllNearPoint(pt, svc.beachMaxWQODistance)
		if err_ != nil {
			logger.Error().Err(err_).Msgf("failed to get water qualities near %s (%s)", b.Name, b.ID)
		} else {
			wq := []WaterQuality{}

			for _, t := range *wqots {
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

		jsonBytes, err_ := json.Marshal(details)
		if err_ != nil {
			err = fmt.Errorf("failed to marshal beach to json: %w", err_)
			return
		}

		svc.storeBeachDetails(b.ID, jsonBytes)

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

	jsonBytes, err_ := json.Marshal(beaches)
	if err_ != nil {
		err = fmt.Errorf("failed to marshal beaches to json: %w", err_)
		return
	}

	svc.storeBeachList(jsonBytes)

	return
}

func (svc *beachSvc) storeBeachDetails(id string, body []byte) {
	svc.beachMutex.Lock()
	defer svc.beachMutex.Unlock()

	svc.beachDetails[id] = body
}

func (svc *beachSvc) storeBeachList(body []byte) {
	svc.beachMutex.Lock()
	defer svc.beachMutex.Unlock()

	svc.beaches = body
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
	WaterQuality *WaterQuality `json:"waterquality"`
}

type BeachDetails struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	Location     domain.Point    `json:"location"`
	WaterQuality *[]WaterQuality `json:"waterquality"`
	SeeAlso      *[]string       `json:"seeAlso,omitempty"`
}
