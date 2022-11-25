package citywork

import (
	"context"
	"net/http"
	"testing"

	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatRefreshReturnsErrorOnNoValidHostNotFound(t *testing.T) {
	is, log, _ := testSetup(t, http.StatusOK, "")

	cwSvc := NewCityworksService(context.Background(), log, "http://lolcat:1234", "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.True(err != nil) // should return err due to invalid host
}

func TestThatRefreshFailsOnEmptyResponseBody(t *testing.T) {
	is, log, server := testSetup(t, http.StatusOK, "")

	cwSvc := NewCityworksService(context.Background(), log, server.URL(), "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.True(err != nil)
	is.Equal("failed to retrieve cityworks from context broker: failed to unmarshal response: unexpected end of JSON input", err.Error()) // should fail to unmarshal due to empty response
}

func TestThatRefreshFailsOnStatusCode400(t *testing.T) {
	is, log, server := testSetup(t, http.StatusBadRequest, "")

	cwSvc := NewCityworksService(context.Background(), log, server.URL(), "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.True(err != nil)
	is.Equal("failed to retrieve cityworks from context broker: request failed", err.Error()) // should fail on failed get request to context broker
}

func TestThatItWorks(t *testing.T) {
	is, log, server := testSetup(t, http.StatusOK, testData)

	cwSvc := NewCityworksService(context.Background(), log, server.URL(), "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	_, err := svc.refresh()
	is.NoErr(err)
	is.Equal(len(svc.cityworksDetails), 2) // should be equal to 2
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
		"description": "cityworks0",
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
		"description": "cityworks",
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
