package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"

	"github.com/matryer/is"
)

var Expects = testutils.Expects
var Returns = testutils.Returns
var anyInput = expects.AnyInput

func TestCalc(t *testing.T) {
	is := is.New(t)

	v1 := 7.0
	v2 := 3.0

	w1 := time.Now().UTC().Add(-1 * time.Hour)
	w2 := time.Now().UTC()

	w := domain.Weather{
		Temperature: domain.Temperature{
			Values: &[]domain.Temperature{
				{
					Value: &v1,
					When:  &w1,
				},
				{
					Value: &v2,
					When:  &w2,
				},
			},
		},
	}

	calc(&w)

	is.Equal(5.0, *w.Temperature.Average)
	is.Equal(7.0, *w.Temperature.Max)
	is.Equal(3.0, *w.Temperature.Min)
	is.Equal(w1, *w.Temperature.From)
	is.Equal(w2, *w.Temperature.To)
}

type temporalPropertyMock struct {
	type_       string
	value_      float64
	observedAt_ string
}

func (t *temporalPropertyMock) Type() string {
	return t.type_
}
func (t *temporalPropertyMock) Value() any {
	return t.value_
}
func (t *temporalPropertyMock) ObservedAt() string {
	return t.observedAt_
}

func TestGroupByDay(t *testing.T) {
	is := is.New(t)
	props := []types.TemporalProperty{
		&temporalPropertyMock{
			type_:       "temperature",
			value_:      7.0,
			observedAt_: "2019-01-01T00:00:00Z",
		},
		&temporalPropertyMock{
			type_:       "temperature",
			value_:      3.0,
			observedAt_: "2019-01-01T01:00:00Z",
		},
		&temporalPropertyMock{
			type_:       "temperature",
			value_:      5.0,
			observedAt_: "2019-01-02T00:00:00Z",
		},
	}
	byDay := groupByTime(temporalPropertiesToTemperatureDto(props), "day")
	is.Equal(2, len(byDay))
}

func TestGetByID(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ngsi-ld/v1/entities/WeatherObserved:net:serva:iot:a81758fffe051cff":
			response := `
			{
				"@context": [
					"https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonl"
				],
				"dateObserved": {
					"type": "Property",
					"value": {
						"@type": "DateTime",
						"@value": "2023-11-10T15:04:49.990701012Z"
					}
				},
				"id": "urn:ngsi-ld:WeatherObserved:net:serva:iot:a81758fffe051cff",
				"location": {
					"type": "GeoProperty",
					"value": {
						"type": "Point",
						"coordinates": [
							17.02068,
							62.34731
						]
					}
				},
				"temperature": {
					"type": "Property",
					"value": 22.2,
					"observedAt": "2023-11-10T15:04:49.000Z"
				},
				"type": "WeatherObserved"
			}
			`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		case "/ngsi-ld/v1/temporal/entities/urn:ngsi-ld:WeatherObserved:net:serva:iot:a81758fffe051cff":
			response := `
			{
				"@context": [
					"https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld",
					"https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
				],
				"id": "urn:ngsi-ld:WeatherObserved:net:serva:iot:a81758fffe051cff",
				"temperature": [
					{
						"type": "Property",
						"value": 30,
						"observedAt": "2022-11-10T00:00:00Z"
					},
					{
						"type": "Property",
						"value": 35,
						"observedAt": "2022-11-10T01:00:00Z"
					},
					{
						"type": "Property",
						"value": 30,
						"observedAt": "2022-11-10T02:00:00Z"
					},
					{
						"type": "Property",
						"value": 20,
						"observedAt": "2023-11-10T00:00:00Z"
					},
					{
						"type": "Property",
						"value": 25,
						"observedAt": "2023-11-10T01:00:00Z"
					},
					{
						"type": "Property",
						"value": 10,
						"observedAt": "2023-11-10T02:00:00Z"
					}
				],
				"type": "WeatherObserved"
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		}
	}))
	defer server.Close()

	ws := NewWeatherService(ctx, server.URL, "default")

	w, err := ws.Query().ID("WeatherObserved:net:serva:iot:a81758fffe051cff").Aggr("year").GetByID(ctx)
	is.NoErr(err)

	is.Equal("urn:ngsi-ld:WeatherObserved:net:serva:iot:a81758fffe051cff", w.ID)
	is.Equal(2, len(*w.Temperature.Values))
}
func TestAggr(t *testing.T) {
	is := is.New(t)

	q := wsq{}
	is.Equal("", q.aggr)

	q = q.Aggr("hour").(wsq)
	is.Equal("hour", q.aggr)

	q = q.Aggr("day").(wsq)
	is.Equal("day", q.aggr)

	q = q.Aggr("month").(wsq)
	is.Equal("month", q.aggr)

	q = q.Aggr("year").(wsq)
	is.Equal("year", q.aggr)

	q = q.Aggr("invalid").(wsq)
	is.Equal("", q.aggr)
}
