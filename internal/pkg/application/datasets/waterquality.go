package datasets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"
)

func NewRetrieveWaterQualityHandler(log logging.Logger, contextBroker string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		waterQualityCsv := bytes.NewBufferString("timestamp;latitude;longitude;temperature;sensor")

		waterquality, err := getWaterQualityFromContextBroker(contextBroker)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("Failed to get waterquality from %s: %s", contextBroker, err.Error())
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

func getWaterQualityFromContextBroker(host string) ([]*fiware.WaterQualityObserved, error) {
	response, err := http.Get(fmt.Sprintf("https://%s/ngsi-ld/v1/entities?type=WaterQualityObserved&georel=near;maxDistance==50000&geometry=Point&coordinates=%%5B17.2742640,62.37492958%%5D", host))
	if response.StatusCode != http.StatusOK {
		return nil, err
	}
	defer response.Body.Close()

	waterquality := []*fiware.WaterQualityObserved{}

	err = json.NewDecoder(response.Body).Decode(&waterquality)

	return waterquality, err
}
