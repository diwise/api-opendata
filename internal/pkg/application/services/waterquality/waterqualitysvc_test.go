package waterquality

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

func TestWaterQualityRuns(t *testing.T) {
	is, log, svc := testSetup(t, http.StatusOK, "[]")
	wq := NewWaterQualityService(context.Background(), log, svc.URL(), "default")

	svcMock := wq.(*wqsvc)

	err := svcMock.refresh()

	is.NoErr(err)
}

func TestGetAll(t *testing.T) {
	is, log, svc := testSetup(t, http.StatusOK, waterqualityJson)
	wq := NewWaterQualityService(context.Background(), log, svc.URL(), "default")

	svcMock := wq.(*wqsvc)

	err := svcMock.refresh()
	is.NoErr(err)

	wqos := svcMock.GetAll()
	is.True(wqos != nil)

	wqoJson, _ := json.Marshal(wqos)

	expectation := `[{"id":"urn:ngsi-ld:WaterQualityObserved:temperature:se:servanet:lora:sk-elt-temp-02:2021-05-18T19:23:09Z","location":{"type":"GeoProperty","value":{"type":"Point","coordinates":[17.39364,62.297684]}},"temperature":[{"value":10.8,"observedAt":"2021-05-18T19:23:09Z"}],"dateObserved":{"type":"Property","value":{"@type":"DateTime","@value":"2021-05-18T19:23:09Z"}}}]`

	is.Equal(string(wqoJson), expectation)
}

func TestGetAllNearPoint(t *testing.T) {
	is, log, svc := testSetup(t, http.StatusOK, waterqualityJson)

	wq := NewWaterQualityService(context.Background(), log, svc.URL(), "default")

	svcMock := wq.(*wqsvc)

	err := svcMock.refresh()
	is.NoErr(err)

	pt := NewPoint(62.297684, 17.39364)
	wqos, err := svcMock.GetAllNearPoint(pt, 500)
	is.NoErr(err)
	is.True(wqos != nil)

	wqoJson, _ := json.Marshal(wqos)

	expectation := `[{"id":"urn:ngsi-ld:WaterQualityObserved:temperature:se:servanet:lora:sk-elt-temp-02:2021-05-18T19:23:09Z","location":{"type":"GeoProperty","value":{"type":"Point","coordinates":[17.39364,62.297684]}},"temperature":[{"value":10.8,"observedAt":"2021-05-18T19:23:09Z"}],"dateObserved":{"type":"Property","value":{"@type":"DateTime","@value":"2021-05-18T19:23:09Z"}}}]`

	is.Equal(string(wqoJson), expectation)
}

func TestGetAllNearPointReturnsErrorIfNoPointsAreWithinRange(t *testing.T) {
	is, log, svc := testSetup(t, http.StatusOK, waterqualityJson)

	wq := NewWaterQualityService(context.Background(), log, svc.URL(), "default")

	svcMock := wq.(*wqsvc)

	err := svcMock.refresh()
	is.NoErr(err)

	pt := NewPoint(0.0, 0.0)
	wqos, err := svcMock.GetAllNearPoint(pt, 500)
	is.True(err != nil)
	is.Equal(len(*wqos), 0)

	wqoJson, _ := json.Marshal(wqos)

	expectation := `[]`

	is.Equal(string(wqoJson), expectation)
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

const waterqualityJson string = `[{
	"@context": [
	  "https://schema.lab.fiware.org/ld/context",
	  "https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
	],
	"dateObserved": {
	  "type": "Property",
	  "value": {
		"@type": "DateTime",
		"@value": "2021-05-18T19:23:09Z"
	  }
	},
	"id": "urn:ngsi-ld:WaterQualityObserved:temperature:se:servanet:lora:sk-elt-temp-02:2021-05-18T19:23:09Z",
	"location": {
	  "type": "GeoProperty",
	  "value": {
		"coordinates": [
		  17.39364,
		  62.297684
		],
		"type": "Point"
	  }
	},
	"refDevice": {
	  "object": "urn:ngsi-ld:Device:temperature:se:servanet:lora:sk-elt-temp-02",
	  "type": "Relationship"
	},
	"temperature": [{
	  "type": "Property",
	  "value": 10.8,
	  "observedAt": "2021-05-18T19:23:09Z"
	}],
	"type": "WaterQualityObserved"
  }]`
