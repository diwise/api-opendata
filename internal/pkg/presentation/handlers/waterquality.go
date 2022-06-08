package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/api")

func NewRetrieveWaterQualityHandler(log zerolog.Logger, contextBroker string, waterQualityQueryParams string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		waterQualityCsv := bytes.NewBufferString("timestamp;latitude;longitude;temperature;sensor")

		waterquality, err := getWaterQualityFromContextBroker(r, log, contextBroker, waterQualityQueryParams)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Msgf("failed to get waterquality from %s", contextBroker)
			return
		}

		for _, wq := range waterquality {
			lonLat := wq.Location.GetAsPoint()
			timestamp := wq.DateObserved.Value.Value
			temp := strconv.FormatFloat(wq.Temperature.Value, 'f', -1, 64)

			var sensor string
			if wq.RefDevice != nil {
				sensor = strings.TrimPrefix(wq.RefDevice.Object, fiware.DeviceIDPrefix)
			}

			wqInfo := fmt.Sprintf("\r\n%s;%f;%f;%s;%s",
				timestamp, lonLat.Coordinates[1], lonLat.Coordinates[0],
				temp,
				sensor,
			)

			waterQualityCsv.Write([]byte(wqInfo))
		}

		w.Header().Add("Content-Type", "text/csv")
		w.Write(waterQualityCsv.Bytes())

	})
}

func getWaterQualityFromContextBroker(r *http.Request, log zerolog.Logger, host string, queryParams string) ([]*fiware.WaterQualityObserved, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	ctx, span := tracer.Start(r.Context(), "water-quality-handler")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	traceID := span.SpanContext().TraceID()
	if traceID.IsValid() {
		log = log.With().Str("traceID", traceID.String()).Logger()
	}

	url := host + "/ngsi-ld/v1/entities?type=WaterQualityObserved"
	if len(queryParams) > 0 {
		url = url + "&" + queryParams
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to create http request")
		return nil, err
	}

	response, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get water quality observed from context broker")
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed with status code %d", response.StatusCode)
	}

	defer response.Body.Close()

	waterquality := []*fiware.WaterQualityObserved{}
	err = json.NewDecoder(response.Body).Decode(&waterquality)

	return waterquality, err
}
