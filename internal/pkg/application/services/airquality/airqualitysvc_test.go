package airquality

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestGetByID(t *testing.T) {
	is := is.New(t)
	broker := setupMockServiceThatReturns(http.StatusOK, testData)
	defer broker.Close()

	svci := NewAirQualityService(context.Background(), zerolog.Logger{}, broker.URL, "ignored")
	svc, ok := svci.(*aqsvc)
	is.True(ok)

	err := svc.refresh()
	is.NoErr(err)

	aq, err := svc.GetByID("urn:ngsi-ld:AirQualityObserved:888100")
	is.NoErr(err)

	is.True(strings.Contains(string(aq), "urn:ngsi-ld:AirQualityObserved:888100"))
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

const testData string = `[{"@context": [
	  "https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld"
	],
	"NO": {
	  "type": "Property",
	  "value": 1.747,
	  "unitCode": "61"
	},
	"NO2": {
	  "type": "Property",
	  "value": 4.464,
	  "unitCode": "61"
	},
	"NOx": {
	  "type": "Property",
	  "value": 6.211,
	  "unitCode": "61"
	},
	"PM1": {
	  "type": "Property",
	  "value": 0.659,
	  "unitCode": "GQ"
	},
	"PM10": {
	  "type": "Property",
	  "value": 26.74,
	  "unitCode": "GQ"
	},
	"PM25": {
	  "type": "Property",
	  "value": 3.843,
	  "unitCode": "GQ"
	},
	"PM4": {
	  "type": "Property",
	  "value": 6.871,
	  "unitCode": "GQ"
	},
	"atmosphericPressure": {
	  "type": "Property",
	  "value": 1019,
	  "unitCode": "MBR"
	},
	"dateObserved": {
	  "type": "Property",
	  "value": {
		"@type": "DateTime",
		"@value": "2022-10-20T13:10:00Z"
	  }
	},
	"id": "urn:ngsi-ld:AirQualityObserved:888100",
	"location": {
	  "type": "GeoProperty",
	  "value": {
		"type": "Point",
		"coordinates": [
		  17.308968,
		  62.388618
		]
	  }
	},
	"particleCount": {
	  "type": "Property",
	  "value": 15.075,
	  "unitCode": "Particles per cm3"
	},
	"relativeHumidity": {
	  "type": "Property",
	  "value": 64.42,
	  "unitCode": "P1"
	},
	"temperature": {
	  "type": "Property",
	  "value": 9.845,
	  "unitCode": "CEL"
	},
	"totalSuspendedParticulate": {
	  "type": "Property",
	  "value": 60.597,
	  "unitCode": "GQ"
	},
	"type": "AirQualityObserved"
  },
  {
	"@context": [
	  "https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld"
	],
	"PM1": {
	  "type": "Property",
	  "value": 0.659,
	  "unitCode": "GQ"
	},
	"PM10": {
	  "type": "Property",
	  "value": 32.69,
	  "unitCode": "GQ"
	},
	"PM25": {
	  "type": "Property",
	  "value": 4.723,
	  "unitCode": "GQ"
	},
	"PM4": {
	  "type": "Property",
	  "value": 8.641,
	  "unitCode": "GQ"
	},
	"atmosphericPressure": {
	  "type": "Property",
	  "value": 1018,
	  "unitCode": "A97"
	},
	"dateObserved": {
	  "type": "Property",
	  "value": {
		"@type": "DateTime",
		"@value": "2022-10-20T13:11:00Z"
	  }
	},
	"id": "urn:ngsi-ld:AirQualityObserved:1098100",
	"location": {
	  "type": "GeoProperty",
	  "value": {
		"type": "Point",
		"coordinates": [
		  17.303442,
		  62.386485
		]
	  }
	},
	"particleCount": {
	  "type": "Property",
	  "value": 15.3,
	  "unitCode": "GQ"
	},
	"relativeHumidity": {
	  "type": "Property",
	  "value": 61.61,
	  "unitCode": "P1"
	},
	"temperature": {
	  "type": "Property",
	  "value": 10,
	  "unitCode": "CEL"
	},
	"totalSuspendedParticulate": {
	  "type": "Property",
	  "value": 76.77,
	  "unitCode": "GQ"
	},
	"type": "AirQualityObserved",
	"voltage": {
	  "type": "Property",
	  "value": 12.3,
	  "unitCode": "VLT"
	}
  }
  ]`
