package beaches

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
)

func mockWaterService(is *is.I) *waterquality.WaterQualityServiceMock {
	return &waterquality.WaterQualityServiceMock{
		GetAllNearPointFunc: func(ctx context.Context, pt waterquality.Point, distance int) ([]domain.WaterQuality, error) {
			dto := []waterquality.WaterQualityDTO{}
			err := json.Unmarshal([]byte(waterqualityJson), &dto)
			is.NoErr(err)

			wqos := []domain.WaterQuality{}
			for _, d := range dto {
				wq := domain.WaterQuality{
					ID:           d.ID,
					Temperature:  d.Temperature,
					Source:       d.Source,
					DateObserved: d.DateObserved.Value,
					Location:     d.Location,
				}

				wqos = append(wqos, wq)
			}

			return wqos, nil
		},
	}
}

func TestBeachServiceStartsProperly(t *testing.T) {
	is, mockBeachSvc := testSetup(t, 200, beachesJson)
	wq := mockWaterService(is)
	ctx := context.Background()

	bs := NewBeachService(ctx, mockBeachSvc.URL(), "default", 1000, wq)
	bs.Start(ctx)
	defer bs.Shutdown(ctx)

	_, err := bs.Refresh(ctx)
	is.NoErr(err)

	is.Equal(len(wq.GetAllNearPointCalls()), 2)
}

func TestBeachServiceGetsByIDContainsWaterQuality(t *testing.T) {
	is, mockBeachSvc := testSetup(t, 200, beachesJson)
	wq := mockWaterService(is)
	ctx := context.Background()

	bs := NewBeachService(ctx, mockBeachSvc.URL(), "default", 1000, wq)
	bs.Start(ctx)
	defer bs.Shutdown(ctx)

	_, err := bs.Refresh(ctx)
	is.NoErr(err)

	b, err := bs.GetByID(ctx, "urn:ngsi-ld:Beach:se:sundsvall:anlaggning:283")
	is.NoErr(err)
	is.True(b != nil)

	expectation := `"waterquality":[{"temperature":10.8,"dateObserved":"2021-05-18T19:23:09Z"}]`
	is.True(strings.Contains(string(b), expectation))
}

var Expects = testutils.Expects
var Returns = testutils.Returns
var anyInput = expects.AnyInput

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, testutils.MockService) {
	is := is.New(t)

	ms := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(statusCode),
			response.ContentType("application/ld+json"),
			response.Body([]byte(responseBody)),
		),
	)

	return is, ms
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

const waterqualityJson string = `[{
		"@context": [
		  "https://schema.lab.fiware.org/ld/context",
		  "https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
		],
		"dateObserved": {
			"@type": "DateTime",
			"@value": "2021-05-18T19:23:09Z"
		},
		"id": "urn:ngsi-ld:WaterQualityObserved:temperature:se:servanet:lora:sk-elt-temp-02:2021-05-18T19:23:09Z",
		"location": {
		  "type": "GeoProperty",
		  "value": {
			"coordinates": [
				17.473565135911233,
				62.43447998588642
			],
			"type": "Point"
		  }
		},
		"temperature": 10.8,
		"type": "WaterQualityObserved"
	  }]`
