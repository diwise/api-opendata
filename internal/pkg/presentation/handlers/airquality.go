package handlers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/diwise/api-opendata/internal/pkg/application/services/airquality"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func NewRetrieveAirQualitiesHandler(logger zerolog.Logger, aqsvc airquality.AirQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		_, span := tracer.Start(r.Context(), "retrieve-air-qualities")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		body := aqsvc.GetAll()

		aqJSON := "{\n  \"data\": " + string(body) + "\n}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(aqJSON))
	})
}

func NewRetrieveAirQualityByIDHandler(logger zerolog.Logger, aqsvc airquality.AirQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-air-quality-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		airQualityID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if airQualityID == "" {
			err = fmt.Errorf("no air quality id supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		body, err := aqsvc.GetByID(airQualityID)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		body = []byte("{\n  \"data\": " + string(body) + "\n}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(body)
	})
}
