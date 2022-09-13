package citywork

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

	cwSvc := NewCityworksService(context.Background(), log, "http://lolcat:1234", "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	err := svc.refresh()
	is.True(err != nil) // should return err due to invalid host
}

func TestThatRefreshFailsOnEmptyResponseBody(t *testing.T) {
	is, log, server := testSetup(t, 200, "")

	cwSvc := NewCityworksService(context.Background(), log, server.URL, "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	err := svc.refresh()
	is.True(err != nil)
	is.Equal("failed to unmarshal response: unexpected end of JSON input", err.Error()) // should fail to unmarshal due to empty response
}

func TestThatRefreshFailsOnStatusCode400(t *testing.T) {
	is, log, server := testSetup(t, 400, "")

	cwSvc := NewCityworksService(context.Background(), log, server.URL, "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	err := svc.refresh()
	is.True(err != nil)
	is.Equal("request failed", err.Error()) // should fail on failed get request to context broker
}

func TestThatItWorks(t *testing.T) {
	is, log, server := testSetup(t, 200, testData)

	cwSvc := NewCityworksService(context.Background(), log, server.URL, "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	err := svc.refresh()
	is.NoErr(err)
	is.Equal(len(svc.cityworksDetails), 2) // should be equal to 2
}

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, zerolog.Logger, *httptest.Server) {
	is := is.New(t)
	log := zerolog.Logger{}
	server := setupMockServiceThatReturns(statusCode, responseBody)

	return is, log, server
}

const testData string = `[
	{
		"id":"urn:ngsi-ld:CityWork:citywork0",
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
		"description":{
			"type":"Property",
			"value":"cityworks0"
		},
		"startDate":{
			"@type":"Property",
			"@value":"2016-12-07T11:10:00Z"
		},
		"endDate":{
			"@type":"Property",
			"@value":"2016-12-07T11:10:00Z"
		}
	},
	{
		"id":"urn:ngsi-ld:CityWork:citywork1",
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
		"description":{
			"type":"Property",
			"value":"cityworks"
		},
		"startDate":{
			"@type":"Property",
			"@value":"2022-12-07T11:10:01Z"
		},
		"endDate":{
			"@type":"Property",
			"@value":"2022-09-07T11:10:01Z"
		}
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
