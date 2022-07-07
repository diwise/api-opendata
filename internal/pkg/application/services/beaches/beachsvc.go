package beaches

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"sort"
	"strconv"
	"strings"
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

var tracer = otel.Tracer("api-opendata/svcs/beaches")

const (
	DefaultBrokerTenant string = "default"
)

type BeachService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start()
	Shutdown()
}

func NewBeachService(ctx context.Context, logger zerolog.Logger, contextBrokerURL, tenant string) BeachService {
	svc := &beachSvc{
		ctx:              ctx,
		beaches:          []byte("[]"),
		beachDetails:     map[string][]byte{},
		contextBrokerURL: contextBrokerURL,
		tenant:           tenant,
		log:              logger,
		keepRunning:      true,
	}

	return svc
}

type beachSvc struct {
	contextBrokerURL string
	tenant           string

	beachMutex   sync.Mutex
	beaches      []byte
	beachDetails map[string][]byte

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
			err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh beaches")
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

	svc.log.Info().Msg("beach service exiting")
}

func (svc *beachSvc) refresh() error {
	var err error
	ctx, span := tracer.Start(svc.ctx, "refresh-beaches")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	beaches := []domain.Beach{}

	err = svc.getBeachesFromContextBroker(ctx, func(b beachDTO) {
		latitude, longitude := b.LatLon()

		details := domain.BeachDetails{
			ID:          b.ID,
			Name:        b.Name,
			Description: &b.Description,
			Location:    *domain.NewPoint(latitude, longitude),
		}

		seeAlso := b.SeeAlso()
		if len(seeAlso) > 0 {
			details.SeeAlso = &seeAlso
		}

		wqo, err := svc.getWaterQualitiesNearBeach(ctx, latitude, longitude)
		if err != nil {
			logger.Error().Err(err).Msgf("failed to get water qualities near %s (%s)", b.Name, b.ID)
		} else {
			details.WaterQuality = &wqo
		}

		jsonBytes, err := json.MarshalIndent(details, "  ", "  ")
		if err != nil {
			logger.Error().Err(err).Msg("failed to marshal beach to json")
			return
		}

		svc.storeBeachDetails(b.ID, jsonBytes)

		beach := domain.Beach{
			ID:       b.ID,
			Name:     b.Name,
			Location: details.Location,
		}

		if details.WaterQuality != nil && len(*details.WaterQuality) > 0 {
			mostRecentWQ := (*details.WaterQuality)[0]
			if mostRecentWQ.Age() < 24*time.Hour {
				beach.WaterQuality = &mostRecentWQ
			}
		}

		beaches = append(beaches, beach)
	})

	jsonBytes, err := json.MarshalIndent(beaches, "  ", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal beaches to json")
		return err
	}

	svc.storeBeachList(jsonBytes)

	return err
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

func (svc *beachSvc) getBeachesFromContextBroker(ctx context.Context, callback func(b beachDTO)) error {
	var err error

	logger := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.contextBrokerURL+"/ngsi-ld/v1/entities?type=Beach&limit=100&options=keyValues", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Add("Accept", "application/ld+json")
	linkHeaderURL := fmt.Sprintf("<%s>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"", entities.DefaultContextURL)
	req.Header.Add("Link", linkHeaderURL)

	if svc.tenant != DefaultBrokerTenant {
		req.Header.Add("NGSILD-Tenant", svc.tenant)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %s", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
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

	var beaches []beachDTO
	err = json.Unmarshal(respBody, &beaches)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	for _, b := range beaches {
		callback(b)
	}

	return nil
}

func (svc *beachSvc) getWaterQualitiesNearBeach(ctx context.Context, latitude, longitude float64) ([]domain.WaterQuality, error) {
	var err error
	ctx, span := tracer.Start(ctx, "retrieve-water-qualites")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	baseURL := fmt.Sprintf(
		"%s/ngsi-ld/v1/entities?type=WaterQualityObserved&georel=near%%3BmaxDistance==500&geometry=Point&coordinates=[%f,%f]",
		svc.contextBrokerURL, longitude, latitude,
	)

	count, err := func() (int64, error) {
		subctx, subspan := tracer.Start(ctx, "retrieve-wqo-count")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, subspan) }()

		requestURL := fmt.Sprintf("%s&limit=0&count=true", baseURL)

		req, err := http.NewRequestWithContext(subctx, http.MethodGet, requestURL, nil)
		if err != nil {
			err = fmt.Errorf("failed to create request: %s", err.Error())
			return 0, err
		}

		req.Header.Add("Accept", "application/ld+json")
		linkHeaderURL := fmt.Sprintf("<%s>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"", entities.DefaultContextURL)
		req.Header.Add("Link", linkHeaderURL)

		if svc.tenant != DefaultBrokerTenant {
			req.Header.Add("NGSILD-Tenant", svc.tenant)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			err = fmt.Errorf("failed to send request: %s", err.Error())
			return 0, err
		}
		defer resp.Body.Close()

		resultsCount := resp.Header.Get("Ngsild-Results-Count")
		if resultsCount == "" {
			return 0, nil
		}

		count, err := strconv.ParseInt(resultsCount, 10, 64)
		if err != nil {
			err = fmt.Errorf("malformed results header value: %s", err.Error())
		}

		return count, err
	}()

	if count == 0 || err != nil {
		return []domain.WaterQuality{}, err
	}

	const MaxTempCount int64 = 12
	requestURL := baseURL + "&options=keyValues"

	if MaxTempCount < count {
		requestURL = fmt.Sprintf("%s&offset=%d", requestURL, count-MaxTempCount)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		err = fmt.Errorf("failed to create request: %s", err.Error())
		return nil, err
	}

	req.Header.Add("Accept", "application/ld+json")
	linkHeaderURL := fmt.Sprintf("<%s>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"", entities.DefaultContextURL)
	req.Header.Add("Link", linkHeaderURL)

	if svc.tenant != DefaultBrokerTenant {
		req.Header.Add("NGSILD-Tenant", svc.tenant)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to send request: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response body: %s", err.Error())
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		reqbytes, _ := httputil.DumpRequest(req, false)
		respbytes, _ := httputil.DumpResponse(resp, false)

		log := logging.GetFromContext(ctx)
		log.Error().Str("request", string(reqbytes)).Str("response", string(respbytes)).Msg("request failed")
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		contentType := resp.Header.Get("Content-Type")
		err = fmt.Errorf("context source returned status code %d (content-type: %s, body: %s)", resp.StatusCode, contentType, string(respBody))
		return nil, err
	}

	var wqo []struct {
		Temperature  float64         `json:"temperature"`
		DateObserved domain.DateTime `json:"dateObserved"`
	}
	err = json.Unmarshal(respBody, &wqo)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal response: %s", err.Error())
		return nil, err
	}

	// Sort the observations in chronologically reverse order
	sort.Slice(wqo, func(i, j int) bool {
		return strings.Compare(wqo[i].DateObserved.Value, wqo[j].DateObserved.Value) == 1
	})

	waterQualities := make([]domain.WaterQuality, 0, len(wqo))
	previousTime := ""

	for _, observation := range wqo {
		// Filter out all but the first observation with the same timestamp
		if previousTime == observation.DateObserved.Value {
			continue
		}

		waterQualities = append(waterQualities, domain.WaterQuality{
			Temperature:  observation.Temperature,
			DateObserved: observation.DateObserved.Value,
		})

		previousTime = observation.DateObserved.Value
	}

	return waterQualities, nil
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
	DateModified domain.DateTime `json:"dateModified"`
}

func (b *beachDTO) LatLon() (float64, float64) {
	// TODO: A more fancy calculation of midpoint or something?
	return b.Location.Coordinates[0][0][0][1], b.Location.Coordinates[0][0][0][0]
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
