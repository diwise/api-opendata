package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/diwise/api-opendata/internal/pkg/application/services/beaches"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/diwise/service-chassis/pkg/presentation/api/http/errors"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	NUTSCodePrefix      string = "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/"
	WikidataPrefix      string = "https://www.wikidata.org/wiki/"
	YearMonthDayISO8601 string = "2006-01-02"
)

func NewRetrieveBeachByIDHandler(logger zerolog.Logger, beachService beaches.BeachService) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-beach-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		beachID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if beachID == "" {
			err = fmt.Errorf("no beach id supplied in query")
			log.Error().Err(err).Msg("bad request")
			problem := errors.NewProblemReport(http.StatusBadRequest, "badrequest", errors.Detail(err.Error()))
			problem.WriteResponse(w)
			return
		}

		beach, err := beachService.GetByID(ctx, beachID)
		if err != nil {
			problem := errors.NewProblemReport(http.StatusNotFound, "notfound", errors.Detail("no such beach"))
			problem.WriteResponse(w)
			return
		}

		beachJSON, err := json.Marshal(beach)

		body := []byte("{\n  \"data\": " + string(beachJSON) + "\n}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(body)
	})
}

func NewRetrieveBeachesHandler(logger zerolog.Logger, beachService beaches.BeachService) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		_, span := tracer.Start(r.Context(), "retrieve-beaches")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, _ := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, r.Context())

		beaches := beachService.GetAll(ctx)

		beachesJSON, err := json.Marshal(beaches)
		if err != nil {
			err = fmt.Errorf("failed to marshal beaches into json")
			log.Error().Err(err).Msg("internal server error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		body := "{\n  \"data\": " + string(beachesJSON) + "\n}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(body))
	})
}
