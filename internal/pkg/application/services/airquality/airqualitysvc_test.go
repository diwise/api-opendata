package airquality

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
)

func TestGetByID(t *testing.T) {
	is, server := testSetup(t, http.StatusOK, testData)

	ctx := context.Background()

	svc := NewAirQualityService(ctx, server.URL(), "ignored")
	svc.Start(ctx)
	defer svc.Shutdown(ctx)

	_, err := svc.Refresh(ctx)
	is.NoErr(err)

	aq, err := svc.GetByID(ctx, "urn:ngsi-ld:AirQualityObserved:test1")
	is.NoErr(err)

	aqBytes, err := json.Marshal(aq)
	is.NoErr(err)

	is.Equal(string(aqBytes), `{"id":"urn:ngsi-ld:AirQualityObserved:test1","location":{"type":"Point","coordinates":[17.472639,62.435152]},"dateObserved":{"@type":"DateTime","@value":"2023-03-12T06:23:09Z"}}`)
}

func TestGetAll(t *testing.T) {
	is, server := testSetup(t, http.StatusOK, testData)
	ctx := context.Background()

	svc := NewAirQualityService(ctx, server.URL(), "ignored")
	svc.Start(ctx)
	defer svc.Shutdown(ctx)

	_, err := svc.Refresh(ctx)
	is.NoErr(err)

	aqos := svc.GetAll(ctx)
	is.True(len(aqos) > 0)

	aqosBytes, _ := json.Marshal(aqos)

	is.Equal(string(aqosBytes), `[{"id":"urn:ngsi-ld:AirQualityObserved:test1","location":{"type":"Point","coordinates":[17.472639,62.435152]},"dateObserved":{"@type":"DateTime","@value":"2023-03-12T06:23:09Z"}}]`)
}

var Expects = testutils.Expects
var Returns = testutils.Returns
var anyInput = expects.AnyInput

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, testutils.MockService) {
	is := is.New(t)

	server := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(statusCode),
			response.ContentType("application/ld+json"),
			response.Body([]byte(responseBody)),
		),
	)

	return is, server
}

const testData string = `[
	{
		"@context": [
			"https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonl"
		],
		"CO": 500,
		"CO_Level": "moderate",
		"NO": 45,
		"NO2": 69,
		"NOx": 139,
		"SO2": 11,
		"airQualityIndex": 65,
		"airQualityLevel": "moderate",
		"dateObserved": {
			"@type": "DateTime",
			"@value": "2023-03-12T06:23:09Z"
		},
		"location": {
			"type": "Point",
			"coordinates": [17.472639,62.435152]
		},
		"id": "urn:ngsi-ld:AirQualityObserved:test1",
		"precipitation": 0,
		"relativeHumidity": 0.54,
		"reliability": 0.7,
		"source": "http://datos.madrid.es",
		"temperature": 12.2,
		"type": "AirQualityObserved",
		"windDirection": 186,
		"windSpeed": 0.64
	}
]`
