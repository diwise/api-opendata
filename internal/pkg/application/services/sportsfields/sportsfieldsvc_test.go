package sportsfields

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestExpectedOutputOfGetByID(t *testing.T) {
	is, log, server := testSetup(t, http.StatusOK, testData)
	defer server.Close()

	svci := NewSportsFieldService(context.Background(), log, server.URL(), "ignored", nil)
	svc, ok := svci.(*sportsfieldSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.NoErr(err)

	sportsfield, err := svc.GetByID("urn:ngsi-ld:SportsField:se:sundsvall:facilities:3142")
	is.NoErr(err)

	sportsfieldJSON, err := json.Marshal(sportsfield)
	is.NoErr(err)

	is.Equal(expectedOutput, string(sportsfieldJSON))
}

func TestExpectedOutputOfGetAll(t *testing.T) {
	is, log, server := testSetup(t, http.StatusOK, testData)
	defer server.Close()

	svci := NewSportsFieldService(context.Background(), log, server.URL(), "ignored", nil)
	svc, ok := svci.(*sportsfieldSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.NoErr(err)

	sportsfields := svc.GetAll([]string{})

	is.Equal(len(sportsfields), 1)
}

var Expects = testutils.Expects
var Returns = testutils.Returns
var anyInput = expects.AnyInput

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, zerolog.Logger, testutils.MockService) {
	is := is.New(t)
	log := zerolog.Logger{}

	ms := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(statusCode),
			response.ContentType("application/ld+json"),
			response.Body([]byte(responseBody)),
		),
	)

	return is, log, ms
}

const testData string = `[{"@context":["https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonl"],"category":["skating","floodlit","ice-rink"],"dateCreated":{"@type":"DateTime","@value":"2022-01-25T15:37:55Z"},"dateModified":{"@type":"DateTime","@value":"2022-01-25T22:08:19Z"},"description":"Stenstans konstfrusna isbana på Stora Torget är alltid öppen för alla att åka på fram tom sportlovsveckan. Snöröjs och spolas fem gånger i veckan beroende på väder. Belysning är alltid på och musik spelas under dagtid. Fritidsbanken lånar gratis ut skridskor och hjälmar måndag-torsdag 9-21, fredag 9-18, lördag-söndag 10-18.","id":"urn:ngsi-ld:SportsField:se:sundsvall:facilities:3142","location":{"type":"MultiPolygon","coordinates":[[[[17.306436,62.390592],[17.306383,62.390501],[17.30692,62.390437],[17.306973,62.390532],[17.306436,62.390592]]]]},"name":"Stora Torget isbana","publicAccess":"after-school","source":"https://api.sundsvall.se/facilities/2.1/get/3142","type":"SportsField"}]`

const expectedOutput string = `{"id":"urn:ngsi-ld:SportsField:se:sundsvall:facilities:3142","name":"Stora Torget isbana","description":"Stenstans konstfrusna isbana på Stora Torget är alltid öppen för alla att åka på fram tom sportlovsveckan. Snöröjs och spolas fem gånger i veckan beroende på väder. Belysning är alltid på och musik spelas under dagtid. Fritidsbanken lånar gratis ut skridskor och hjälmar måndag-torsdag 9-21, fredag 9-18, lördag-söndag 10-18.","categories":["skating","floodlit","ice-rink"],"publicAccess":"after-school","location":{"type":"MultiPolygon","coordinates":[[[[17.306436,62.390592],[17.306383,62.390501],[17.30692,62.390437],[17.306973,62.390532],[17.306436,62.390592]]]]},"dateCreated":"2022-01-25T15:37:55Z","dateModified":"2022-01-25T22:08:19Z","source":"https://api.sundsvall.se/facilities/2.1/get/3142"}`
