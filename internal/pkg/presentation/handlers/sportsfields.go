package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsfields"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func NewRetrieveSportsFieldByIDHandler(logger zerolog.Logger, sfsvc sportsfields.SportsFieldService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-sportsfield-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		sportsfieldID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if sportsfieldID == "" {
			err = fmt.Errorf("no sports field is supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		sportsfield, err := sfsvc.GetByID(sportsfieldID)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		responseBody, err := json.Marshal(sportsfield)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal sportsfield to json")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		responseBody = []byte("{\"data\":" + string(responseBody) + "}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(responseBody)
	})
}

func NewRetrieveSportsFieldsHandler(logger zerolog.Logger, sfsvc sportsfields.SportsFieldService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//todo
	})
}
