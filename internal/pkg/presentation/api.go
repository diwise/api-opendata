package application

import (
	"bytes"
	"compress/flate"
	"context"
	"net/http"
	"os"

	"github.com/diwise/api-opendata/internal/pkg/application/services/temperature"
	"github.com/diwise/api-opendata/internal/pkg/presentation/handlers"
	"github.com/diwise/api-opendata/internal/pkg/presentation/handlers/stratsys"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"

	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

type API interface {
	Start(port string) error
}

type opendataAPI struct {
	router chi.Router
	log    zerolog.Logger
}

func NewAPI(r chi.Router, ctx context.Context, dcatResponse *bytes.Buffer, openapiResponse *bytes.Buffer) API {
	return newOpendataAPI(r, ctx, dcatResponse, openapiResponse)
}

func newOpendataAPI(r chi.Router, ctx context.Context, dcatResponse *bytes.Buffer, openapiResponse *bytes.Buffer) *opendataAPI {
	log := logging.GetFromContext(ctx)

	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	// Enable gzip compression for our responses
	compressor := middleware.NewCompressor(
		flate.DefaultCompression,
		"text/csv", "application/json", "application/xml", "application/rdf+xml",
	)
	r.Use(compressor.Handler)
	r.Use(otelchi.Middleware("api-opendata", otelchi.WithChiRoutes(r)))

	o := &opendataAPI{
		router: r,
		log:    log,
	}

	o.addDiwiseHandlers(r, log)
	o.addProbeHandlers(r)

	o.router.Get("/api/datasets/dcat", o.newRetrieveDatasetsHandler(log, dcatResponse))
	o.router.Get("/api/api-docs", o.newRetrieveOpenAPIHandler(log, openapiResponse))
	o.router.Get("/api/openapi", o.newRetrieveOpenAPIHandler(log, openapiResponse))

	return o
}

func (a *opendataAPI) Start(port string) error {
	a.log.Info().Msgf("Starting api-opendata on port:%s", port)
	return http.ListenAndServe(":"+port, a.router)
}

func (o *opendataAPI) addDiwiseHandlers(r chi.Router, log zerolog.Logger) {
	contextBrokerURL := env.GetVariableOrDie(log, "DIWISE_CONTEXT_BROKER_URL", "context broker URL")
	contextBrokerTenant := env.GetVariableOrDefault(log, "DIWISE_CONTEXT_BROKER_TENANT", handlers.DefaultBrokerTenant)

	waterQualityQueryParams := os.Getenv("WATER_QUALITY_QUERY_PARAMS")

	stratsysEnabled := (env.GetVariableOrDefault(log, "STRATSYS_ENABLED", "true") != "false")
	stratsysCompanyCode := os.Getenv("STRATSYS_COMPANY_CODE")
	stratsysClientId := os.Getenv("STRATSYS_CLIENT_ID")
	stratsysScope := os.Getenv("STRATSYS_SCOPE")
	stratsysLoginUrl := os.Getenv("STRATSYS_LOGIN_URL")
	stratsysDefaultUrl := os.Getenv("STRATSYS_DEFAULT_URL")

	r.Get(
		"/api/temperature/water",
		handlers.NewRetrieveWaterQualityHandler(log, contextBrokerURL, waterQualityQueryParams),
	)
	r.Get(
		"/api/beaches",
		handlers.NewRetrieveBeachesHandler(log, contextBrokerURL, contextBrokerTenant),
	)
	r.Get(
		"/api/beaches/{id}",
		handlers.NewRetrieveBeachByIDHandler(log, contextBrokerURL, contextBrokerTenant),
	)
	r.Get(
		"/api/temperature/air",
		handlers.NewRetrieveTemperaturesHandler(log, temperature.NewTempService(contextBrokerURL)),
	)
	r.Get(
		"/api/temperature/air/sensors",
		handlers.NewRetrieveTemperatureSensorsHandler(log, contextBrokerURL),
	)
	r.Get(
		"/api/trafficflow",
		handlers.NewRetrieveTrafficFlowsHandler(log, contextBrokerURL),
	)

	if stratsysEnabled {
		r.Get(
			"/api/stratsys/publishedreports",
			stratsys.NewRetrieveStratsysReportsHandler(log, stratsysCompanyCode, stratsysClientId, stratsysScope, stratsysLoginUrl, stratsysDefaultUrl),
		)
		r.Get(
			"/api/stratsys/publishedreports/{id}",
			stratsys.NewRetrieveStratsysReportsHandler(log, stratsysCompanyCode, stratsysClientId, stratsysScope, stratsysLoginUrl, stratsysDefaultUrl),
		)
	}
}

func (o *opendataAPI) addProbeHandlers(r chi.Router) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (o *opendataAPI) newRetrieveDatasetsHandler(log zerolog.Logger, dcatResponse *bytes.Buffer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/rdf+xml")
		w.Write(dcatResponse.Bytes())
	})
}

func (o *opendataAPI) newRetrieveOpenAPIHandler(log zerolog.Logger, openapiResponse *bytes.Buffer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if openapiResponse == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(openapiResponse.Bytes())
	})
}
