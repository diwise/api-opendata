package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
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

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		maxDistance := r.URL.Query().Get("maxDistance")
		var distance int64
		if maxDistance != "" {
			distance, err = strconv.ParseInt(maxDistance, 0, 64)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse distance from query parameters")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		coordinates := r.URL.Query().Get("coordinates")
		var longitude, latitude float64
		if coordinates != "" {
			coords := strings.Split(coordinates, ",")

			longitude, _ = strconv.ParseFloat(coords[0], 64)
			latitude, _ = strconv.ParseFloat(coords[1], 64)
		}

		var wqos []domain.WaterQuality

		if distance != 0 {
			wqos, err = svc.GetAllNearPoint(ctx, waterquality.NewPoint(latitude, longitude), int(distance))
		} else {
			wqos = svc.GetAll(ctx)
		}

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

		from := time.Time{}
		to := time.Time{}

		fromStr := chi.URLParam(r, "from")
		if fromStr != "" {
			from, err = time.Parse(time.RFC3339, fromStr)
			if err != nil {
				log.Error().Err(err).Msg("time parameter from is incorrect format")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		toStr := chi.URLParam(r, "to")
		if toStr != "" {
			to, err = time.Parse(time.RFC3339, toStr)
			if err != nil {
				log.Error().Err(err).Msg("time parameter to is incorrect format")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		wqo, err := svc.GetByID(ctx, waterqualityID, from, to)
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
