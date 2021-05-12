package datasets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"
)

func NewRetrieveWaterQualityHandler(log logging.Logger, contextBroker string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		waterQualityCsv := bytes.NewBufferString("updated;latitude;longitude;temperature;sensor")

		waterquality, err := getWaterQualityFromContextBroker(contextBroker)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("Failed to get waterquality from %s: %s", contextBroker, err.Error())
			return
		}

		for _, wq := range waterquality {
			lonLat := wq.Location.GetAsPoint()
			timestamp := wq.DateObserved.Value.Value
			temp := wq.Temperature.Value

			var sensor string
			if wq.RefDevice != nil {
				sensor = strings.TrimPrefix(wq.RefDevice.Object, fiware.DeviceIDPrefix)
			}

			wqInfo := fmt.Sprintf("\r\n%s;%f;%f;%f;%s",
				timestamp, lonLat.Coordinates[0], lonLat.Coordinates[1],
				temp,
				sensor,
			)

			waterQualityCsv.Write([]byte(wqInfo))
		}

		w.Header().Add("Content-Type", "text/csv")
		w.Write(waterQualityCsv.Bytes())

	})
}

func getWaterQualityFromContextBroker(host string) ([]*fiware.WaterQualityObserved, error) {
	response, err := http.Get(fmt.Sprintf("https://%s/ngsi-ld/v1/entities?type=WaterQualityObserved", host))
	if response.StatusCode != http.StatusOK {
		return nil, err
	}
	defer response.Body.Close()

	waterquality := []*fiware.WaterQualityObserved{}

	err = json.NewDecoder(response.Body).Decode(&waterquality)

	return waterquality, err
}
