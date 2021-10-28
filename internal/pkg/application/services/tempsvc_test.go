package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	"github.com/matryer/is"
)

func TestEmptyResponse(t *testing.T) {
	is := is.New(t)
	svc := setupMockServiceThatReturns(http.StatusOK, "[]")
	ts := NewTempService(svc.URL)

	temps, err := ts.Query().Get()

	is.NoErr(err)
	is.Equal(len(temps), 0) // should not return any temperatures
}

func TestFailureResponse(t *testing.T) {
	is := is.New(t)
	svc := setupMockServiceThatReturns(http.StatusInternalServerError, "")
	ts := NewTempService(svc.URL)

	temps, err := ts.Query().Get()

	is.True(err != nil)   // should return an error
	is.True(temps == nil) // should return a nil slice
}

func TestSomething(t *testing.T) {
	is := is.New(t)

	from, _ := time.Parse(time.RFC3339, "2021-09-01T12:00:00Z")
	svc := setupMockServiceThatReturns(http.StatusOK, generateTestData(from, time.Hour, 12.7, 13.2, 14.1, 9.2))

	ts := NewTempService(svc.URL)
	temps, err := ts.Query().Get()
	is.NoErr(err)
	is.Equal(len(temps), 4) // should return 4 temperatures
}

func generateTestData(from time.Time, delay time.Duration, temps ...float64) string {

	obs := from
	observations := []fiware.WeatherObserved{}

	for _, t := range temps {
		wo := fiware.NewWeatherObserved("somedevice", 23.0, 17.2, obs.Format(time.RFC3339))
		wo.Temperature = ngsitypes.NewNumberProperty(t)
		observations = append(observations, *wo)
		obs = obs.Add(delay)
	}

	bytes, _ := json.MarshalIndent(observations, " ", "  ")
	return string(bytes)
}

func setupMockServiceThatReturns(responseCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseCode)
		w.Header().Add("Content-Type", "application/ld+json")
		if body != "" {
			w.Write([]byte(body))
		}
	}))
}
