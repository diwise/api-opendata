package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/beaches"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestGetBeachesAsGeoJSON(t *testing.T) {
	is, r, ts := setupTest(t)

	svc := mockBeachSvc(is)

	r.Get("/beaches", NewRetrieveBeachesHandler(zerolog.Logger{}, svc))
	resp, body := newGetRequest(is, ts, "application/geo+json", "/beaches?fields=waterquality,seealso", nil)

	is.Equal(resp.StatusCode, http.StatusOK)

	expectation := `{"type":"FeatureCollection", "features": [{"type":"Feature","id":"urn:ngsi-ld:Beach:se:sundsvall:anlaggning:283","geometry":{"type":"MultiPolygon","coordinates":[[[[17.47263962458644,62.435152221329254],[17.473786216873332,62.43536925656754],[17.474885857246488,62.43543825037522],[17.475474288895757,62.43457483986073],[17.474334094644085,62.43422493307671],[17.47407369318257,62.434225532314045],[17.473565135911233,62.43447998588642],[17.472995143072257,62.434936697524215],[17.47263962458644,62.435152221329254]]]]},"properties":{"location":{"coordinates":[[[[17.47263962458644,62.435152221329254],[17.473786216873332,62.43536925656754],[17.474885857246488,62.43543825037522],[17.475474288895757,62.43457483986073],[17.474334094644085,62.43422493307671],[17.47407369318257,62.434225532314045],[17.473565135911233,62.43447998588642],[17.472995143072257,62.434936697524215],[17.47263962458644,62.435152221329254]]]],"type":"MultiPolygon"},"name":"Slädaviken","seeAlso":["https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/SE0712281000003473","https://www.wikidata.org/wiki/Q10671745"],"type":"Beach","waterQuality":{"dateObserved":"2023-03-17T08:23:09Z","temperature":21.8}}}]}`
	is.Equal(body, expectation)
}

func TestGetBeachesByID(t *testing.T) {
	is, router, server := testSetup(t)
	beachsvc := mockBeachSvc(is)

	router.Get("/{id}", NewRetrieveBeachByIDHandler(zerolog.Logger{}, beachsvc))
	resp, body := newGetRequest(is, server, "application/json", "/urn:ngsi-ld:Beach:se:sundsvall:anlaggning:283", nil)

	is.Equal(resp.StatusCode, http.StatusOK)
	is.Equal(len(beachsvc.GetByIDCalls()), 1)

	const expectation string = `{"data":{"id":"urn:ngsi-ld:Beach:se:sundsvall:anlaggning:283","name":"Slädaviken","location":{"type":"MultiPolygon","coordinates":[[[[17.47263962458644,62.435152221329254],[17.473786216873332,62.43536925656754],[17.474885857246488,62.43543825037522],[17.475474288895757,62.43457483986073],[17.474334094644085,62.43422493307671],[17.47407369318257,62.434225532314045],[17.473565135911233,62.43447998588642],[17.472995143072257,62.434936697524215],[17.47263962458644,62.435152221329254]]]]},"waterquality":[{"temperature":21.8,"dateObserved":"2023-03-17T08:23:09Z"}],"description":"Slädavikens havsbad är en badstrand belägen på den östra sidan av Alnön, öppen maj-augusti. Sandstranden är långgrund och badet passar därför barnfamiljer. Det finns grillplats, omklädningshytt, WC och parkering för cirka 20 bilar. Vattenprover tas.","seeAlso":["https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/SE0712281000003473","https://www.wikidata.org/wiki/Q10671745"]}}`
	is.Equal(body, expectation)
}

func TestGetBeaches(t *testing.T) {
	is, router, ts := testSetup(t)
	svc := mockBeachSvc(is)

	router.Get("/beaches", NewRetrieveBeachesHandler(zerolog.Logger{}, svc))
	resp, body := newGetRequest(is, ts, "application/json", "/beaches?fields=waterquality", nil)

	is.Equal(resp.StatusCode, http.StatusOK) // Request failed, status code not OK

	const expectation string = `{"data":[{"id":"urn:ngsi-ld:Beach:se:sundsvall:anlaggning:283","location":{"type":"Point","coordinates":[17.47263962458644,62.435152221329254]},"name":"Slädaviken","waterQuality":{"temperature":21.8,"dateObserved":"2023-03-17T08:23:09Z"}}]}`
	is.Equal(body, expectation)
}

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
