package application

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/datasets"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/repositories/database"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestThatRetrieveCatalogsSucceeds(t *testing.T) {
	log := logging.NewLogger()
	db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(), log)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost:8080/catalogs", nil)

	NewRetrieveCatalogsHandler(log, db).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", w.Code)
	}
}

func TestGetBeaches(t *testing.T) {

	log := logging.NewLogger()

	server := setupMockService(http.StatusOK, beachesJson)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/beaches", nil)

	datasets.NewRetrieveBeachesHandler(log, server.URL).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", w.Code)
	}

	fmt.Println(w.Body.String())
}

func TestGetWaterQuality(t *testing.T) {
	log := logging.NewLogger()

	server := setupMockService(http.StatusOK, waterqualityJson)

	nr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/waterquality", nil)

	datasets.NewRetrieveWaterQualityHandler(log, server.URL, "").ServeHTTP(nr, req)
	if nr.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", nr.Code)
	}
}

func TestGetTrafficFlows(t *testing.T) {
	log := logging.NewLogger()

	server := setupMockService(http.StatusOK, trafficFlowJson)

	nr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/trafficflow", nil)

	datasets.NewRetrieveTrafficFlowsHandler(log, server.URL).ServeHTTP(nr, req)
	if nr.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", nr.Code)
	}

	log.Infof(nr.Body.String())
}

func setupMockService(responseCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/ld+json")
		w.WriteHeader(responseCode)
		w.Write([]byte(responseBody))
	}))
}

const trafficFlowJson string = `[{
    "@context": [
      "https://schema.lab.fiware.org/ld/context",
      "https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
    ],
    "id": "urn:ngsi-ld:TrafficFlowObserved:sn-tcr-01:test",
    "type": "TrafficFlowObserved",
		"location": {
			"type": "GeoProperty",
			"value": {
				"coordinates": [
					17.0,
					62.2
				],
			"type": "Point"
			}
		},
		"dateObserved": {
			"type": "Property",
			"value": "2016-12-07T11:10:00Z"
		},
		"laneID": {
			"type": "Property",
			"value": 1
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 17.3
		},
		"intensity": {
			"type": "Property",
			"value": 8
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
}]`

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
	"temperature": {
	  "type": "Property",
	  "value": 10.8
	},
	"type": "WaterQualityObserved"
  }]`

const beachesJson string = `[
	{
	  "@context": [
		"https://schema.lab.fiware.org/ld/context",
		"https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
	  ],
	  "dateCreated": {
		"type": "Property",
		"value": {
		  "@type": "DateTime",
		  "@value": "2018-06-21T14:47:44Z"
		}
	  },
	  "dateModified": {
		"type": "Property",
		"value": {
		  "@type": "DateTime",
		  "@value": "2020-09-25T14:05:09Z"
		}
	  },
	  "description": {
		"type": "Property",
		"value": "Slädavikens havsbad är en badstrand belägen på den östra sidan av Alnön, öppen maj-augusti. Sandstranden är långgrund och badet passar därför barnfamiljer. Det finns grillplats, omklädningshytt, WC och parkering för cirka 20 bilar. Vattenprover tas."
	  },
	  "id": "urn:ngsi-ld:Beach:se:sundsvall:anlaggning:283",
	  "location": {
		"type": "GeoProperty",
		"value": {
		  "coordinates": [
			[
			  [
				[
				  17.47263962458644,
				  62.435152221329254
				],
				[
				  17.473786216873332,
				  62.43536925656754
				],
				[
				  17.474885857246488,
				  62.43543825037522
				],
				[
				  17.475474288895757,
				  62.43457483986073
				],
				[
				  17.474334094644085,
				  62.43422493307671
				],
				[
				  17.47407369318257,
				  62.434225532314045
				],
				[
				  17.473565135911233,
				  62.43447998588642
				],
				[
				  17.472995143072257,
				  62.434936697524215
				],
				[
				  17.47263962458644,
				  62.435152221329254
				]
			  ]
			]
		  ],
		  "type": "MultiPolygon"
		}
	  },
	  "name": {
		"type": "Property",
		"value": "Slädaviken"
	  },
	  "refSeeAlso": {
		"object": [
		  "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/SE0712281000003473",
		  "https://www.wikidata.org/wiki/Q10671745"
		],
		"type": "Relationship"
	  },
	  "type": "Beach"
	}]`
