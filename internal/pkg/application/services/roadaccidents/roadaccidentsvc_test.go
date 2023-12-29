package roadaccidents

import (
	"context"
	"net/http"
	"testing"

	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
)

func TestThatRefreshReturnsErrorOnNoValidHostNotFound(t *testing.T) {
	is, _, _ := testSetup(t, http.StatusOK, "")

	raSvc := NewRoadAccidentService(context.Background(), "http://lolcat:1234", "default")

	svc, ok := raSvc.(*roadAccidentSvc)
	is.True(ok)

	_, err := svc.refresh(context.Background())
	is.True(err != nil) // should return err due to invalid host
}

func TestThatRefreshFailsOnEmptyResponseBody(t *testing.T) {
	is, _, server := testSetup(t, http.StatusOK, "")

	raSvc := NewRoadAccidentService(context.Background(), server.URL(), "default")

	svc, ok := raSvc.(*roadAccidentSvc)
	is.True(ok)

	_, err := svc.refresh(context.Background())
	is.True(err != nil)
	is.Equal("failed to retrieve road accidents from context broker: failed to unmarshal response: unexpected end of JSON input", err.Error()) // should fail to unmarshal due to empty response
}

func TestThatRefreshFailsOnStatusCode400(t *testing.T) {
	is, _, server := testSetup(t, http.StatusBadRequest, "")

	raSvc := NewRoadAccidentService(context.Background(), server.URL(), "default")

	svc, ok := raSvc.(*roadAccidentSvc)
	is.True(ok)

	_, err := svc.refresh(context.Background())
	is.True(err != nil)
	is.Equal("failed to retrieve road accidents from context broker: request failed", err.Error()) // should fail on failed get request to context broker
}

func TestThatItWorks(t *testing.T) {
	is, _, server := testSetup(t, http.StatusOK, testData)

	raSvc := NewRoadAccidentService(context.Background(), server.URL(), "default")

	svc, ok := raSvc.(*roadAccidentSvc)
	is.True(ok)

	_, err := svc.refresh(context.Background())
	is.NoErr(err)
	is.Equal(len(svc.roadAccidentDetails), 2) // should be equal to 2
}

var Expects = testutils.Expects
var Returns = testutils.Returns
var anyInput = expects.AnyInput

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, context.Context, testutils.MockService) {
	is := is.New(t)
	ctx := context.Background()

	ms := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(statusCode),
			response.ContentType("application/ld+json"),
			response.Body([]byte(responseBody)),
		),
	)

	return is, ctx, ms
}

const testData string = `[
	{
		"id":"urn:ngsi-ld:RoadAccident:RoadAccident0",
		"type":"RoadAccident",
		"location":{
			"type":"Point",
			"coordinates":[
				17.0,
				62.0
			]
		},
		"dateCreated":{
			"@type":"Property",
			"@value":"2016-12-07T11:10:00Z"
		},
		"description": "RoadAccidents0",
		"accidentDate":{
			"@type":"Property",
			"@value":"2016-12-07T11:10:00Z"
		},
		"dateModified":{
			"@type":"Property",
			"@value":"2016-12-07T11:10:00Z"
		},
		"status": "ongoing"
	},
	{
		"id":"urn:ngsi-ld:RoadAccident:RoadAccident1",
		"type":"RoadAccident",
		"location":{
			"type":"Point",
			"coordinates":[
				17.1,
				62.1
			]
		},
		"dateCreated":{
			"@type":"Property",
			"@value":"2016-12-07T11:10:01Z"
		},
		"description":"RoadAccidents",
		"accidentDate":{
			"@type":"Property",
			"@value":"2022-12-07T11:10:01Z"
		},
		"dateModified":{
			"@type":"Property",
			"@value":"2022-09-07T11:10:01Z"
		},
		"status": "cleared"
	}
]
`
