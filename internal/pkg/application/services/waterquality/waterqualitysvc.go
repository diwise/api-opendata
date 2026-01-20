package waterquality

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	contextbroker "github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/waterquality")

type WaterQualityService interface {
	Start(ctx context.Context)
	Refresh(ctx context.Context) (int, error)
	Shutdown(ctx context.Context)

	Tenant() string
	Broker() string

	GetAll(ctx context.Context) []domain.WaterQuality
	GetAllNearPointWithinTimespan(ctx context.Context, pt Point, distance int, from, to time.Time) ([]domain.WaterQuality, error)
	GetByID(ctx context.Context, id string, from, to time.Time) (*domain.WaterQualityTemporal, error)
}

func NewWaterQualityService(ctx context.Context, url, tenant string) WaterQualityService {
	return &wqsvc{
		contextBrokerURL: url,
		tenant:           tenant,

		waterQualityByID: map[string]WaterQuality{},

		queue:       make(chan func()),
		keepRunning: &atomic.Bool{},
	}
}

var ErrWQNotFound error = errors.New("not found")

type wqsvc struct {
	contextBrokerURL string
	tenant           string

	waterQualityByID map[string]WaterQuality

	queue chan func()

	keepRunning *atomic.Bool
	wg          sync.WaitGroup
}

func (svc *wqsvc) Start(ctx context.Context) {
	// use atomic swap to avoid startup races
	alreadyStarted := svc.keepRunning.Swap(true)
	if alreadyStarted {
		logger := logging.GetFromContext(ctx)
		logger.Error("attempt to start the water quality service multiple times")
		return
	}

	go svc.run(ctx)
}

func (svc *wqsvc) Refresh(ctx context.Context) (int, error) {

	refreshDone := make(chan int)
	refreshFailed := make(chan error)

	svc.queue <- func() {
		count, err := svc.refresh(ctx)
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

func (svc *wqsvc) Shutdown(ctx context.Context) {
	if svc.keepRunning.Load() {
		svc.queue <- func() {
			svc.keepRunning.Store(false)
		}

		svc.wg.Wait()
	}
}

func (svc *wqsvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *wqsvc) Tenant() string {
	return svc.tenant
}

func (svc *wqsvc) GetAll(ctx context.Context) []domain.WaterQuality {
	result := make(chan []domain.WaterQuality)

	svc.queue <- func() {
		l := []domain.WaterQuality{}

		for _, i := range svc.waterQualityByID {
			l = append(l, i.Latest)
		}

		result <- l
	}

	return <-result
}

func (svc *wqsvc) GetAllNearPointWithinTimespan(ctx context.Context, pt Point, maxDistance int, from, to time.Time) ([]domain.WaterQuality, error) {
	result := make(chan []domain.WaterQuality)
	failure := make(chan error)

	between := func(t, from, to time.Time) bool {
		if from.IsZero() && to.IsZero() {
			return true
		}

		if !from.IsZero() && t.Before(from) {
			return false
		}

		if !to.IsZero() && t.After(to) {
			return false
		}

		return true
	}

	svc.queue <- func() {
		waterQualitiesWithinDistance := make([]domain.WaterQuality, 0, len(svc.waterQualityByID))

		for _, storedWQ := range svc.waterQualityByID {
			wqPoint := NewPoint(storedWQ.Location.Coordinates[1], storedWQ.Location.Coordinates[0])
			distanceBetweenPoints := distance(wqPoint, pt)

			storedDate, err := time.ParseInLocation(time.RFC3339, storedWQ.Latest.DateObserved, time.UTC)
			if err != nil {
				failure <- fmt.Errorf("failed to parse time from stored water quality observed: %s", err.Error())
				return
			}

			if distanceBetweenPoints < maxDistance {
				// check if latest observation is within time range
				if between(storedDate, from, to) {
					waterQualitiesWithinDistance = append(waterQualitiesWithinDistance, storedWQ.Latest)
					continue
				}

				// check historical observations if latest is not within time range. Stop at first match.
				for _, temp := range *storedWQ.History {
					tempDate, err := time.ParseInLocation(time.RFC3339, temp.ObservedAt, time.UTC)
					if err != nil {
						failure <- fmt.Errorf("failed to parse time from stored water quality history: %s", err.Error())
						return
					}

					if between(tempDate, from, to) {
						waterQualitiesWithinDistance = append(waterQualitiesWithinDistance, domain.WaterQuality{
							ID:           storedWQ.ID,
							Temperature:  temp.Value,
							DateObserved: temp.ObservedAt,
							Source:       storedWQ.Latest.Source,
							Location:     storedWQ.Location,
						})
						break
					}
				}
			}

		}

		result <- waterQualitiesWithinDistance
	}

	return <-result, nil
}

func (svc *wqsvc) GetByID(ctx context.Context, id string, from, to time.Time) (*domain.WaterQualityTemporal, error) {

	result := make(chan *domain.WaterQualityTemporal)
	failure := make(chan error)

	svc.queue <- func() {
		_, ok := svc.waterQualityByID[id]
		if !ok {
			failure <- ErrWQNotFound
			return
		}

		if from.IsZero() && to.IsZero() {
			to = time.Now().UTC()
			from = to.Add(-24 * time.Hour)
		}

		wqo := svc.waterQualityByID[id]

		wqoTemp := domain.WaterQualityTemporal{
			ID: wqo.ID,
		}

		if wqo.Latest.Source != nil {
			wqoTemp.Source = *wqo.Latest.Source
		}

		if wqo.Location != nil {
			wqoTemp.Location = wqo.Location
		}

		if wqo.History != nil {
			temps := *wqo.History

			if len(temps) != 0 {
				sort.Slice(temps, func(i, j int) bool {
					return strings.Compare(temps[i].ObservedAt, temps[j].ObservedAt) > 0
				})

				wqoTemp.Temperature = temps
			}
		}

		result <- &wqoTemp
	}

	select {
	case err := <-failure:
		return nil, err
	case r := <-result:
		return r, nil
	}
}

type Point struct {
	Latitude  float64
	Longitude float64
}

func NewPoint(lat, lon float64) Point {
	return Point{
		Latitude:  lat,
		Longitude: lon,
	}
}

func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

func distance(point1, point2 Point) int {
	const earthRadiusM = 6371000.0

	lat1 := degreesToRadians(point1.Latitude)
	lon1 := degreesToRadians(point1.Longitude)
	lat2 := degreesToRadians(point2.Latitude)
	lon2 := degreesToRadians(point2.Longitude)

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	// Clamp pga flyttalsavrundning
	if a < 0 {
		a = 0
	} else if a > 1 {
		a = 1
	}

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return int(math.Round(earthRadiusM * c))
}

func (svc *wqsvc) run(ctx context.Context) {
	svc.wg.Add(1)
	defer svc.wg.Done()

	logger := logging.GetFromContext(ctx)
	logger.Info("starting water quality service")

	const RefreshIntervalOnFail time.Duration = 5 * time.Second
	const RefreshIntervalOnSuccess time.Duration = 30 * time.Second

	var refreshTimer *time.Timer
	count, err := svc.refresh(ctx)

	if err != nil {
		logger.Error("failed to refresh water qualities", slog.String("err", err.Error()))
		refreshTimer = time.NewTimer(RefreshIntervalOnFail)
	} else {
		logger.Info("refreshed water quality", slog.Int("count", count))
		refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
	}

	for svc.keepRunning.Load() {
		select {
		case fn := <-svc.queue:
			fn()
		case <-refreshTimer.C:
			count, err := svc.refresh(ctx)
			if err != nil {
				logger.Error("failed to refresh water quality info", slog.String("err", err.Error()))
				refreshTimer = time.NewTimer(RefreshIntervalOnFail)
			} else {
				logger.Info("refreshed water quality entities", slog.Int("count", count))
				refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
			}
		}
	}

	logger.Info("water quality service exiting")
}

func (svc *wqsvc) refresh(ctx context.Context) (count int, err error) {

	ctx, span := tracer.Start(ctx, "refresh-water-quality")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	logger := logging.GetFromContext(ctx)
	_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	logger.Info("refreshing water quality info")

	after := func(d1, d2 string) bool {
		dt1, err := time.Parse(time.RFC3339, d1)
		if err != nil {
			return false
		}

		if d2 == "" {
			return true
		}

		dt2, err := time.Parse(time.RFC3339, d2)
		if err != nil {
			return false
		}

		return dt1.After(dt2)
	}

	round := func(f float64) float64 {
		return math.Round(f*10) / 10
	}

	_, err = contextbroker.QueryEntities(ctx, svc.Broker(), svc.Tenant(), "WaterQualityObserved", nil, func(w WaterQualityDTO) {
		wq := WaterQuality{
			ID: w.ID,
		}

		latest := domain.WaterQuality{
			ID:           wq.ID,
			Temperature:  round(w.Temperature),
			DateObserved: w.DateObserved.Value,
		}

		if w.Source != nil {
			latest.Source = w.Source
		}

		if w.Location != nil {
			wq.Location = domain.NewPoint(w.Location.Coordinates[1], w.Location.Coordinates[0])
			latest.Location = wq.Location
		}

		if after(latest.DateObserved, wq.Latest.DateObserved) {
			wq.Latest = latest
		}

		dto := WaterQualityTemporalDTO{}

		b, err := svc.requestTemporalDataForSingleEntity(ctx, svc.Broker(), w.ID, svc.Tenant(), time.Time{}, time.Time{})
		if err != nil {
			logger.Error("no temporal data found for water quality", "id", wq.ID, "err", err.Error())
			return
		}

		json.Unmarshal(b, &dto)

		temps := []domain.Value{}

		if len(dto.Temperature) != 0 {
			for i, t := range dto.Temperature {
				dto.Temperature[i].Value = round(t.Value)
			}

			temps = append(temps, dto.Temperature...)
		} else {
			logger.Info("no temporal data found for water quality", "id", wq.ID)
		}

		wq.History = &temps

		svc.waterQualityByID[w.ID] = wq

	})

	if err != nil {
		err = fmt.Errorf("failed to retrieve water qualities from context broker: %w", err)
		return
	}

	return len(svc.waterQualityByID), nil
}

func (q *wqsvc) requestTemporalDataForSingleEntity(ctx context.Context, ctxBrokerURL, id, tenant string, from, to time.Time) ([]byte, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	params := url.Values{}

	if from.IsZero() && to.IsZero() {
		from = time.Now().UTC().Add(-24 * time.Hour)
		to = time.Now().UTC()
	}

	params.Add("timerel", "between")
	params.Add("timeAt", from.Format(time.RFC3339))
	params.Add("endTimeAt", to.Format(time.RFC3339))
	params.Add("lastN", "50")

	requestURL := fmt.Sprintf(
		"%s/ngsi-ld/v1/temporal/entities/%s?%s",
		ctxBrokerURL, id, params.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %s", err.Error())
	}

	req.Header.Add("Accept", "application/ld+json")
	req.Header.Add("Link", entities.LinkHeader)

	if tenant != entities.DefaultNGSITenant {
		req.Header.Add("NGSILD-Tenant", tenant)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %s", err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		rb, _ := httputil.DumpResponse(response, true)
		logging.GetFromContext(ctx).Debug("bad response", "response", string(rb))
		return nil, fmt.Errorf("request failed, status code %d not ok (or partial)", response.StatusCode)
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	return b, nil
}

type WaterQualityTemporalDTO struct {
	ID           string          `json:"id"`
	Temperature  []domain.Value  `json:"temperature"`
	Source       *string         `json:"source,omitempty"`
	DateObserved domain.DateTime `json:"dateObserved,omitempty"`
}

type WaterQualityDTO struct {
	ID           string          `json:"id"`
	Location     *domain.Point   `json:"location,omitempty"`
	Temperature  float64         `json:"temperature"`
	Source       *string         `json:"source,omitempty"`
	DateObserved domain.DateTime `json:"dateObserved,omitempty"`
}

type WaterQuality struct {
	ID       string              `json:"id"`
	Location *domain.Point       `json:"location"`
	Latest   domain.WaterQuality `json:"latest"`
	History  *[]domain.Value     `json:"history"`
}
