package datasets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
)

const (
	NUTSCodePrefix      string = "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/"
	WikidataPrefix      string = "https://www.wikidata.org/wiki/"
	YearMonthDayISO8601 string = "2006-01-02"
)

func NewRetrieveBeachesHandler(log logging.Logger, contextBroker string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		beachesCsv := bytes.NewBufferString("place_id;name;latitude;longitude;kommunkod;updated;nutscode;wikidata;temp_url;description")

		beaches, err := getBeachesFromContextBroker(contextBroker)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("Failed to get beaches from %s: %s", contextBroker, err.Error())
			return
		}

		for _, beach := range beaches {
			lonLat := beach.Location.GetAsPoint()
			longitude := lonLat.Coordinates[0]
			latitude := lonLat.Coordinates[1]

			time := getDateModifiedFromBeach(beach)
			nutsCode := getNutsCodeFromBeach(beach)
			wiki := getWikiRefFromBeach(beach)
			beachID := strings.TrimPrefix(beach.ID, fiware.BeachIDPrefix)

			tempURL := fmt.Sprintf("%s/ngsi-ld/v1/entities?type=WaterQualityObserved&georel=near;maxDistance==1000&geometry=Point&coordinates=[%f,%f]", contextBroker, longitude, latitude)

			beachInfo := fmt.Sprintf("\r\n%s;%s;%f;%f;%s;%s;%s;%s;%s;\"%s\"",
				beachID, beach.Name.Value, latitude, longitude,
				"2281",
				time,
				nutsCode,
				wiki,
				tempURL,
				strings.ReplaceAll(beach.Description.Value, "\"", "\"\""),
			)

			beachesCsv.Write([]byte(beachInfo))
		}

		w.Header().Add("Content-Type", "text/csv")
		w.Write(beachesCsv.Bytes())
	})
}

func getNutsCodeFromBeach(beach *fiware.Beach) string {
	refSeeAlso := beach.RefSeeAlso
	if refSeeAlso == nil {
		return ""
	}

	for _, ref := range refSeeAlso.Object {

		if strings.HasPrefix(ref, NUTSCodePrefix) {
			return strings.TrimPrefix(ref, NUTSCodePrefix)
		}
	}

	return ""
}

func getDateModifiedFromBeach(beach *fiware.Beach) string {
	dateModified := beach.DateModified
	if dateModified == nil {
		return ""
	}

	timestamp, err := time.Parse(time.RFC3339, dateModified.Value.Value)
	if err != nil {
		return ""
	}

	date := timestamp.Format(YearMonthDayISO8601)

	return date
}

func getWikiRefFromBeach(beach *fiware.Beach) string {
	refSeeAlso := beach.RefSeeAlso
	if refSeeAlso == nil {
		return ""
	}

	for _, ref := range refSeeAlso.Object {

		if strings.HasPrefix(ref, WikidataPrefix) {
			return strings.TrimPrefix(ref, WikidataPrefix)
		}
	}

	return ""
}

func getBeachesFromContextBroker(host string) ([]*fiware.Beach, error) {
	response, err := http.Get(fmt.Sprintf("%s/ngsi-ld/v1/entities?type=Beach", host))
	if response.StatusCode != http.StatusOK {
		return nil, err
	}
	defer response.Body.Close()

	beaches := []*fiware.Beach{}

	err = json.NewDecoder(response.Body).Decode(&beaches)

	return beaches, err
}
