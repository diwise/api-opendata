package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/beaches"
	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestGetBeaches(t *testing.T) {
	is := is.New(t)
	ms := setupMockService(http.StatusOK, beachesJson)
	ctx := context.Background()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/beaches", nil)
	req.Header.Add("Accept", "application/json")

	wqsvc := waterquality.NewWaterQualityService(ctx, ms.URL, "default")
	wqsvc.Start(ctx)
	defer wqsvc.Shutdown(ctx)

	beachSvc := beaches.NewBeachService(ctx, ms.URL, "default", 500, wqsvc)
	beachSvc.Start(ctx)
	defer beachSvc.Shutdown(ctx)

	NewRetrieveBeachesHandler(zerolog.Logger{}, beachSvc).ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}

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
