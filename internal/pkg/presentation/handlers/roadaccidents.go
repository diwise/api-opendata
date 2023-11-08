package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/application/services/roadaccidents"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
)

func NewRetrieveRoadAccidentByIDHandler(ctx context.Context, roadAccidentSvc roadaccidents.RoadAccidentService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-road-accident-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		roadAccidentID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if roadAccidentID == "" {
			err = fmt.Errorf("no road accident id supplied in query")
			log.Error("bad request", slog.String("err", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		body, err := roadAccidentSvc.GetByID(roadAccidentID)

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

func NewRetrieveRoadAccidentsHandler(ctx context.Context, roadAccidentSvc roadaccidents.RoadAccidentService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		_, span := tracer.Start(r.Context(), "retrieve-road-accidents")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		body := roadAccidentSvc.GetAll()

		roadAccidentJSON := "{\"data\": " + string(body) + "}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(roadAccidentJSON))
	})
}
