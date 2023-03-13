package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestGetWaterQuality(t *testing.T) {
	is, log, server := testSetup(t, http.StatusOK, waterqualityJson)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/waterquality", nil)

	wqSvc := waterquality.NewWaterQualityService(context.Background(), log, server.URL(), "default")
	wqSvc.Start()

	NewRetrieveWaterQualityHandler(log, wqSvc).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}

func TestGetWaterQualityByID(t *testing.T) {
	is, log, server := testSetup(t, http.StatusOK, waterqualityJson)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/waterquality", nil)

	wqSvc := waterquality.NewWaterQualityService(context.Background(), log, server.URL(), "default")
	wqSvc.Start()

	NewRetrieveWaterQualityByIDHandler(log, wqSvc).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, zerolog.Logger, testutils.MockService) {
	is := is.New(t)
	log := zerolog.Logger{}

	ms := testutils.NewMockServiceThat(
		testutils.Expects(is, expects.AnyInput()),
		testutils.Returns(
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
