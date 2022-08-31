package citywork

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

var tracer = otel.Tracer("api-opendata/svcs/citywork")

const (
	DefaultBrokerTenant string = "default"
)

type CityworkService interface {
	Broker() string
	Tenant() string

	GetAll() []byte

	Start()
	Shutdown()
}

func NewCityworkService(ctx context.Context, logger zerolog.Logger, contextBrokerUrl, tenant string) CityworkService {
	svc := &cityworkSvc{
		ctx:              ctx,
		log:              logger,
		citywork:         []byte("[]"),
		tenant:           tenant,
		contextBrokerURL: contextBrokerUrl,

		keepRunning: true,
	}

	return svc
}

type cityworkSvc struct {
	contextBrokerURL string
	tenant           string

	cityworkMutex sync.Mutex
	citywork      []byte

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *cityworkSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *cityworkSvc) Tenant() string {
	return svc.tenant
}

func (svc *cityworkSvc) GetAll() []byte {
	svc.cityworkMutex.Lock()
	defer svc.cityworkMutex.Unlock()

	return svc.citywork
}

func (svc *cityworkSvc) Start() {
	svc.log.Info().Msg("starting citywork service")
	go svc.run()
}

func (svc *cityworkSvc) Shutdown() {
	svc.log.Info().Msg("shutting down citywork service")
	svc.keepRunning = false
}

func (svc *cityworkSvc) run() {
	nextRefreshTime := time.Now()

	for svc.keepRunning {
		if time.Now().After(nextRefreshTime) {
			svc.log.Info().Msg("refreshing roadwork info")
			err := svc.refresh()

			if err != nil {
				svc.log.Error().Err(err).Msg("failed to refresh citywork")
				// Retry every 10 seconds on error
				nextRefreshTime = time.Now().Add(10 * time.Second)
			} else {
				// Refresh every 5 minutes on success
				nextRefreshTime = time.Now().Add(5 * time.Minute)
			}
		}

		time.Sleep(1 * time.Second)
	}

	svc.log.Info().Msg("citywork service exiting")
}

func (svc *cityworkSvc) refresh() error {
	var err error
	ctx, span := tracer.Start(svc.ctx, "refresh-citywork")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, svc.log, ctx)

	citywork := []domain.Citywork{}

	err = svc.getCityworkFromContextBroker(ctx, func(r cityworkDTO) {
		// do we need a callback function here? saving this and filling in bit more until i'm sure i need this
	})

	jsonBytes, err := json.MarshalIndent(citywork, "  ", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal citywork to json")
		return err
	}

	svc.storeCityworkList(jsonBytes)

	return err
}

func (svc *cityworkSvc) storeCityworkList(body []byte) {
	svc.cityworkMutex.Lock()
	defer svc.cityworkMutex.Unlock()

	svc.citywork = body
}

func (svc *cityworkSvc) getCityworkFromContextBroker(ctx context.Context, callback func(r cityworkDTO)) error {
	var err error

	logger := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	//unsure if below url suffix is correct
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.contextBrokerURL+"/ngsi-ld/v1/entities?type=RoadWork&limit=100&options=keyValues", nil)
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

	var citywork []cityworkDTO
	err = json.Unmarshal(respBody, &citywork)
	if err != nil {
		return fmt.Errorf("failed to unmarshal repsonse: %s", err.Error())
	}

	for _, r := range citywork {
		callback(r)
	}

	return nil
}

type cityworkDTO struct {
}
