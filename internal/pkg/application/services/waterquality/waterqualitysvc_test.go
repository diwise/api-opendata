package waterquality

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
)

func TestWaterQualityRuns(t *testing.T) {
	is, ms := testSetup(t, http.StatusOK, "[]")
	ctx := context.Background()

	wq := NewWaterQualityService(context.Background(), ms.URL(), "default")
	wq.Start(ctx)
	defer wq.Shutdown(ctx)

	_, err := wq.Refresh(ctx)

	is.NoErr(err)
}

func TestGetAll(t *testing.T) {
	is, ms := testSetup(t, http.StatusOK, waterQualityJSON)
	ctx := context.Background()

	wq := NewWaterQualityService(ctx, ms.URL(), "default")
	wq.Start(ctx)
	defer wq.Shutdown(ctx)

	refreshCount, err := wq.Refresh(ctx)
	is.NoErr(err)
	is.Equal(refreshCount, 2)

	wqos := wq.GetAll(ctx)
	is.Equal(len(wqos), refreshCount)

	wqoJson, _ := json.Marshal(wqos)
	expectation := `[{"id":"urn:ngsi-ld:WaterQualityObserved:testID","temperature":10.8,"dateObserved":"2021-05-18T19:23:09Z","location":{"type":"Point","coordinates":[17.57263982458684,62.53515242132986]}},{"id":"urn:ngsi-ld:WaterQualityObserved:testID2","temperature":10.8,"dateObserved":"2021-05-18T19:23:09Z","location":{"type":"Point","coordinates":[17.47263962458644,62.435152221329254]}}]`
	is.Equal(string(wqoJson), expectation)
}

func TestGetAllNearPointWithinTimespan(t *testing.T) {
	is, ms := testSetup(t, http.StatusOK, waterQualityJSON)
	ctx := context.Background()

	wq := NewWaterQualityService(ctx, ms.URL(), "default")
	wq.Start(ctx)
	defer wq.Shutdown(ctx)

	from, _ := time.Parse(time.RFC3339, "2021-05-17T19:23:09Z")
	to, _ := time.Parse(time.RFC3339, "2021-05-20T19:23:09Z")

	pt := NewPoint(62.43515222, 17.47263962)
	wqos, err := wq.GetAllNearPointWithinTimespan(ctx, pt, 500, from, to)
	is.NoErr(err)
	is.Equal(len(wqos), 1)

	wqoJson, _ := json.Marshal(wqos)
	expectation := `[{"id":"urn:ngsi-ld:WaterQualityObserved:testID2","temperature":10.8,"dateObserved":"2021-05-18T19:23:09Z","location":{"type":"Point","coordinates":[17.47263962458644,62.435152221329254]}}]`
	is.Equal(string(wqoJson), expectation)
}

func TestGetAllNearPointReturnsEmptyListIfNoPointsAreWithinRange(t *testing.T) {
	is, ms := testSetup(t, http.StatusOK, waterQualityJSON)
	ctx := context.Background()

	wq := NewWaterQualityService(ctx, ms.URL(), "default")
	wq.Start(ctx)

	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC()

	pt := NewPoint(0.0, 0.0)
	wqos, err := wq.GetAllNearPointWithinTimespan(ctx, pt, 500, from, to)

	is.NoErr(err)
	is.Equal(len(wqos), 0)
}

func TestGetByID(t *testing.T) {
	is, ms := testSetup(t, http.StatusOK, waterQualityJSON)
	ctx := context.Background()
	defer ms.Close()

	wq := NewWaterQualityService(ctx, ms.URL(), "default")
	wq.Start(ctx)
	defer wq.Shutdown(ctx)

	_, err := wq.Refresh(ctx)
	is.NoErr(err)

	ms2 := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(http.StatusOK),
			response.ContentType("application/ld+json"),
			response.Body([]byte(singleTemporalJSON)),
		),
	)
	defer ms2.Close()

	svc := wq.(*wqsvc)
	svc.contextBrokerURL = ms2.URL() // doing this to ensure the request in svcMock.GetByID reaches the correct response body

	wqo, err := wq.GetByID(ctx, "urn:ngsi-ld:WaterQualityObserved:testID", time.Time{}, time.Time{})
	is.NoErr(err)

	wqoJson, _ := json.Marshal(wqo)
	expectation := `{"id":"urn:ngsi-ld:WaterQualityObserved:temperature:se:servanet:lora:sk-elt-temp-02:2021-05-18T19:23:09Z","temperature":[{"value":10.8,"observedAt":"2021-05-18T19:23:09Z"}]}`
	is.Equal(string(wqoJson), expectation)
}

func TestGetByIDSortsTemporalData(t *testing.T) {
	is, ms := testSetup(t, http.StatusOK, waterQualityJSON)
	ctx := context.Background()
	defer ms.Close()

	wq := NewWaterQualityService(ctx, ms.URL(), "default")
	wq.Start(ctx)
	defer wq.Shutdown(ctx)

	_, err := wq.Refresh(ctx)
	is.NoErr(err)

	ms2 := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(http.StatusOK),
			response.ContentType("application/ld+json"),
			response.Body([]byte(multipleTemporalJSON)),
		),
	)
	defer ms2.Close()

	svc := wq.(*wqsvc)
	svc.contextBrokerURL = ms2.URL() // doing this to ensure the request in svcMock.GetByID reaches the correct response body

	wqo, err := wq.GetByID(ctx, "urn:ngsi-ld:WaterQualityObserved:testID", time.Time{}, time.Time{})
	is.NoErr(err)

	wqoJson, _ := json.Marshal(wqo)
	expectation := `"temperature":[{"value":10.8,"observedAt":"2021-05-22T15:23:09Z"},{"value":10.8,"observedAt":"2021-05-21T14:23:09Z"},{"value":10.8,"observedAt":"2021-05-20T13:23:09Z"},{"value":10.8,"observedAt":"2021-05-18T12:23:09Z"}]`
	is.True(strings.Contains(string(wqoJson), expectation))
}

func TestGetByIDWithTimespan(t *testing.T) {
	is, ms := testSetup(t, http.StatusOK, waterQualityJSON)
	ctx := context.Background()
	defer ms.Close()

	wq := NewWaterQualityService(ctx, ms.URL(), "default")
	wq.Start(ctx)
	defer wq.Shutdown(ctx)

	_, err := wq.Refresh(ctx)
	is.NoErr(err)

	ms2 := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(http.StatusOK),
			response.ContentType("application/ld+json"),
			response.Body([]byte(singleTemporalJSON)),
		),
	)
	defer ms2.Close()

	// doing this to ensure the request in svcMock.GetByID receives the correct response body
	svc := wq.(*wqsvc)
	svc.contextBrokerURL = ms2.URL()

	from, _ := time.Parse(time.RFC3339, "2021-05-18T19:23:09Z")
	to, _ := time.Parse(time.RFC3339, "2021-05-18T19:23:09Z")

	wqo, err := wq.GetByID(ctx, "urn:ngsi-ld:WaterQualityObserved:testID", from, to)
	is.NoErr(err)

	wqoJson, _ := json.Marshal(wqo)
	expectation := `{"id":"urn:ngsi-ld:WaterQualityObserved:temperature:se:servanet:lora:sk-elt-temp-02:2021-05-18T19:23:09Z","temperature":[{"value":10.8,"observedAt":"2021-05-18T19:23:09Z"}]}`
	is.Equal(string(wqoJson), expectation)
}

var Expects = testutils.Expects
var Returns = testutils.Returns
var anyInput = expects.AnyInput

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, testutils.MockService) {
	is := is.New(t)

	ms := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(statusCode),
			response.ContentType("application/ld+json"),
			response.Body([]byte(responseBody)),
		),
	)

	return is, ms
}

const singleTemporalJSON string = `{
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
	"temperature": [{
	  "type": "Property",
	  "value": 10.8,
	  "observedAt": "2021-05-18T19:23:09Z"
	}],
	"type": "WaterQualityObserved"
  }`

const multipleTemporalJSON string = `{
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
	"temperature": [{
		"type": "Property",
		"value": 10.8,
		"observedAt": "2021-05-22T15:23:09Z"
	},
	{
		"type": "Property",
		"value": 10.8,
		"observedAt": "2021-05-20T13:23:09Z"
	},
	{
		"type": "Property",
		"value": 10.8,
		"observedAt": "2021-05-18T12:23:09Z"
	},
	{
		"type": "Property",
		"value": 10.8,
		"observedAt": "2021-05-21T14:23:09Z"
	}],
	"type": "WaterQualityObserved"
  }`

const waterQualityJSON string = `[{
	"@context": [
	  "https://schema.lab.fiware.org/ld/context",
	  "https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
	],
	"dateObserved": {
		"@type": "DateTime",
		"@value": "2021-05-18T19:23:09Z"
	},
	"id": "urn:ngsi-ld:WaterQualityObserved:testID",
	"location": {
		"coordinates": [
			17.57263982458684,
			62.535152421329864
		],
		"type": "Point"
	},
	"temperature": 10.8,
	"type": "WaterQualityObserved"
  },
  {
	"@context": [
	  "https://schema.lab.fiware.org/ld/context",
	  "https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
	],
	"dateObserved": {
		"@type": "DateTime",
		"@value": "2021-05-18T19:23:09Z"
	},
	"id": "urn:ngsi-ld:WaterQualityObserved:testID2",
	"location": {
		"coordinates": [
			17.47263962458644,
			62.435152221329254
		],
		"type": "Point"
	},
	"temperature": 10.8,
	"type": "WaterQualityObserved"
  }]`
