package datasets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"
)

func NewRetrieveBeachesHandler(log logging.Logger, contextBroker string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		beachesCsv := bytes.NewBufferString("place_id;name;latitude;longitude;updated;nuts_code;wikidata_ref;description")

		beaches, err := getBeachesFromContextBroker(contextBroker)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("Failed to get beaches from %s: %s", contextBroker, err.Error())
			return
		}

		for _, beach := range beaches {
			beachInfo := fmt.Sprintf("\r\n%s;%s;%f;%f;%s;%s;%s;%s",
				beach.ID, beach.Name.Value, 65.2, 17.1,
				"2021-04-28",
				"nuts-kod",
				"Q16498519",
				beach.Description.Value,
			)
			beachesCsv.Write([]byte(beachInfo))
		}

		w.Header().Add("Content-Type", "text/csv")
		w.Write(beachesCsv.Bytes())
	})

}

func getBeachesFromContextBroker(host string) ([]*fiware.Beach, error) {
	response, err := http.Get(fmt.Sprintf("http://%s/ngsi-ld/v1/entities?type=Beach", host))
	if response.StatusCode != http.StatusOK {
		return nil, err
	}
	defer response.Body.Close()

	beaches := []*fiware.Beach{}

	json.NewDecoder(response.Body).Decode(&beaches)

	return beaches, err
}
