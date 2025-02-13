package airquality

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/diwise/context-broker/pkg/ngsild"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	cbtest "github.com/diwise/context-broker/pkg/test"
	"github.com/matryer/is"
)

func TestGetByID(t *testing.T) {
	is, cbMock := testSetup(t)

	ctx := context.Background()

	svc := NewAirQualityService(ctx, cbMock, "ignored")
	svc.Start(ctx)
	defer svc.Shutdown(ctx)

	_, err := svc.Refresh(ctx)
	is.NoErr(err)

	aq, err := svc.GetByID(ctx, "urn:ngsi-ld:AirQualityObserved:test3")
	is.NoErr(err)

	aqBytes, err := json.Marshal(aq)
	is.NoErr(err)

	is.Equal(string(aqBytes), `{"id":"urn:ngsi-ld:AirQualityObserved:test3","location":{"type":"Point","coordinates":[-3.712247,40.423853]},"dateObserved":{"@type":"DateTime","@value":"2025-02-12T19:23:09Z"},"pollutants":[{"name":"Temperature","values":[{"value":12.2,"observedAt":"2025-02-12T06:23:09Z"},{"value":12.2,"observedAt":"2025-02-12T16:23:09Z"},{"value":12.2,"observedAt":"2025-02-12T19:23:09Z"}]},{"name":"RelativeHumidity","values":[{"value":0.54,"observedAt":"2025-02-12T06:23:09Z"},{"value":0.54,"observedAt":"2025-02-12T16:23:09Z"},{"value":0.54,"observedAt":"2025-02-12T19:23:09Z"}]},{"name":"CO2","values":[{"value":500,"observedAt":"2025-02-12T06:23:09Z"},{"value":500,"observedAt":"2025-02-12T16:23:09Z"},{"value":500,"observedAt":"2025-02-12T19:23:09Z"}]},{"name":"NO","values":[{"value":45,"observedAt":"2025-02-12T06:23:09Z"},{"value":45,"observedAt":"2025-02-12T16:23:09Z"},{"value":45,"observedAt":"2025-02-12T19:23:09Z"}]},{"name":"NO2","values":[{"value":69,"observedAt":"2025-02-12T06:23:09Z"},{"value":69,"observedAt":"2025-02-12T16:23:09Z"},{"value":69,"observedAt":"2025-02-12T19:23:09Z"}]},{"name":"NOx","values":[{"value":139,"observedAt":"2025-02-12T06:23:09Z"},{"value":139,"observedAt":"2025-02-12T16:23:09Z"},{"value":139,"observedAt":"2025-02-19T16:23:09Z"}]}]}`)
}

func TestGetAll(t *testing.T) {
	is, cbMock := testSetup(t)
	ctx := context.Background()

	svc := NewAirQualityService(ctx, cbMock, "ignored")
	svc.Start(ctx)
	defer svc.Shutdown(ctx)

	_, err := svc.Refresh(ctx)
	is.NoErr(err)

	aqos := svc.GetAll(ctx)
	is.True(len(aqos) > 0)

	aqosBytes, _ := json.Marshal(aqos)

	is.Equal(string(aqosBytes), `[{"id":"urn:ngsi-ld:AirQualityObserved:test3","location":{"type":"Point","coordinates":[-3.712247,40.423853]},"dateObserved":{"@type":"DateTime","@value":"2025-02-12T19:23:09Z"}}]`)
}

func testSetup(t *testing.T) (*is.I, *cbtest.ContextBrokerClientMock) {
	is := is.New(t)

	cbMock := &cbtest.ContextBrokerClientMock{
		QueryEntitiesFunc: func(ctx context.Context, entityTypes, entityAttributes []string, query string, headers map[string][]string) (*ngsild.QueryEntitiesResult, error) {
			var entts []entities.EntityImpl
			err := json.Unmarshal([]byte(testData), &entts)
			if err != nil {
				return &ngsild.QueryEntitiesResult{}, err
			}

			qer := ngsild.NewQueryEntitiesResult()
			go func() {
				for _, e := range entts {
					qer.Found <- e
				}
				qer.Found <- nil
			}()
			return qer, nil
		},
		RetrieveTemporalEvolutionOfEntityFunc: func(ctx context.Context, entityID string, headers map[string][]string, parameters ...client.RequestDecoratorFunc) (*ngsild.RetrieveTemporalEvolutionOfEntityResult, error) {
			var entity entities.EntityTemporalImpl
			json.Unmarshal([]byte(temporalJSON), &entity)

			rteer := ngsild.NewRetrieveTemporalEvolutionOfEntityResult(&entity)

			return rteer, nil
		},
	}

	return is, cbMock
}

const testData string = `[
	{
		"@context": [
			"https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld"
		],
		"CO2": {
			"type": "Property",
			"value": 500,
			"observedAt": "2025-02-12T19:23:09.000Z",
			"unitCode": "GP"
		},
		"NO": {
			"type": "Property",
			"value": 45,
			"observedAt": "2025-02-12T19:23:09.000Z",
			"unitCode": "GQ"
		},
		"NO2": {
			"type": "Property",
			"value": 69,
			"observedAt": "2025-02-12T19:23:09.000Z",
			"unitCode": "GQ"
		},
		"NOx": {
			"type": "Property",
			"value": 139,
			"observedAt": "2025-02-19T16:23:09.000Z",
			"unitCode": "GQ"
		},
		"SO2": {
			"type": "Property",
			"value": 11,
			"observedAt": "2025-02-19T16:23:09.000Z",
			"unitCode": "GQ"
		},
		"dateObserved": {
			"type": "Property",
			"value": {
				"@type": "DateTime",
				"@value": "2025-02-12T19:23:09Z"
			}
		},
		"id": "urn:ngsi-ld:AirQualityObserved:test3",
		"location": {
			"type": "GeoProperty",
			"value": {
				"type": "Point",
				"coordinates": [
					-3.712247,
					40.423853
				]
			}
		},
		"relativeHumidity": {
			"type": "Property",
			"value": 0.54,
			"observedAt": "2025-02-12T19:23:09.000Z",
			"unitCode": "P1"
		},
		"source": {
			"type": "Property",
			"value": "http://datos.madrid.es"
		},
		"temperature": {
			"type": "Property",
			"value": 12.2,
			"observedAt": "2025-02-12T19:23:09.000Z",
			"unitCode": "CEL"
		},
		"type": "AirQualityObserved"
	}
]`

const temporalJSON string = `{
	"@context": [
		"https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld",
		"https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
	],
	"CO2": [
		{
			"type": "Property",
			"value": 500,
			"observedAt": "2025-02-12T06:23:09Z",
			"unitCode": "GP"
		},
		{
			"type": "Property",
			"value": 500,
			"observedAt": "2025-02-12T16:23:09Z",
			"unitCode": "GP"
		},
		{
			"type": "Property",
			"value": 500,
			"observedAt": "2025-02-12T19:23:09Z",
			"unitCode": "GP"
		}
	],
	"NO": [
		{
			"type": "Property",
			"value": 45,
			"observedAt": "2025-02-12T06:23:09Z",
			"unitCode": "GQ"
		},
		{
			"type": "Property",
			"value": 45,
			"observedAt": "2025-02-12T16:23:09Z",
			"unitCode": "GQ"
		},
		{
			"type": "Property",
			"value": 45,
			"observedAt": "2025-02-12T19:23:09Z",
			"unitCode": "GQ"
		}
	],
	"NO2": [
		{
			"type": "Property",
			"value": 69,
			"observedAt": "2025-02-12T06:23:09Z",
			"unitCode": "GQ"
		},
		{
			"type": "Property",
			"value": 69,
			"observedAt": "2025-02-12T16:23:09Z",
			"unitCode": "GQ"
		},
		{
			"type": "Property",
			"value": 69,
			"observedAt": "2025-02-12T19:23:09Z",
			"unitCode": "GQ"
		}
	],
	"NOx": [
		{
			"type": "Property",
			"value": 139,
			"observedAt": "2025-02-12T06:23:09Z",
			"unitCode": "GQ"
		},
		{
			"type": "Property",
			"value": 139,
			"observedAt": "2025-02-12T16:23:09Z",
			"unitCode": "GQ"
		},
		{
			"type": "Property",
			"value": 139,
			"observedAt": "2025-02-19T16:23:09Z",
			"unitCode": "GQ"
		}
	],
	"SO2": [
		{
			"type": "Property",
			"value": 11,
			"observedAt": "2025-02-12T06:23:09Z",
			"unitCode": "GQ"
		},
		{
			"type": "Property",
			"value": 11,
			"observedAt": "2025-02-12T16:23:09Z",
			"unitCode": "GQ"
		},
		{
			"type": "Property",
			"value": 11,
			"observedAt": "2025-02-19T16:23:09Z",
			"unitCode": "GQ"
		}
	],
	"id": "urn:ngsi-ld:AirQualityObserved:test3",
	"relativeHumidity": [
		{
			"type": "Property",
			"value": 0.54,
			"observedAt": "2025-02-12T06:23:09Z",
			"unitCode": "P1"
		},
		{
			"type": "Property",
			"value": 0.54,
			"observedAt": "2025-02-12T16:23:09Z",
			"unitCode": "P1"
		},
		{
			"type": "Property",
			"value": 0.54,
			"observedAt": "2025-02-12T19:23:09Z",
			"unitCode": "P1"
		}
	],
	"temperature": [
		{
			"type": "Property",
			"value": 12.2,
			"observedAt": "2025-02-12T06:23:09Z",
			"unitCode": "CEL"
		},
		{
			"type": "Property",
			"value": 12.2,
			"observedAt": "2025-02-12T16:23:09Z",
			"unitCode": "CEL"
		},
		{
			"type": "Property",
			"value": 12.2,
			"observedAt": "2025-02-12T19:23:09Z",
			"unitCode": "CEL"
		}
	],
	"type": "AirQualityObserved"
}`
