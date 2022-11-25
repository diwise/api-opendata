package temperature

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatQueryRequiresASensor(t *testing.T) {
	is, log, svc := testSetup(t, http.StatusOK, "[]")
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Get(context.Background(), log)

	is.Equal(sensors, nil) // nil sensors should be returned
	is.True(err != nil)    // an error should be returned
}

func TestEmptyResponse(t *testing.T) {
	is, log, svc := testSetup(t, http.StatusOK, "[]")
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Sensor("testsensor").Get(context.Background(), log)

	is.NoErr(err)
	is.Equal(len(sensors[0].Temperatures), 0) // should not return any temperatures
}

func TestFailureResponse(t *testing.T) {
	is, log, svc := testSetup(t, http.StatusInternalServerError, "")
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Sensor("testsensor").Get(context.Background(), log)

	is.True(err != nil)     // should return an error
	is.True(sensors == nil) // should return a nil slice
}

func TestSingleObservationResponse(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2021-09-01T12:00:00Z")
	is, log, svc := testSetup(t, http.StatusOK, generateTestData(from, time.Hour, 12.7))
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Sensor("testsensor").Get(context.Background(), log)

	is.NoErr(err)
	is.Equal(len(sensors), 1) // should return a single temperature
}

func TestMultipleObservationResponse(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2021-09-01T12:00:00Z")
	is, log, svc := testSetup(t, http.StatusOK, generateTestData(from, time.Hour, 1.0, 1.1, 1.2, 1.3, 1.4))
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Sensor("testsensor").Get(context.Background(), log)

	is.NoErr(err)
	is.Equal(len(sensors[0].Temperatures), 5) // should return 5 temperatures
}

func TestAverageAggregationPT1H(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2021-09-01T12:00:00Z")
	is, log, svc := testSetup(t, http.StatusOK, generateTestData(from, 20*time.Minute, 1.0, 2.0, 3.0, 4.0, 5.23))
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Sensor("testsensor").Aggregate("PT1H", "avg").Get(context.Background(), log)

	is.NoErr(err)
	is.Equal(len(sensors[0].Temperatures), 2) // should return 2 temperature averages
	is.Equal(*sensors[0].Temperatures[0].Average, 2.0)
	is.Equal(*sensors[0].Temperatures[1].Average, 4.6)
}

func TestAverageAggregationPT24H(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2021-09-01T12:00:00Z")
	is, log, svc := testSetup(t, http.StatusOK, generateTestData(from, 12*time.Hour, 1.0, 2.0, 3.0, 4.0, 5.0))
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Sensor("testsensor").Aggregate("PT24H", "avg").Get(context.Background(), log)

	is.NoErr(err)
	is.Equal(len(sensors[0].Temperatures), 3) // should return 3 temperature averages
	is.Equal(*sensors[0].Temperatures[0].Average, 1.5)
	is.Equal(*sensors[0].Temperatures[1].Average, 3.5)
	is.Equal(*sensors[0].Temperatures[2].Average, 5.0)
}

func TestAverageAggregationP1MFails(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2021-09-01T12:00:00Z")
	is, log, svc := testSetup(t, http.StatusOK, generateTestData(from, time.Hour, 1.0, 2.0))
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Sensor("testsensor").Aggregate("P1M", "avg").Get(context.Background(), log)

	is.Equal(sensors, nil) // sensors should be nil
	is.True(err != nil)    // an error should be returned
}

func TestMaxMinAggregationPT1H(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2021-09-01T12:00:00Z")
	is, log, svc := testSetup(t, http.StatusOK, generateTestData(from, 20*time.Minute, 1.0, 2.0, 3.0, 4.0, 5.0))
	ts := NewTempService(svc.URL())

	sensors, err := ts.Query().Sensor("testsensor").Aggregate("PT1H", "max,min").Get(context.Background(), log)

	is.NoErr(err)
	is.Equal(len(sensors[0].Temperatures), 2)         // should return 2 temperature aggregates
	is.Equal(sensors[0].Temperatures[0].Average, nil) // should not aggregate an average
	is.Equal(*sensors[0].Temperatures[0].Min, 1.0)    // minimum value of first aggregate should be 1.0
	is.Equal(*sensors[0].Temperatures[1].Max, 5.0)    // maximum value of second aggregate should be 5.0
}

func generateTestData(from time.Time, delay time.Duration, temps ...float64) string {
	obs := from
	observations := []fiware.WeatherObserved{}

	for _, t := range temps {
		wo := fiware.NewWeatherObserved("testsensor", 23.0, 17.2, obs.Format(time.RFC3339))
		wo.Temperature = ngsitypes.NewNumberProperty(t)
		observations = append(observations, *wo)
		obs = obs.Add(delay)
	}

	bytes, _ := json.MarshalIndent(observations, " ", "  ")
	return string(bytes)
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
