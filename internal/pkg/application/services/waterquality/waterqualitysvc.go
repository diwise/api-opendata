package waterquality

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type WaterQualityService interface {
	Start()
	Shutdown()

	Tenant() string
	Broker() string

	BetweenTimes(from, to time.Time)
	Distance(distance int)
	Location(latitude, longitude float64)

	GetAll() []WaterQualityTemporal
	GetAllNearPoint(latitude, longitude float64, distance int) (*[]WaterQualityTemporal, error)
}

func NewWaterQualityService(ctx context.Context, log zerolog.Logger, url, tenant string) WaterQualityService {
	return &wqsvc{
		contextBrokerURL: url,
		tenant:           tenant,

		waterQualities: []WaterQualityTemporal{},
		keepRunning:    true,

		ctx: ctx,
		log: log,
	}
}

type wqsvc struct {
	contextBrokerURL string
	tenant           string

	from      time.Time
	to        time.Time
	latitude  float64
	longitude float64
	distance  int

	wqoMutex       sync.Mutex
	waterQualities []WaterQualityTemporal

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *wqsvc) BetweenTimes(from, to time.Time) {
	svc.from = from
	svc.to = to
}

func (svc *wqsvc) Distance(distance int) {
	svc.distance = distance
}

func (svc *wqsvc) Location(latitude, longitude float64) {
	svc.latitude = latitude
	svc.longitude = longitude
}

func (svc *wqsvc) Start() {
	svc.log.Info().Msg("starting water quality service")
	go svc.run()
}

func (svc *wqsvc) Shutdown() {
	svc.log.Info().Msg("shutting down water quality service")
	svc.keepRunning = false
}

func (svc *wqsvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *wqsvc) Tenant() string {
	return svc.tenant
}

func (svc *wqsvc) GetAll() []WaterQualityTemporal {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	return svc.waterQualities
}

func (svc *wqsvc) GetAllNearPoint(latitude, longitude float64, maxDistance int) (*[]WaterQualityTemporal, error) {
	waterQualitiesWithinDistance := []WaterQualityTemporal{}

	for _, storedWQ := range svc.waterQualities {
		wqPoint := NewCoords(storedWQ.Location.Value.Coordinates[0], storedWQ.Location.Value.Coordinates[1]) // double check n
		comparativeLocation := NewCoords(latitude, longitude)
		_, distanceBetweenPoints := Distance(wqPoint, comparativeLocation)

		if distanceBetweenPoints < maxDistance {
			waterQualitiesWithinDistance = append(waterQualitiesWithinDistance, storedWQ)
		}
	}
	if len(waterQualitiesWithinDistance) == 0 {
		return &[]WaterQualityTemporal{}, fmt.Errorf("no stored water qualities exist within %d meters of point %f,%f", maxDistance, longitude, latitude)
	}

	return &waterQualitiesWithinDistance, nil
}

type Coords struct {
	Latitude  float64
	Longitude float64
}

func NewCoords(lat, lon float64) Coords {
	return Coords{
		Latitude:  lat,
		Longitude: lon,
	}
}

func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

func Distance(point1, point2 Coords) (int, int) {
	earthRadiusKm := 6371

	lat1 := degreesToRadians(point1.Latitude)
	lon1 := degreesToRadians(point1.Longitude)
	lat2 := degreesToRadians(point2.Latitude)
	lon2 := degreesToRadians(point2.Longitude)

	diffLat := lat2 - lat1
	diffLon := lon2 - lon1

	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*
		math.Pow(math.Sin(diffLon/2), 2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distanceInKm := c * float64(earthRadiusKm)
	distanceInM := distanceInKm * 1000

	return int(distanceInKm), int(distanceInM)
}

func (svc *wqsvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing water quality info")
			err := svc.refresh()
			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh water quality info")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				svc.log.Info().Msgf("refreshed water qualities")
				// Refresh every 5 minutes of success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		time.Sleep(1 * time.Second)
	}
	svc.log.Info().Msg("water quality service exiting")
}

func (svc *wqsvc) refresh() error {
	wqoBytes, err := svc.requestData(svc.ctx, svc.log, svc.contextBrokerURL)
	if err != nil {
		svc.log.Error().Err(err).Msg("failed to retrieve water quality data")
		return nil
	}

	wqos := []WaterQualityTemporal{}
	err = json.Unmarshal(wqoBytes, &wqos)
	if err != nil {
		svc.log.Error().Err(err).Msg("failed to unmarshal water qualities")
		return err
	}

	svc.storeWaterQualityList(wqos)

	return nil
}

func (svc *wqsvc) storeWaterQualityList(wqs []WaterQualityTemporal) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	svc.waterQualities = wqs
}

func (q *wqsvc) requestData(ctx context.Context, log zerolog.Logger, ctxBrokerURL string) ([]byte, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	url := fmt.Sprintf(
		"%s/ngsi-ld/v1/temporal/entities?type=WaterQualityObserved",
		ctxBrokerURL,
	)

	if !q.from.IsZero() && !q.to.IsZero() {
		url = fmt.Sprintf("%s&timerel=between&time=%s&endTime=%s", url, q.from.Format(time.RFC3339), q.to.Format(time.RFC3339))
	} else {
		q.from = time.Now().UTC().Add(-24 * time.Hour)
		q.to = time.Now().UTC()
		url = fmt.Sprintf("%s&timerel=between&time=%s&endTime=%s", url, q.from.Format(time.RFC3339), q.to.Format(time.RFC3339))
	}

	if q.latitude > 0.0 && q.longitude > 0.0 {
		url = fmt.Sprintf("%s&georel=near%%3BmaxDistance==%d&geometry=Point&coordinates=[%f,%f]", url, q.distance, q.latitude, q.longitude)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed, status code not ok: %d", response.StatusCode)
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err)
	}

	return b, nil
}

type WaterQualityTemporal struct {
	ID       string `json:"id"`
	Location struct {
		Type  string       `json:"type"`
		Value domain.Point `json:"value"`
	} `json:"location"`
	Temperature  []domain.Value `json:"temperature"`
	Source       string         `json:"source,omitempty"`
	DateObserved struct {
		Type            string `json:"type"`
		domain.DateTime `json:"value"`
	} `json:"dateObserved,omitempty"`
}
