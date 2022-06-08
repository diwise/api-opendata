package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/temperature"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type TempResponseValue struct {
	Average *float64   `json:"avg,omitempty"`
	Max     *float64   `json:"max,omitempty"`
	Min     *float64   `json:"min,omitempty"`
	Value   *float64   `json:"value,omitempty"`
	When    *time.Time `json:"when,omitempty"`
	From    *time.Time `json:"from,omitempty"`
	To      *time.Time `json:"to,omitempty"`
}

type TempResponseSensor struct {
	ID     string              `json:"id"`
	Values []TempResponseValue `json:"values"`
}

type TempResponse struct {
	Sensors []TempResponseSensor `json:"sensors"`
}

func getTimeParamsFromURL(r *http.Request) (time.Time, time.Time, error) {

	var err error

	startTime := time.Now().UTC().Add(-1 * 24 * time.Hour)
	endTime := time.Now().UTC()

	from := r.URL.Query().Get("timeAt")
	if from != "" {
		startTime, err = time.Parse(time.RFC3339, from)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	to := r.URL.Query().Get("endTimeAt")
	if to != "" {
		endTime, err = time.Parse(time.RFC3339, to)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	return startTime, endTime, nil
}

func NewRetrieveTemperaturesHandler(log zerolog.Logger, svc services.TempService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		query := svc.Query().Sensor(r.URL.Query().Get("sensor"))

		from, to, err := getTimeParamsFromURL(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Msg("unable to get time range")
			return
		}

		query = query.BetweenTimes(from, to)

		if r.URL.Query().Get("options") == "aggregatedValues" {
			methods := r.URL.Query().Get("aggrMethods")
			duration := r.URL.Query().Get("aggrPeriodDuration")
			query = query.Aggregate(duration, methods)
		}

		sensors, err := query.Get(r, log)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Msg("unable to get temperatures")
			return
		}

		response := &TempResponse{
			Sensors: []TempResponseSensor{},
		}

		for _, s := range sensors {
			sensor := TempResponseSensor{
				ID:     s.Id,
				Values: []TempResponseValue{},
			}

			for _, t := range s.Temperatures {
				value := TempResponseValue{
					Average: t.Average,
					Max:     t.Max,
					Min:     t.Min,
					Value:   t.Value,
					When:    t.When,
					From:    t.From,
					To:      t.To,
				}

				sensor.Values = append(sensor.Values, value)
			}

			response.Sensors = append(response.Sensors, sensor)
		}

		w.Header().Add("Content-Type", "application/json")

		bytes, err := json.MarshalIndent(response, " ", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Msg("unable to marshal results to json")
			return
		}

		w.Write(bytes)
	})
}

func NewRetrieveTemperatureSensorsHandler(log zerolog.Logger, brokerURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		httpClient := http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}

		ctx, span := tracer.Start(r.Context(), "temp-sensors-handler")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		traceID := span.SpanContext().TraceID()
		if traceID.IsValid() {
			log = log.With().Str("traceID", traceID.String()).Logger()
		}

		ctx = logging.NewContextWithLogger(ctx, log)

		url := fmt.Sprintf("%s/ngsi-ld/v1/entities?type=Device", brokerURL)

		log.Info().Msgf("requesting device information from %s", url)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			log.Error().Err(err).Msg("failed to create http request")
			return
		}

		response, err := httpClient.Do(req)
		if err != nil {
			log.Error().Err(err).Msg("failed to query devices from broker")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			log.Error().Err(err).Msgf("broker responded to device query with status %d", response.StatusCode)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		devices := []fiware.Device{}
		b, _ := io.ReadAll(response.Body)
		err = json.Unmarshal(b, &devices)

		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal response from broker")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		numberOfTempSensors := 0

		for _, d := range devices {
			tempSensor, isTempSensor := filterTempSensorInfo(d)
			if isTempSensor {
				devices[numberOfTempSensors] = tempSensor
				numberOfTempSensors++
			}
		}

		bytes, err := json.MarshalIndent(devices[0:numberOfTempSensors], " ", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Msg("unable to marshal devices to json")
			return
		}

		w.Header().Add("Content-Type", "application/ld+json")
		w.WriteHeader(http.StatusOK)
		w.Write(bytes)
	})
}

func filterTempSensorInfo(device fiware.Device) (fiware.Device, bool) {
	deviceID := device.ID

	const tfvSensorPath string = "se:trafikverket:temp:"
	if strings.Contains(deviceID, tfvSensorPath) {
		device.ID = strings.ReplaceAll(deviceID, tfvSensorPath, "")
		device.DateLastValueReported = nil // could be useful information, but isn't always correct at this point
		device.RefDeviceModel = nil
		device.Value = nil
		return device, true
	}

	return device, false
}
