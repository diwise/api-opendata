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

	beaches := []domain.Beach{}

	_, err = contextbroker.QueryEntities(ctx, svc.contextBrokerURL, svc.tenant, "Beach", nil, func(b beachDTO) {
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

		wqots, err_ := svc.wqsvc.GetAllNearPoint(latitude, longitude, svc.beachMaxWQODistance)
		if err_ != nil {
			logger.Error().Err(err_).Msgf("failed to get water qualities near %s (%s)", b.Name, b.ID)
		} else {
			wq := []domain.WaterQualityTemporal{}

			for _, t := range *wqots {
				newWQ := domain.WaterQualityTemporal{}

				if len(t.Temperature) > 0 {
					newWQ.Temperature = t.Temperature
				}

				if t.Source != "" {
					newWQ.Source = t.Source
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

		beach := domain.Beach{
			ID:       b.ID,
			Name:     b.Name,
			Location: details.Location,
		}

		// This is probably not a great solution to check how old the reading actually is
		// Saying this almost entirely based on the fact that now have an array of temperatures
		// and I'm only grabbing the first entry in the array and using those values. I don't actually
		// know for a fact that that is the most recent entry.
		if details.WaterQuality != nil && len(*details.WaterQuality) > 0 {
			mostRecentWQ := (*details.WaterQuality)[0]
			if mostRecentWQ.Temperature[0].Age() < 24*time.Hour {
				beach.WaterQuality = &domain.WaterQuality{
					Temperature:  mostRecentWQ.Temperature[0].Value,
					DateObserved: mostRecentWQ.Temperature[0].ObservedAt,
					Source:       &mostRecentWQ.Source,
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

/* func (svc *beachSvc) getWaterQualitiesNearBeach(ctx context.Context, latitude, longitude float64) ([]domain.WaterQuality, error) {
	var err error
	ctx, span := tracer.Start(ctx, "retrieve-water-qualites")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	startTime := time.Now().UTC().Add(-24 * time.Hour)
	endTime := time.Now().UTC()

	baseURL := fmt.Sprintf(
		"%s/ngsi-ld/v1/temporal/entities/?time=%s&endTime=%s&timerel=between&type=WaterQualityObserved&georel=near%%3BmaxDistance==%d&geometry=Point&coordinates=[%f,%f]",
		svc.contextBrokerURL, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), svc.beachMaxWQODistance, longitude, latitude,
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

		if svc.tenant != entities.DefaultNGSITenant {
			req.Header.Add("NGSILD-Tenant", svc.tenant)
		}

		svc.log.Debug().Msgf("calling %s", requestURL)

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

	if svc.tenant != entities.DefaultNGSITenant {
		req.Header.Add("NGSILD-Tenant", svc.tenant)
	}

	svc.log.Debug().Msgf("calling %s", requestURL)

	resp, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to send request: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
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
		Source       *string         `json:"source"`
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
			Temperature:  math.Round(observation.Temperature*10) / 10,
			DateObserved: observation.DateObserved.Value,
			Source:       observation.Source,
		})

		previousTime = observation.DateObserved.Value
	}

	return waterQualities, nil
} */

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
