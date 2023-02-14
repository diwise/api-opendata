package beaches

import (
	"context"
	"testing"

	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestXXX(t *testing.T) {
	is, log, mockSvc := testSetup(t, 200, beachesJson)

	bs := NewBeachService(context.Background(), log, mockSvc.URL(), "default", 250)
	bs.Start()
	defer bs.Shutdown()

	beaches := bs.GetAll()
	is.True(beaches != nil)
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

const beachesJson string = `[
	{
	  "dateCreated": "2018-06-21T14:47:44Z",
	  "dateModified": "2023-02-14T14:05:09Z",
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
