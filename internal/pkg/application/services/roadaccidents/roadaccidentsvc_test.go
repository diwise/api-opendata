package roadaccidents

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatRefreshReturnsErrorOnNoValidHostNotFound(t *testing.T) {
	is, log, _ := testSetup(t, 200, "")

	raSvc := NewRoadAccidentService(context.Background(), log, "http://lolcat:1234", "default")

	svc, ok := raSvc.(*roadAccidentSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.True(err != nil) // should return err due to invalid host
}

func TestThatRefreshFailsOnEmptyResponseBody(t *testing.T) {
	is, log, server := testSetup(t, 200, "")

	raSvc := NewRoadAccidentService(context.Background(), log, server.URL, "default")

	svc, ok := raSvc.(*roadAccidentSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.True(err != nil)
	is.Equal("failed to retrieve road accidents from context broker: failed to unmarshal response: unexpected end of JSON input", err.Error()) // should fail to unmarshal due to empty response
}

func TestThatRefreshFailsOnStatusCode400(t *testing.T) {
	is, log, server := testSetup(t, 400, "")

	raSvc := NewRoadAccidentService(context.Background(), log, server.URL, "default")

	svc, ok := raSvc.(*roadAccidentSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.True(err != nil)
	is.Equal("failed to retrieve road accidents from context broker: request failed", err.Error()) // should fail on failed get request to context broker
}

func TestThatItWorks(t *testing.T) {
	is, log, server := testSetup(t, 200, testData)

	raSvc := NewRoadAccidentService(context.Background(), log, server.URL, "default")

	svc, ok := raSvc.(*roadAccidentSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.NoErr(err)
	is.Equal(len(svc.roadAccidentDetails), 2) // should be equal to 2
}

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, zerolog.Logger, *httptest.Server) {
	is := is.New(t)
	log := zerolog.Logger{}
	server := setupMockServiceThatReturns(statusCode, responseBody)

	return is, log, server
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

func setupMockServiceThatReturns(responseCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseCode)
		w.Header().Add("Content-Type", "application/ld+json")
		if body != "" {
			w.Write([]byte(body))
		}
	}))
}
