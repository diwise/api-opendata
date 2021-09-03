package datasets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
)

func NewRetrieveTrafficFlowsHandler(log logging.Logger, contextBroker string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tfosCsv := bytes.NewBufferString("id;date_observed;lane_id;intensity;latitude;longitude")

		tfos, err := getTrafficFlowsFromContextBroker(contextBroker)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("Failed to get trafficFlows from %s: %s", contextBroker, err.Error())
			return
		}

		for _, tfo := range tfos {

			tfoInfo := fmt.Sprintf("\r\n%s;%s;%d;%d;",
				tfo.ID,
				tfo.DateObserved.Value,
				int(tfo.LaneID.Value),
				int(tfo.Intensity.Value),
			)

			tfosCsv.Write([]byte(tfoInfo))
		}

		w.Header().Add("Content-Type", "text/csv")
		w.Write(tfosCsv.Bytes())
	})
}

func getTrafficFlowsFromContextBroker(host string) ([]*fiware.TrafficFlowObserved, error) {
	response, err := http.Get(fmt.Sprintf("%s/ngsi-ld/v1/entities?type=TrafficFlowObserved", host))
	if response.StatusCode != http.StatusOK {
		return nil, err
	}
	defer response.Body.Close()

	tfos := []*fiware.TrafficFlowObserved{}

	err = json.NewDecoder(response.Body).Decode(&tfos)

	return tfos, err
}
