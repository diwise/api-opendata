package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/api")

func NewRetrieveWaterQualityHandler(logger zerolog.Logger, contextBroker string, waterQualityQueryParams string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-water-qualities")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		waterQualityCsv := bytes.NewBufferString("timestamp;latitude;longitude;temperature;sensor")

		waterquality, err := getWaterQualityFromContextBroker(ctx, log, contextBroker, waterQualityQueryParams)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get waterquality from %s", contextBroker)
			w.WriteHeader(http.StatusInternalServerError)
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

func getWaterQualityFromContextBroker(ctx context.Context, log zerolog.Logger, host string, queryParams string) ([]*fiware.WaterQualityObserved, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	url := host + "/ngsi-ld/v1/entities?type=WaterQualityObserved"
	if len(queryParams) > 0 {
		url = url + "&" + queryParams
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		err = fmt.Errorf("failed to create http request: %w", err)
		return nil, err
	}

	response, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("request failed: %w", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed with status code %d", response.StatusCode)
		return nil, err
	}

	waterquality := []*fiware.WaterQualityObserved{}
	err = json.NewDecoder(response.Body).Decode(&waterquality)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return waterquality, err
}
