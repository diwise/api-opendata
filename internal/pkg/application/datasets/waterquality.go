package datasets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/rs/zerolog"
)

func NewRetrieveWaterQualityHandler(log zerolog.Logger, contextBroker string, waterQualityQueryParams string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		waterQualityCsv := bytes.NewBufferString("timestamp;latitude;longitude;temperature;sensor")

		waterquality, err := getWaterQualityFromContextBroker(contextBroker, waterQualityQueryParams)
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

func getWaterQualityFromContextBroker(host string, queryParams string) ([]*fiware.WaterQualityObserved, error) {
	url := host + "/ngsi-ld/v1/entities?type=WaterQualityObserved"
	if len(queryParams) > 0 {
		url = url + "&" + queryParams
	}

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed with status code %d", response.StatusCode)
	}

	waterquality := []*fiware.WaterQualityObserved{}
	err = json.NewDecoder(response.Body).Decode(&waterquality)

	return waterquality, err
}
