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
		GetByIDFunc: func(ctx context.Context, id string) (*beaches.BeachDetails, error) {
			beach := &beaches.BeachDetails{}

			err := json.Unmarshal([]byte(beachByIdJson), beach)
			is.NoErr(err)

			return beach, nil
		},
	}
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
				17.47263962458644,
				62.435152221329254
			],
			"type": "MultiPolygon"
		},
		"name": "Slädaviken",
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
				17.47263962458644,
				62.435152221329254
			],
			"type": "Point"
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
