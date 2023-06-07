package handlers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/diwise/api-opendata/internal/pkg/application/services/citywork"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func NewRetrieveCityworksHandler(logger zerolog.Logger, cityworkSvc citywork.CityworksService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := cityworkSvc.GetAll()

		roadworksJSON := "{\"data\": " + string(body) + "}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(roadworksJSON))
	})
}

func NewRetrieveCityworksByIDHandler(logger zerolog.Logger, cityworkSvc citywork.CityworksService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-cityworks-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		cityworkID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if cityworkID == "" {
			err = fmt.Errorf("no cityworks id supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		body, err := cityworkSvc.GetByID(cityworkID)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		body = []byte("{\"data\": " + string(body) + "}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(body)
	})
}
