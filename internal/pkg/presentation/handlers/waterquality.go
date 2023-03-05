package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
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

		wqos := svc.GetAll()
		if err != nil {
			log.Error().Err(err).Msgf("failed to get waterquality from %s", svc.Broker())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		body, err := json.Marshal(wqos)

		waterQualityJSON := "{\n  \"data\": " + string(body) + "\n}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(waterQualityJSON))

	})
}
