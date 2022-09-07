package application

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/beaches"
	"github.com/diwise/api-opendata/internal/pkg/application/services/citywork"
	"github.com/diwise/api-opendata/internal/pkg/presentation/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/matryer/is"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func NewAppForTesting() (zerolog.Logger, *opendataAPI) {
	r := chi.NewRouter()

	return zerolog.Logger{}, newOpendataAPI(r, context.Background(), &bytes.Buffer{}, &bytes.Buffer{})
}

func NewTestRequest(is *is.I, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, _ := http.NewRequest(method, ts.URL+path, body)
	resp, _ := http.DefaultClient.Do(req)
	respBody, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	return resp, string(respBody)
}

func TestGetCitywork(t *testing.T) {
	is := is.New(t)
	server := setupMockService(http.StatusOK, "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/cityworks", nil)
	req.Header.Add("Accept", "application/json")

	cityworkSvc := citywork.NewCityworksService(context.Background(), zerolog.Logger{}, server.URL, "default")
	cityworkSvc.Start()

	handlers.NewRetrieveCityworksHandler(zerolog.Logger{}, cityworkSvc).ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}

func TestGetBeaches(t *testing.T) {
	is := is.New(t)
	server := setupMockService(http.StatusOK, beachesJson)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/beaches", nil)
	req.Header.Add("Accept", "application/json")

	beachSvc := beaches.NewBeachService(context.Background(), zerolog.Logger{}, server.URL, "default", 500)
	defer beachSvc.Shutdown()

	handlers.NewRetrieveBeachesHandler(zerolog.Logger{}, beachSvc).ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}

func TestGetWaterQuality(t *testing.T) {
	is := is.New(t)
	server := setupMockService(http.StatusOK, waterqualityJson)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/waterquality", nil)

	handlers.NewRetrieveWaterQualityHandler(zerolog.Logger{}, server.URL, "").ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}

func TestGetTrafficFlowsHandlesEmptyResult(t *testing.T) {
	is := is.New(t)
	server := setupMockService(http.StatusOK, "[]")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/trafficflow", nil)

	handlers.NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, server.URL).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)                                                                                                                                         // return code must be 200, Status OK
	is.Equal(w.Body.String(), "date_observed;road_segment;L0_CNT;L0_AVG;L1_CNT;L1_AVG;L2_CNT;L2_AVG;L3_CNT;L3_AVG;R0_CNT;R0_AVG;R1_CNT;R1_AVG;R2_CNT;R2_AVG;R3_CNT;R3_AVG") // body should only contain Csv Header
}

func TestGetTrafficFlowsHandlesSingleObservation(t *testing.T) {
	is := is.New(t)
	server := setupMockService(http.StatusOK, `[{
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
				"value": 0
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
	}]`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/trafficflow", nil)

	handlers.NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, server.URL).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)                                                                                                                                                                                                                                                           // return code must be 200, Status OK
	is.Equal(w.Body.String(), "date_observed;road_segment;L0_CNT;L0_AVG;L1_CNT;L1_AVG;L2_CNT;L2_AVG;L3_CNT;L3_AVG;R0_CNT;R0_AVG;R1_CNT;R1_AVG;R2_CNT;R2_AVG;R3_CNT;R3_AVG\r\n2016-12-07T11:10:00Z;urn:ngsi-ld:RoadSegment:19312:2860:35243;8;17.3;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0") // expected body to return values for intensity and average speed for only one observation
}

func TestGetTrafficFlowsHandlesSameDateObservations(t *testing.T) {
	is := is.New(t)
	server := setupMockService(http.StatusOK, trafficFlowJson)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/trafficflow", nil)

	handlers.NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, server.URL).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)                                                                                                                                                                                                                                                                         // return code must be 200, Status OK
	is.Equal(w.Body.String(), "date_observed;road_segment;L0_CNT;L0_AVG;L1_CNT;L1_AVG;L2_CNT;L2_AVG;L3_CNT;L3_AVG;R0_CNT;R0_AVG;R1_CNT;R1_AVG;R2_CNT;R2_AVG;R3_CNT;R3_AVG\r\n2016-12-07T11:10:00Z;urn:ngsi-ld:RoadSegment:19312:2860:35243;8;17.3;11;78.3;41;39.5;14;34.2;15;68.5;18;22.8;11;20.5;15;42.5") // expected body to return values for intensity and average speed for eight same date observations
}

func TestGetTrafficFlowsHandlesDifferentDateObservations(t *testing.T) {
	is := is.New(t)
	server := setupMockService(http.StatusOK, differentDateTfos)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/trafficflow", nil)

	handlers.NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, server.URL).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               // return code must be 200, Status OK
	is.Equal(w.Body.String(), "date_observed;road_segment;L0_CNT;L0_AVG;L1_CNT;L1_AVG;L2_CNT;L2_AVG;L3_CNT;L3_AVG;R0_CNT;R0_AVG;R1_CNT;R1_AVG;R2_CNT;R2_AVG;R3_CNT;R3_AVG\r\n2016-12-07T11:10:00Z;urn:ngsi-ld:RoadSegment:19312:2860:35243;8;17.3;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0\r\n2016-12-07T13:10:00Z;urn:ngsi-ld:RoadSegment:19312:2860:35243;0;0.0;0;0.0;0;0.0;3;25.4;0;0.0;0;0.0;0;0.0;0;0.0\r\n2016-12-07T18:10:00Z;urn:ngsi-ld:RoadSegment:19312:2860:35243;0;0.0;0;0.0;0;0.0;3;25.4;0;0.0;0;0.0;0;0.0;0;0.0") // expected body to return values for intensity and average speed for two different date observations
}

func TestGetTrafficFlowsHandlesDateObservationsFromTimeSpan(t *testing.T) {
	is := is.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		is.Equal(r.URL.RequestURI(), "/ngsi-ld/v1/entities?type=TrafficFlowObserved&timerel=between&timeAt=2016-12-07T11:10:00Z&endTimeAt=2016-12-07T13:10:00Z")

		w.Header().Add("Content-Type", "application/ld+json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/trafficflows?from=2016-12-07T11:10:00Z&to=2016-12-07T13:10:00Z", nil)

	handlers.NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, server.URL).ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK

}

func setupMockService(responseCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/ld+json")
		w.WriteHeader(responseCode)
		w.Write([]byte(responseBody))
	}))
}

const differentDateTfos string = `[{
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
			"value": 0
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
},
{
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
			"value": "2016-12-07T13:10:00Z"
		},
		"laneID": {
			"type": "Property",
			"value": 3
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 25.4
		},
		"intensity": {
			"type": "Property",
			"value": 3
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
},
{
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
			"value": "2016-12-07T18:10:00Z"
		},
		"laneID": {
			"type": "Property",
			"value": 3
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 25.4
		},
		"intensity": {
			"type": "Property",
			"value": 3
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
}]`

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
			"value": 0
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
},
{
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
			"value": 78.3
		},
		"intensity": {
			"type": "Property",
			"value": 11
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
},
{
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
			"value": 2
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 39.5
		},
		"intensity": {
			"type": "Property",
			"value": 41
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
},
{
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
			"value": 3
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 34.2
		},
		"intensity": {
			"type": "Property",
			"value": 14
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
},
{
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
			"value": 4
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 68.5
		},
		"intensity": {
			"type": "Property",
			"value": 15
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
},
{
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
			"value": 5
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 22.8
		},
		"intensity": {
			"type": "Property",
			"value": 18
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
},
{
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
			"value": 6
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 20.5
		},
		"intensity": {
			"type": "Property",
			"value": 11
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
},
{
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
			"value": 7
		},
		"averageVehicleSpeed": {
			"type": "Property",
			"value": 42.5
		},
		"intensity": {
			"type": "Property",
			"value": 15
		},
		"refRoadSegment": {
			"type": "Relationship",
			"object": ""
		}
}
]`

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
	  "dateCreated": "2018-06-21T14:47:44Z",
	  "dateModified": "2020-09-25T14:05:09Z",
	  "description": "Slädavikens havsbad är en badstrand belägen på den östra sidan av Alnön, öppen maj-augusti. Sandstranden är långgrund och badet passar därför barnfamiljer. Det finns grillplats, omklädningshytt, WC och parkering för cirka 20 bilar. Vattenprover tas.",
	  "id": "urn:ngsi-ld:Beach:se:sundsvall:anlaggning:283",
	  "location": {
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
	},
	  "name": "Slädaviken",
	  "seeAlso": [
		  "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/SE0712281000003473",
		  "https://www.wikidata.org/wiki/Q10671745"
		],
	  "type": "Beach"
	}]`
