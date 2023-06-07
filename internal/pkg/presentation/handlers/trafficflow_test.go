package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestGetTrafficFlowsHandlesEmptyResult(t *testing.T) {
	is := is.New(t)
	ms := setupMockService(http.StatusOK, "[]")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/trafficflow", nil)

	NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, ms.URL).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)                                                                                                                                         // return code must be 200, Status OK
	is.Equal(w.Body.String(), "date_observed;road_segment;L0_CNT;L0_AVG;L1_CNT;L1_AVG;L2_CNT;L2_AVG;L3_CNT;L3_AVG;R0_CNT;R0_AVG;R1_CNT;R1_AVG;R2_CNT;R2_AVG;R3_CNT;R3_AVG") // body should only contain Csv Header
}

func TestGetTrafficFlowsHandlesSingleObservation(t *testing.T) {
	is := is.New(t)
	ms := setupMockService(http.StatusOK, `[{
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

	NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, ms.URL).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)                                                                                                                                                                                                                                                           // return code must be 200, Status OK
	is.Equal(w.Body.String(), "date_observed;road_segment;L0_CNT;L0_AVG;L1_CNT;L1_AVG;L2_CNT;L2_AVG;L3_CNT;L3_AVG;R0_CNT;R0_AVG;R1_CNT;R1_AVG;R2_CNT;R2_AVG;R3_CNT;R3_AVG\r\n2016-12-07T11:10:00Z;urn:ngsi-ld:RoadSegment:19312:2860:35243;8;17.3;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0;0;0.0") // expected body to return values for intensity and average speed for only one observation
}

func TestGetTrafficFlowsHandlesSameDateObservations(t *testing.T) {
	is := is.New(t)
	ms := setupMockService(http.StatusOK, trafficFlowJson)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/trafficflow", nil)

	NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, ms.URL).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)                                                                                                                                                                                                                                                                         // return code must be 200, Status OK
	is.Equal(w.Body.String(), "date_observed;road_segment;L0_CNT;L0_AVG;L1_CNT;L1_AVG;L2_CNT;L2_AVG;L3_CNT;L3_AVG;R0_CNT;R0_AVG;R1_CNT;R1_AVG;R2_CNT;R2_AVG;R3_CNT;R3_AVG\r\n2016-12-07T11:10:00Z;urn:ngsi-ld:RoadSegment:19312:2860:35243;8;17.3;11;78.3;41;39.5;14;34.2;15;68.5;18;22.8;11;20.5;15;42.5") // expected body to return values for intensity and average speed for eight same date observations
}

func TestGetTrafficFlowsHandlesDifferentDateObservations(t *testing.T) {
	is := is.New(t)
	ms := setupMockService(http.StatusOK, differentDateTfos)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/trafficflow", nil)

	NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, ms.URL).ServeHTTP(w, req)

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

	NewRetrieveTrafficFlowsHandler(zerolog.Logger{}, server.URL).ServeHTTP(w, req)
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
