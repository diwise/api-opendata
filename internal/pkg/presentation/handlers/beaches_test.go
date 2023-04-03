package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/beaches"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func mockBeachSvc(is *is.I) *beaches.BeachServiceMock {
	return &beaches.BeachServiceMock{
		GetAllFunc: func(ctx context.Context) []beaches.Beach {
			beaches := []beaches.Beach{}

			err := json.Unmarshal([]byte(beachesJson), &beaches)
			is.NoErr(err)

			return beaches
		},
		GetByIDFunc: func(ctx context.Context, id string) (*beaches.Beach, error) {
			beach := &beaches.Beach{}

			err := json.Unmarshal([]byte(beachByIdJson), beach)
			is.NoErr(err)

			return beach, nil
		},
	}
}

func TestGetBeachesAsGeoJSON(t *testing.T) {
	is, r, ts := setupTest(t)

	svc := mockBeachSvc(is)

	r.Get("/beaches", NewRetrieveBeachesHandler(zerolog.Logger{}, svc))
	resp, body := newGetRequest(is, ts, "application/geo+json", "/beaches?fields=waterquality,seealso", nil)

	expectation := `"geometry":{"type":"MultiPolygon","coordinates":[[[[17.47263962458644,62.435152221329254],`

	is.Equal(resp.StatusCode, http.StatusOK)
	is.True(strings.Contains(body, expectation))
}

func TestGetBeachesByID(t *testing.T) {
	is, router, server := testSetup(t)
	beachsvc := mockBeachSvc(is)

	router.Get("/{id}", NewRetrieveBeachByIDHandler(zerolog.Logger{}, beachsvc))
	resp, body := newGetRequest(is, server, "application/ld+json", "/urn:ngsi-ld:Beach:se:sundsvall:anlaggning:283", nil)

	is.Equal(resp.StatusCode, http.StatusOK)
	is.Equal(len(beachsvc.GetByIDCalls()), 1)

	is.True(strings.Contains(string(body), `"waterquality":[{"temperature":21.8,`))
}

func TestGetBeaches(t *testing.T) {
	is := is.New(t)
	beachsvc := mockBeachSvc(is)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/beaches", nil)
	NewRetrieveBeachesHandler(zerolog.Logger{}, beachsvc).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}

const beachesJson string = `[
	{
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
		"waterquality": [
			{
				"temperature": 21.8,
				"dateObserved": "2023-03-17T08:23:09Z"
			},
			{
				"temperature": 22.9,
				"dateObserved": "2023-03-20T08:23:09Z",
				"source": "acoolweatherinstituteorsomething"
			}
		],
		"seeAlso": [
			"https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/SE0712281000003473",
			"https://www.wikidata.org/wiki/Q10671745"
		]
	}]`

const beachByIdJson string = `
	{
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
		"waterquality": [
			{
				"id": "urn:ngsi-ld:WaterQualityObserved:temperature:se:servanet:lora:sk-elt-temp-14",
				"temperature": 21.8,
				"dateObserved": "2023-03-17T08:23:09Z",
				"location": {
					"type": "Point",
					"coordinates": [
						17.47263962458644,
						62.435152221329254
					]
				}
			}
		],
	  	"name": "Slädaviken",
	  	"seeAlso": [
		 	"https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/SE0712281000003473",
		  	"https://www.wikidata.org/wiki/Q10671745"
		]
	}`
