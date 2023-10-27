package presentation

import (
	"bytes"
	"compress/flate"
	"context"
	"io"
	"net/http"
	"os"
	"strconv"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/application/services/beaches"
	"github.com/diwise/api-opendata/internal/pkg/application/services/citywork"
	"github.com/diwise/api-opendata/internal/pkg/application/services/exercisetrails"
	"github.com/diwise/api-opendata/internal/pkg/application/services/organisations"
	"github.com/diwise/api-opendata/internal/pkg/application/services/roadaccidents"
	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsfields"
	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsvenues"
	"github.com/diwise/api-opendata/internal/pkg/application/services/temperature"
	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/api-opendata/internal/pkg/presentation/handlers"
	"github.com/diwise/api-opendata/internal/pkg/presentation/handlers/stratsys"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"

	"github.com/rs/cors"
)

type API interface {
	Start(ctx context.Context, port string) error
}

type opendataAPI struct {
	router chi.Router
}

func NewAPI(ctx context.Context, r chi.Router, dcatResponse *bytes.Buffer, openapiResponse *bytes.Buffer, orgfile io.Reader) API {
	return newOpendataAPI(ctx, r, dcatResponse, openapiResponse, orgfile)
}

func newOpendataAPI(ctx context.Context, r chi.Router, dcatResponse *bytes.Buffer, openapiResponse *bytes.Buffer, orgfile io.Reader) *opendataAPI {
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
	}

	o.addDiwiseHandlers(ctx, r, orgfile)
	o.addProbeHandlers(r)

	o.router.Get("/api/datasets/dcat", o.newRetrieveDatasetsHandler(ctx, dcatResponse))
	o.router.Get("/api/api-docs", o.newRetrieveOpenAPIHandler(ctx, openapiResponse))
	o.router.Get("/api/openapi", o.newRetrieveOpenAPIHandler(ctx, openapiResponse))

	return o
}

func (a *opendataAPI) Start(ctx context.Context, port string) error {
	logger := logging.GetFromContext(ctx)
	logger.Info("Starting api-opendata on port:%s", port)

	return http.ListenAndServe(":"+port, a.router)
}

func (o *opendataAPI) addDiwiseHandlers(ctx context.Context, r chi.Router, orgfile io.Reader) {
	logger := logging.GetFromContext(ctx)

	contextBrokerURL := env.GetVariableOrDie(ctx, "DIWISE_CONTEXT_BROKER_URL", "context broker URL")
	contextBrokerTenant := env.GetVariableOrDefault(ctx, "DIWISE_CONTEXT_BROKER_TENANT", entities.DefaultNGSITenant)
	maxWQODistStr := env.GetVariableOrDefault(ctx, "WATER_QUALITY_MAX_DISTANCE", "1000")

	maxWQODistance, err := strconv.ParseInt(maxWQODistStr, 10, 32)
	if err != nil {
		maxWQODistance = 1000
	}

	organisationsRegistry, err := organisations.NewRegistry(orgfile)
	if err != nil {
		logger.Error("failed to create organisations registry", slog.String("error", err.Error()))
		os.Exit(1)
	}

	waterqualitySvc := waterquality.NewWaterQualityService(ctx, contextBrokerURL, contextBrokerTenant)
	waterqualitySvc.Start(ctx)

	beachService := beaches.NewBeachService(ctx, contextBrokerURL, contextBrokerTenant, int(maxWQODistance), waterqualitySvc)
	beachService.Start(ctx)

	trailService := exercisetrails.NewExerciseTrailService(ctx, contextBrokerURL, contextBrokerTenant, organisationsRegistry)
	trailService.Start(ctx)

	cityworkService := citywork.NewCityworksService(ctx, contextBrokerURL, contextBrokerTenant)
	cityworkService.Start(ctx)

	roadAccidentSvc := roadaccidents.NewRoadAccidentService(ctx, contextBrokerURL, contextBrokerTenant)
	roadAccidentSvc.Start(ctx)

	sportsfieldsSvc := sportsfields.NewSportsFieldService(ctx, contextBrokerURL, contextBrokerTenant, organisationsRegistry)
	sportsfieldsSvc.Start(ctx)

	sportsvenuesSvc := sportsvenues.NewSportsVenueService(ctx, contextBrokerURL, contextBrokerTenant, organisationsRegistry)
	sportsvenuesSvc.Start(ctx)

	r.Get(
		"/api/beaches",
		handlers.NewRetrieveBeachesHandler(ctx, beachService),
	)
	r.Get(
		"/api/beaches/{id}",
		handlers.NewRetrieveBeachByIDHandler(ctx, beachService),
	)
	r.Get(
		"/api/cityworks",
		handlers.NewRetrieveCityworksHandler(ctx, cityworkService),
	)
	r.Get(
		"/api/cityworks/{id}",
		handlers.NewRetrieveCityworksByIDHandler(ctx, cityworkService),
	)
	r.Get(
		"/api/exercisetrails",
		handlers.NewRetrieveExerciseTrailsHandler(ctx, trailService),
	)
	r.Get(
		"/api/exercisetrails/{id}",
		handlers.NewRetrieveExerciseTrailByIDHandler(ctx, trailService),
	)
	r.Get(
		"/api/roadaccidents",
		handlers.NewRetrieveRoadAccidentsHandler(ctx, roadAccidentSvc),
	)
	r.Get(
		"/api/roadaccidents/{id}",
		handlers.NewRetrieveRoadAccidentByIDHandler(ctx, roadAccidentSvc),
	)
	r.Get(
		"/api/sportsfields",
		handlers.NewRetrieveSportsFieldsHandler(ctx, sportsfieldsSvc),
	)
	r.Get(
		"/api/sportsfields/{id}",
		handlers.NewRetrieveSportsFieldByIDHandler(ctx, sportsfieldsSvc),
	)
	r.Get(
		"/api/sportsvenues",
		handlers.NewRetrieveSportsVenuesHandler(ctx, sportsvenuesSvc),
	)
	r.Get(
		"/api/sportsvenues/{id}",
		handlers.NewRetrieveSportsVenueByIDHandler(ctx, sportsvenuesSvc),
	)
	r.Get(
		"/api/temperature/air",
		handlers.NewRetrieveTemperaturesHandler(ctx, temperature.NewTempService(contextBrokerURL)),
	)
	r.Get(
		"/api/temperature/air/sensors",
		handlers.NewRetrieveTemperatureSensorsHandler(ctx, contextBrokerURL),
	)
	r.Get(
		"/api/trafficflow",
		handlers.NewRetrieveTrafficFlowsHandler(ctx, contextBrokerURL),
	)
	r.Get(
		"/api/waterqualities",
		handlers.NewRetrieveWaterQualityHandler(ctx, waterqualitySvc),
	)
	r.Get(
		"/api/waterqualities/{id}",
		handlers.NewRetrieveWaterQualityByIDHandler(ctx, waterqualitySvc),
	)
	
	stratsysEnabled := (env.GetVariableOrDefault(ctx, "STRATSYS_ENABLED", "true") != "false")

	if stratsysEnabled {
		stratsysCompanyCode := os.Getenv("STRATSYS_COMPANY_CODE")
		stratsysClientId := os.Getenv("STRATSYS_CLIENT_ID")
		stratsysScope := os.Getenv("STRATSYS_SCOPE")
		stratsysLoginUrl := os.Getenv("STRATSYS_LOGIN_URL")
		stratsysDefaultUrl := os.Getenv("STRATSYS_DEFAULT_URL")

		r.Get(
			"/api/stratsys/publishedreports",
			stratsys.NewRetrieveStratsysReportsHandler(ctx, stratsysCompanyCode, stratsysClientId, stratsysScope, stratsysLoginUrl, stratsysDefaultUrl),
		)
		r.Get(
			"/api/stratsys/publishedreports/{id}",
			stratsys.NewRetrieveStratsysReportsHandler(ctx, stratsysCompanyCode, stratsysClientId, stratsysScope, stratsysLoginUrl, stratsysDefaultUrl),
		)
	}
}

func (o *opendataAPI) addProbeHandlers(r chi.Router) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (o *opendataAPI) newRetrieveDatasetsHandler(ctx context.Context, dcatResponse *bytes.Buffer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/rdf+xml")
		w.Write(dcatResponse.Bytes())
	})
}

func (o *opendataAPI) newRetrieveOpenAPIHandler(ctx context.Context, openapiResponse *bytes.Buffer) http.HandlerFunc {
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
