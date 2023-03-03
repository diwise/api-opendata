package waterquality

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type WaterQuality interface {
	Start()
	Shutdown()

	Tenant() string
	Broker() string

	GetAll() []byte
}

type WaterQualityQuery interface {
	BetweenTimes(from, to time.Time) WaterQualityQuery
	Point(latitude, longitude float64) WaterQualityQuery
}

func NewWaterQualityService(ctx context.Context, log zerolog.Logger, url, tenant string) WaterQuality {
	return &wqsvc{
		contextBrokerURL: url,
		tenant:           tenant,

		waterQualities:      []byte("[]"),
		waterQualityDetails: map[string][]byte{},

		ctx: ctx,
		log: log,
	}
}

type wqq struct {
	from      time.Time
	to        time.Time
	latitude  float64
	longitude float64
	distance  int
}

type wqsvc struct {
	wqq
	contextBrokerURL string
	tenant           string

	wqoMutex            sync.Mutex
	waterQualities      []byte
	waterQualityDetails map[string][]byte

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
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

func (svc *wqsvc) GetAll() []byte {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	return svc.waterQualities
}

func (svc *wqsvc) GetByID(id string) ([]byte, error) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	body, ok := svc.waterQualityDetails[id]
	if !ok {
		return []byte{}, fmt.Errorf("no such water quality observed")
	}

	return body, nil
}

func (svc *wqsvc) GetByLocation(latitude, longitude float64) ([]byte, error) {

	//ask for suggestions on how best to do this

	return nil, nil
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

	for _, wq := range wqos {
		details := WaterQualityTemporal{
			ID:          wq.ID,
			Location:    wq.Location,
			Temperature: wq.Temperature,
		}

		jsonBytes, err := json.Marshal(details)
		if err != nil {
			svc.log.Error().Err(err).Msg("failed to marshal water quality to json")
			return err
		}

		svc.storeWaterQualityDetails(wq.ID, jsonBytes)
	}

	jsonBytes, err := json.Marshal(wqos)
	if err != nil {
		svc.log.Error().Err(err).Msg("failed to marshal water qualities to json")
		return err
	}

	svc.storeWaterQualityList(jsonBytes)

	return nil
}

func (svc *wqsvc) storeWaterQualityDetails(id string, body []byte) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	svc.waterQualityDetails[id] = body
}

func (svc *wqsvc) storeWaterQualityList(body []byte) {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	svc.waterQualities = body
}

func (q *wqq) requestData(ctx context.Context, log zerolog.Logger, ctxBrokerURL string) ([]byte, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	url := fmt.Sprintf(
		"%s/ngsi-ld/v1/temporal/entities?type=WaterQualityObserved",
		ctxBrokerURL,
	)

	if !q.from.IsZero() && !q.to.IsZero() {
		url = url + fmt.Sprintf("&timerel=between&time=%s&endTime=%s", q.from.Format(time.RFC3339), q.to.Format(time.RFC3339))
	} else {
		q.from = time.Now().UTC().Add(-24 * time.Hour)
		q.to = time.Now().UTC()
		url = url + fmt.Sprintf("&timerel=between&time=%s&endTime=%s", q.from.Format(time.RFC3339), q.to.Format(time.RFC3339))
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
	Temperature []Value `json:"temperature"`
}

type Value struct {
	Value      float64 `json:"value"`
	ObservedAt string  `json:"observedAt"`
}
