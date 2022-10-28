package sportsfields

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestExpectedOutputOfGetByID(t *testing.T) {
	is := is.New(t)
	broker := setupMockServiceThatReturns(http.StatusOK, testData)
	defer broker.Close()

	svci := NewSportsFieldService(context.Background(), zerolog.Logger{}, broker.URL, "ignored")
	svc, ok := svci.(*sportsfieldSvc)
	is.True(ok)

	err := svc.refresh()
	is.NoErr(err)

	sportsfield, err := svc.GetByID("urn:ngsi-ld:SportsField:se:sundsvall:facilities:796")
	is.NoErr(err)

	sportsfieldJSON, err := json.Marshal(sportsfield)

	is.Equal(expectedOutput, string(sportsfieldJSON))
}

func TestExpectedOutputOfGetAll(t *testing.T) {
	is := is.New(t)
	broker := setupMockServiceThatReturns(http.StatusOK, testData)
	defer broker.Close()

	svci := NewSportsFieldService(context.Background(), zerolog.Logger{}, broker.URL, "ignored")
	svc, ok := svci.(*sportsfieldSvc)
	is.True(ok)

	err := svc.refresh()
	is.NoErr(err)

	sportsfields := svc.GetAll()

	is.Equal(len(sportsfields), 1)
}

func setupMockServiceThatReturns(responseCode int, body string, headers ...func(w http.ResponseWriter)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, applyHeaderTo := range headers {
			applyHeaderTo(w)
		}

		w.WriteHeader(responseCode)

		if body != "" {
			w.Write([]byte(body))
		}
	}))
}

const testData string = `[{"@context":["https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld"],"id":"urn:ngsi-ld:SportsField:se:sundsvall:facilities:796","category":{"type":"Property","value":["skating","floodlit","ice-rink"]},"dateCreated":{"type":"Property","value":{"@type":"DateTime","@value":"2019-10-15T16:15:32Z"}},"dateModified":{"type":"Property","value":{"@type":"DateTime","@value":"2021-12-17T16:54:02Z"}},"description":{"type":"Property","value":"7-manna grusplan intill skolan. Vintertid spolas och snöröjs isbanan en gång i veckan."},"location":{"type":"GeoProperty","value":{"type":"MultiPolygon","coordinates":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]}},"name":{"type":"Property","value":"Skolans grusplan och isbana"},"source":{"type":"Property","value":"http://127.0.0.1:60519/get/796"},"type":"SportsField"}]`

const expectedOutput string = `{"id":"urn:ngsi-ld:SportsField:se:sundsvall:facilities:796","name":"Skolans grusplan och isbana","description":"7-manna grusplan intill skolan. Vintertid spolas och snöröjs isbanan en gång i veckan.","categories":["skating","floodlit","ice-rink"],"geometry":{"type":"MultiPolygon","coordinates":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]},"dateCreated":"2019-10-15T16:15:32Z","dateModified":"2021-12-17T16:54:02Z","source":{"type":"Property","value":"http://127.0.0.1:60519/get/796"}}`
