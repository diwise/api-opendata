package datasets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
			lonLat := beach.Location.GetAsPoint()
			time := getDateModifiedFromBeach(beach)
			nutskod := getNutsCodeFromBeach(beach)
			wiki := getWikiRefFromBeach(beach)
			beachID := strings.TrimPrefix(beach.ID, fiware.BeachIDPrefix)
			beachInfo := fmt.Sprintf("\r\n%s;%s;%f;%f;%s;%s;%s;\"%s\"",
				beachID, beach.Name.Value, lonLat.Coordinates[0], lonLat.Coordinates[1],
				time,
				nutskod,
				wiki,
				strings.ReplaceAll(beach.Description.Value, "\"", "\"\""),
			)
			beachesCsv.Write([]byte(beachInfo))
		}

		w.Header().Add("Content-Type", "text/csv")
		w.Write(beachesCsv.Bytes())
	})
}

const hovPrefix string = "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/"

func getNutsCodeFromBeach(beach *fiware.Beach) string {
	refSeeAlso := beach.RefSeeAlso
	if refSeeAlso == nil {
		return ""
	}

	for _, ref := range refSeeAlso.Object {

		if strings.HasPrefix(ref, hovPrefix) {
			return strings.TrimPrefix(ref, hovPrefix)
		}
	}

	return ""
}

const dateFormat string = "2006-01-02"

func getDateModifiedFromBeach(beach *fiware.Beach) string {
	dateModified := beach.DateModified
	if dateModified == nil {
		return ""
	}

	timestamp, err := time.Parse(time.RFC3339, dateModified.Value.Value)
	if err != nil {
		return ""
	}

	date := timestamp.Format(dateFormat)

	return date
}

const wikiPrefix string = "https://www.wikidata.org/wiki/"

func getWikiRefFromBeach(beach *fiware.Beach) string {
	refSeeAlso := beach.RefSeeAlso
	if refSeeAlso == nil {
		return ""
	}

	for _, ref := range refSeeAlso.Object {

		if strings.HasPrefix(ref, wikiPrefix) {
			return strings.TrimPrefix(ref, wikiPrefix)
		}
	}

	return ""
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
