package airquality

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
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

var tracer = otel.Tracer("api-opendata/svcs/airquality")

const (
	DefaultBrokerTenant string = "default"
)

type AirQualityService interface {
	Broker() string
	Tenant() string

	GetAll() []byte
	GetByID(id string) ([]byte, error)

	Start()
	Shutdown()
}

func NewAirQualityService(ctx context.Context, log zerolog.Logger, ctxBrokerURL, ctxBrokerTenant string) AirQualityService {
	return &aqsvc{
		ctx:              ctx,
		aqo:              []byte("[]"),
		aqoDetails:       map[string][]byte{},
		contextBrokerURL: ctxBrokerURL,
		tenant:           ctxBrokerTenant,
		log:              log,
		keepRunning:      true,
	}
}

type aqsvc struct {
	contextBrokerURL string
	tenant           string

	aqoMutex   sync.Mutex
	aqo        []byte
	aqoDetails map[string][]byte

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *aqsvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *aqsvc) Tenant() string {
	return svc.tenant
}

func (svc *aqsvc) GetAll() []byte {
	svc.aqoMutex.Lock()
	defer svc.aqoMutex.Unlock()

	return svc.aqo
}

func (svc *aqsvc) GetByID(id string) ([]byte, error) {
	svc.aqoMutex.Lock()
	defer svc.aqoMutex.Unlock()

	body, ok := svc.aqoDetails[id]
	if !ok {
		return []byte{}, fmt.Errorf("no such air quality")
	}

	return body, nil
}

func (svc *aqsvc) Start() {
	svc.log.Info().Msg("starting air quality service")
	// TODO: Prevent multiple starts on the same service
	go svc.run()
}

func (svc *aqsvc) Shutdown() {
	svc.log.Info().Msg("shutting down air quality service")
	svc.keepRunning = false
}

func (svc *aqsvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing air quality info")
			err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh air quality info")
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

	svc.log.Info().Msg("exercise trail service exiting")
}

func (svc *aqsvc) refresh() error {
	var err error
	ctx, span := tracer.Start(svc.ctx, "refresh-air-quality")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	airqualities := []domain.AirQuality{}

	err = svc.getAirQualitiesFromContextBroker(ctx, func(a airqualityDTO) {

		details := domain.AirQualityDetails{
			ID:                        a.ID,
			Location:                  a.Location,
			DateObserved:              a.DateObserved,
			AtmosphericPressure:       a.AtmosphericPressure,
			Temperature:               a.Temperature,
			RelativeHumidity:          a.RelativeHumidity,
			ParticleCount:             a.ParticleCount,
			PM1:                       a.PM1,
			PM4:                       a.PM4,
			PM10:                      a.PM10,
			PM25:                      a.PM25,
			TotalSuspendedParticulate: a.TotalSuspendedParticulate,
			CO2:                       a.CO2,
			NO:                        a.NO,
			NO2:                       a.NO2,
			NOx:                       a.NOx,
		}

		jsonBytes, err := json.Marshal(details)
		if err != nil {
			logger.Error().Err(err).Msg("failed to marshal air quality to json")
			return
		}

		svc.storeAirQualityDetails(a.ID, jsonBytes)

		aq := domain.AirQuality{
			ID:           a.ID,
			Location:     details.Location,
			DateObserved: details.DateObserved,
		}

		airqualities = append(airqualities, aq)
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve air qualities from context broker")
		return err
	}

	jsonBytes, err := json.Marshal(airqualities)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal air qualities to json")
		return err
	}

	svc.storeAirQualitiesList(jsonBytes)

	return nil
}

func (svc *aqsvc) storeAirQualitiesList(body []byte) {
	svc.aqoMutex.Lock()
	defer svc.aqoMutex.Unlock()

	svc.aqo = body
}

func (svc *aqsvc) storeAirQualityDetails(id string, body []byte) {
	svc.aqoMutex.Lock()
	defer svc.aqoMutex.Unlock()

	svc.aqoDetails[id] = body
}

func (svc *aqsvc) getAirQualitiesFromContextBroker(ctx context.Context, callback func(a airqualityDTO)) error {
	var err error

	logger := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.contextBrokerURL+"/ngsi-ld/v1/entities?type=AirQualityObserved&limit=1000&options=keyValues", nil)
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

	respBody, err := io.ReadAll(resp.Body)
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

	var airQualities []airqualityDTO
	err = json.Unmarshal(respBody, &airQualities)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	for _, a := range airQualities {
		callback(a)
	}

	return nil
}

type airqualityDTO struct {
	ID       string `json:"id"`
	Location struct {
		Type  string       `json:"type"`
		Value domain.Point `json:"value"`
	} `json:"location"`
	DateObserved              domain.DateTime   `json:"dateObserved"`
	AtmosphericPressure       *domain.Pollutant `json:"atmosphericPressure"`
	Temperature               *domain.Pollutant `json:"temperature"`
	RelativeHumidity          *domain.Pollutant `json:"relativeHumidity"`
	ParticleCount             *domain.Pollutant `json:"particleCount"`
	PM1                       *domain.Pollutant `json:"PM1"`
	PM4                       *domain.Pollutant `json:"PM4"`
	PM10                      *domain.Pollutant `json:"PM10"`
	PM25                      *domain.Pollutant `json:"PM25"`
	TotalSuspendedParticulate *domain.Pollutant `json:"totalSuspendedParticulate"`
	CO2                       *domain.Pollutant `json:"CO2"`
	NO                        *domain.Pollutant `json:"NO"`
	NO2                       *domain.Pollutant `json:"NO2"`
	NOx                       *domain.Pollutant `json:"NOx"`
	Voltage                   *domain.Pollutant `json:"voltage"`
}
