package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRetrieveTrafficFlowsHandler(log zerolog.Logger, contextBroker string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tfosCsv := bytes.NewBufferString("date_observed;road_segment;L0_CNT;L0_AVG;L1_CNT;L1_AVG;L2_CNT;L2_AVG;L3_CNT;L3_AVG;R0_CNT;R0_AVG;R1_CNT;R1_AVG;R2_CNT;R2_AVG;R3_CNT;R3_AVG")

		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")

		tfos, err := getTrafficFlowsFromContextBroker(r, log, contextBroker, from, to)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Msgf("failed to get traffic flow observations from %s", contextBroker)
			return
		}

		w.Header().Add("Content-Type", "text/csv")

		if len(tfos) == 0 {
			w.Write(tfosCsv.Bytes())
			return
		}

		sameDateIntensity := [8]int{}
		sameDateAvgSpeed := [8]float64{}

		currentDate := tfos[0].DateObserved.Value

		for _, tfo := range tfos {

			tfoDateObserved := tfo.DateObserved.Value

			if strings.Compare(currentDate, tfoDateObserved) != 0 {
				tfoInfo := fmt.Sprintf("\r\n%s;%s;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f",
					currentDate, "urn:ngsi-ld:RoadSegment:19312:2860:35243",
					sameDateIntensity[0], sameDateAvgSpeed[0], sameDateIntensity[1], sameDateAvgSpeed[1],
					sameDateIntensity[2], sameDateAvgSpeed[2], sameDateIntensity[3], sameDateAvgSpeed[3],
					sameDateIntensity[4], sameDateAvgSpeed[4], sameDateIntensity[5], sameDateAvgSpeed[5],
					sameDateIntensity[6], sameDateAvgSpeed[6], sameDateIntensity[7], sameDateAvgSpeed[7],
				)

				sameDateIntensity = [8]int{}
				sameDateAvgSpeed = [8]float64{}

				currentDate = tfoDateObserved

				tfosCsv.Write([]byte(tfoInfo))
			}

			tfoLaneId := int(tfo.LaneID.Value)
			sameDateAvgSpeed[tfoLaneId] = tfo.AverageVehicleSpeed.Value
			sameDateIntensity[tfoLaneId] = int(tfo.Intensity.Value)

		}

		tfoInfo := fmt.Sprintf("\r\n%s;%s;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f;%d;%.1f",
			currentDate, "urn:ngsi-ld:RoadSegment:19312:2860:35243",
			sameDateIntensity[0], sameDateAvgSpeed[0], sameDateIntensity[1], sameDateAvgSpeed[1],
			sameDateIntensity[2], sameDateAvgSpeed[2], sameDateIntensity[3], sameDateAvgSpeed[3],
			sameDateIntensity[4], sameDateAvgSpeed[4], sameDateIntensity[5], sameDateAvgSpeed[5],
			sameDateIntensity[6], sameDateAvgSpeed[6], sameDateIntensity[7], sameDateAvgSpeed[7],
		)

		tfosCsv.Write([]byte(tfoInfo))

		w.Write(tfosCsv.Bytes())

	})
}

func getTrafficFlowsFromContextBroker(r *http.Request, log zerolog.Logger, host, from, to string) ([]*fiware.TrafficFlowObserved, error) {
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

	url := fmt.Sprintf("%s/ngsi-ld/v1/entities?type=TrafficFlowObserved", host)

	if from != "" && to != "" {
		url = fmt.Sprintf("%s&timerel=between&timeAt=%s&endTimeAt=%s", url, from, to)
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
		return nil, fmt.Errorf("request failed, status code not ok: %d", response.StatusCode)
	}

	defer response.Body.Close()

	tfos := []*fiware.TrafficFlowObserved{}

	err = json.NewDecoder(response.Body).Decode(&tfos)

	return tfos, err
}
