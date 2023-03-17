package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/api")

func NewRetrieveWaterQualityHandler(logger zerolog.Logger, svc waterquality.WaterQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-water-qualities")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		maxDistance := r.URL.Query().Get("maxDistance")
		if maxDistance != "" {
			distance, err := strconv.ParseInt(maxDistance, 0, 64)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse distance from query parameters")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			svc.Distance(int(distance))
		}

		coordinates := r.URL.Query().Get("coordinates")
		if coordinates != "" {
			coords := strings.Split(coordinates, ",")

			longitude, _ := strconv.ParseFloat(coords[0], 64)
			latitude, _ := strconv.ParseFloat(coords[1], 64)

			svc.Location(latitude, longitude)
		}

		wqos := svc.GetAll()

		wqosBytes, err := json.Marshal(wqos)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal water quality into json")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		waterQualityJSON := "{\n  \"data\": " + string(wqosBytes) + "\n}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(waterQualityJSON))

	})
}

func NewRetrieveWaterQualityByIDHandler(logger zerolog.Logger, svc waterquality.WaterQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-water-qualities")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		waterqualityID, err := url.QueryUnescape(chi.URLParam(r, "id"))
		if waterqualityID == "" {
			err = fmt.Errorf("no water quality id is supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		wqo, err := svc.GetByID(waterqualityID)
		if err != nil {
			log.Error().Err(err).Msgf("no water quality found with id %s", waterqualityID)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		body, err := json.Marshal(wqo)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal water quality")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		waterQualityJSON := "{\n  \"data\": " + string(body) + "\n}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(waterQualityJSON))

	})
}
