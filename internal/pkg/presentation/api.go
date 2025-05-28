package presentation

import (
	"bytes"
	"compress/flate"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/application/services/airquality"

	"github.com/diwise/api-opendata/internal/pkg/application/services/beaches"
	"github.com/diwise/api-opendata/internal/pkg/application/services/citywork"
	"github.com/diwise/api-opendata/internal/pkg/application/services/exercisetrails"
	"github.com/diwise/api-opendata/internal/pkg/application/services/organisations"
	"github.com/diwise/api-opendata/internal/pkg/application/services/roadaccidents"
	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsfields"
	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsvenues"
	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/api-opendata/internal/pkg/application/services/weather"
	"github.com/diwise/api-opendata/internal/pkg/presentation/handlers"
	"github.com/diwise/api-opendata/internal/pkg/presentation/handlers/stratsys"
	"github.com/diwise/context-broker/pkg/ngsild/client"
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
	logger.Info(fmt.Sprintf("Starting api-opendata on port:%s", port))

	return http.ListenAndServe(":"+port, a.router)
}

type svcEntry struct {
	key      string
	setup    func(ctx context.Context)
	register func(r chi.Router)
}

func (o *opendataAPI) addDiwiseHandlers(ctx context.Context, r chi.Router, orgfile io.Reader) {
	logger := logging.GetFromContext(ctx)

	contextBrokerURL := env.GetVariableOrDie(ctx, "DIWISE_CONTEXT_BROKER_URL", "context broker URL")
	contextBrokerTenant := env.GetVariableOrDefault(ctx, "DIWISE_CONTEXT_BROKER_TENANT", entities.DefaultNGSITenant)

	organisationsRegistry, err := organisations.NewRegistry(orgfile)
	if err != nil {
		logger.Error("failed to create organisations registry", slog.String("err", err.Error()))
		os.Exit(1)
	}

	cbClient := client.NewContextBrokerClient(contextBrokerURL, client.Tenant("default"))

	enabledEnv := env.GetVariableOrDefault(ctx, "ENABLED_SERVICES", "all")
	enabled, err := parseEnabledServices(enabledEnv)
	if err != nil {
		logger.Error("failed to parse enabled services from environment variable")
		os.Exit(1)
	}

	services := make(map[string]any)

	entries := []svcEntry{
		{
			key: "airqualities",
			setup: func(ctx context.Context) {
				svc := airquality.NewAirQualityService(ctx, cbClient, contextBrokerTenant)
				svc.Start(ctx)
				services["airqualities"] = svc
			},
			register: func(r chi.Router) {
				svc := services["airqualities"].(airquality.AirQualityService)
				r.Get(
					"/api/airqualities",
					handlers.NewRetrieveAirQualitiesHandler(ctx, svc),
				)
				r.Get(
					"/api/airqualities/{id}",
					handlers.NewRetrieveAirQualityByIDHandler(ctx, svc),
				)
			},
		},
		{
			key: "beaches",
			setup: func(ctx context.Context) {
				waterqualitySvc := waterquality.NewWaterQualityService(ctx, contextBrokerURL, contextBrokerTenant)
				waterqualitySvc.Start(ctx)
				services["waterqualities"] = waterqualitySvc

				maxWQODistStr := env.GetVariableOrDefault(ctx, "WATER_QUALITY_MAX_DISTANCE", "1000")
				maxWQODistance, err := strconv.ParseInt(maxWQODistStr, 10, 32)
				if err != nil {
					maxWQODistance = 1000
				}

				beachService := beaches.NewBeachService(ctx, contextBrokerURL, contextBrokerTenant, int(maxWQODistance), waterqualitySvc)
				beachService.Start(ctx)
				services["beaches"] = beachService
			},
			register: func(r chi.Router) {
				wqsvc := services["waterqualities"].(waterquality.WaterQualityService)
				r.Get(
					"/api/waterqualities",
					handlers.NewRetrieveWaterQualityHandler(ctx, wqsvc),
				)
				r.Get(
					"/api/waterqualities/{id}",
					handlers.NewRetrieveWaterQualityByIDHandler(ctx, wqsvc),
				)

				beachsvc := services["beaches"].(beaches.BeachService)
				r.Get(
					"/api/beaches",
					handlers.NewRetrieveBeachesHandler(ctx, beachsvc),
				)
				r.Get(
					"/api/beaches/{id}",
					handlers.NewRetrieveBeachByIDHandler(ctx, beachsvc),
				)
			},
		},
		{
			key: "cityworks",
			setup: func(ctx context.Context) {
				svc := citywork.NewCityworksService(ctx, contextBrokerURL, contextBrokerTenant)
				svc.Start(ctx)
				services["cityworks"] = svc
			},
			register: func(r chi.Router) {
				svc := services["cityworks"].(citywork.CityworksService)
				r.Get(
					"/api/cityworks",
					handlers.NewRetrieveCityworksHandler(ctx, svc),
				)
				r.Get(
					"/api/cityworks/{id}",
					handlers.NewRetrieveCityworksByIDHandler(ctx, svc),
				)
			},
		},
		{
			key: "exercisetrails",
			setup: func(ctx context.Context) {
				svc := exercisetrails.NewExerciseTrailService(ctx, contextBrokerURL, contextBrokerTenant, organisationsRegistry)
				svc.Start(ctx)
				services["exercisetrails"] = svc

			},
			register: func(r chi.Router) {
				svc := services["exercisetrails"].(exercisetrails.ExerciseTrailService)

				r.Get(
					"/api/exercisetrails",
					handlers.NewRetrieveExerciseTrailsHandler(ctx, svc),
				)
				r.Get(
					"/api/exercisetrails/{id}",
					handlers.NewRetrieveExerciseTrailByIDHandler(ctx, svc),
				)
			},
		},
		{
			key: "roadaccidents",
			setup: func(ctx context.Context) {
				svc := roadaccidents.NewRoadAccidentService(ctx, contextBrokerURL, contextBrokerTenant)
				svc.Start(ctx)
				services["roadaccidents"] = svc
			},
			register: func(r chi.Router) {
				svc := services["roadaccidents"].(roadaccidents.RoadAccidentService)
				r.Get(
					"/api/roadaccidents",
					handlers.NewRetrieveRoadAccidentsHandler(ctx, svc),
				)
				r.Get(
					"/api/roadaccidents/{id}",
					handlers.NewRetrieveRoadAccidentByIDHandler(ctx, svc),
				)
			},
		},
		{
			key: "sportsfields",
			setup: func(ctx context.Context) {
				svc := sportsfields.NewSportsFieldService(ctx, contextBrokerURL, contextBrokerTenant, organisationsRegistry)
				svc.Start(ctx)
				services["sportsfields"] = svc
			},
			register: func(r chi.Router) {
				svc := services["sportsfields"].(sportsfields.SportsFieldService)

				r.Get(
					"/api/sportsfields",
					handlers.NewRetrieveSportsFieldsHandler(ctx, svc),
				)
				r.Get(
					"/api/sportsfields/{id}",
					handlers.NewRetrieveSportsFieldByIDHandler(ctx, svc),
				)
			},
		},
		{
			key: "sportsvenues",
			setup: func(ctx context.Context) {
				svc := sportsvenues.NewSportsVenueService(ctx, contextBrokerURL, contextBrokerTenant, organisationsRegistry)
				svc.Start(ctx)
				services["sportsvenues"] = svc
			},
			register: func(r chi.Router) {
				svc := services["sportsvenues"].(sportsvenues.SportsVenueService)
				r.Get(
					"/api/sportsvenues",
					handlers.NewRetrieveSportsVenuesHandler(ctx, svc),
				)
				r.Get(
					"/api/sportsvenues/{id}",
					handlers.NewRetrieveSportsVenueByIDHandler(ctx, svc),
				)
			},
		},
		{
			key:   "stratsys",
			setup: func(ctx context.Context) {},
			register: func(r chi.Router) {
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
			},
		},
		{
			key:   "traffic",
			setup: func(ctx context.Context) {},
			register: func(r chi.Router) {
				r.Get(
					"/api/trafficflow",
					handlers.NewRetrieveTrafficFlowsHandler(ctx, contextBrokerURL),
				)
			},
		},
		{
			key: "waterqualities",
			setup: func(ctx context.Context) {
				svc := waterquality.NewWaterQualityService(ctx, contextBrokerURL, contextBrokerTenant)
				svc.Start(ctx)
				services["waterqualities"] = svc
			},
			register: func(r chi.Router) {
				svc := services["waterqualities"].(waterquality.WaterQualityService)

				r.Get(
					"/api/waterqualities",
					handlers.NewRetrieveWaterQualityHandler(ctx, svc),
				)
				r.Get(
					"/api/waterqualities/{id}",
					handlers.NewRetrieveWaterQualityByIDHandler(ctx, svc),
				)
			},
		},
		{
			key: "weather",
			setup: func(ctx context.Context) {
				svc := weather.NewWeatherService(ctx, contextBrokerURL, contextBrokerTenant)
				services["weather"] = svc
			},
			register: func(r chi.Router) {
				svc := services["weather"].(weather.WeatherService)
				r.Get(
					"/api/weather",
					handlers.NewRetrieveWeatherHandler(ctx, svc),
				)
				r.Get(
					"/api/weather/{id}",
					handlers.NewRetrieveWeatherByIDHandler(ctx, svc),
				)
			},
		},
	}

	for _, e := range entries {
		if enabled["all"] || enabled[e.key] {
			e.setup(ctx)
			e.register(o.router)
		}
	}
}

func parseEnabledServices(s string) (map[string]bool, error) {
	m := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		m[p] = true
	}
	return m, nil
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
