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
			time, _ := getTimestampFromBeaches(beach)
			nutskod, _ := getNutskodFromBeaches(beach)
			wiki, _ := getWikiRefFromBeaches(beach)
			beachInfo := fmt.Sprintf("\r\n%s;%s;%f;%f;%s;%s;%s;%s",
				beach.ID, beach.Name.Value, lonLat.Coordinates[0], lonLat.Coordinates[1],
				time,
				nutskod,
				wiki,
				beach.Description.Value,
			)
			beachesCsv.Write([]byte(beachInfo))
		}

		w.Header().Add("Content-Type", "text/csv")
		w.Write(beachesCsv.Bytes())
	})
}

func getNutskodFromBeaches(beach *fiware.Beach) (string, error) {
	refSeeAlso := beach.RefSeeAlso
	if refSeeAlso == nil {
		return "", fmt.Errorf("no references found in RefSeeAlso")
	}

	for _, ref := range refSeeAlso.Object {
		hovPrefix := "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/"
		if strings.Contains(ref, hovPrefix) {
			return strings.TrimPrefix(ref, hovPrefix), nil
		}
	}

	return "", fmt.Errorf("no nutskod found")
}

func getTimestampFromBeaches(beach *fiware.Beach) (string, error) {
	dateModified := beach.DateModified
	if dateModified == nil {
		return "", fmt.Errorf("dateModified is empty")
	}
	return dateModified.Value.Value, nil
}

func getWikiRefFromBeaches(beach *fiware.Beach) (string, error) {
	refSeeAlso := beach.RefSeeAlso
	if refSeeAlso == nil {
		return "", fmt.Errorf("no references found in RefSeeAlso")
	}

	for _, ref := range refSeeAlso.Object {
		wikiPrefix := "https://www.wikidata.org/wiki/"
		if strings.Contains(ref, wikiPrefix) {
			return strings.TrimPrefix(ref, wikiPrefix), nil
		}
	}

	return "", fmt.Errorf("no wikidata_ref found")
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
