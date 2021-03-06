package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/temperature"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestInvokeTempHandler(t *testing.T) {
	is, log, rw := setup(t)
	svc, tsqm := defaultTempServiceMock()
	req, err := http.NewRequest("GET", "http://diwise.io/api/temperature/air?sensor=thatone", nil)
	is.NoErr(err)

	NewRetrieveTemperaturesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)  // response status should be 200 OK
	is.Equal(len(tsqm.GetCalls()), 1) // Get should have been called once
}

func TestThatSensorValueIsExtractedFromGetParameters(t *testing.T) {
	is, log, rw := setup(t)
	svc, tsqm := defaultTempServiceMock()
	req, _ := http.NewRequest("GET", "?sensor=thesensor", nil)

	NewRetrieveTemperaturesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(len(tsqm.SensorCalls()), 1)                // Sensor should have been called once
	is.Equal(tsqm.SensorCalls()[0].Sensor, "thesensor") // sensor id should match
}

func TestThatTimeSpanIsExtractedFromGetParameters(t *testing.T) {
	is, log, rw := setup(t)
	svc, tsqm := defaultTempServiceMock()
	from, _ := time.Parse(time.RFC3339, "2010-01-01T12:13:14Z")
	to, _ := time.Parse(time.RFC3339, "2010-01-01T22:23:24Z")
	getParams := fmt.Sprintf("?sensor=thatone&timeAt=%s&endTimeAt=%s", from.Format(time.RFC3339), to.Format(time.RFC3339))
	req, _ := http.NewRequest("GET", getParams, nil)

	NewRetrieveTemperaturesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(len(tsqm.BetweenTimesCalls()), 1)       // BetweenTimes should have been called once
	is.Equal(tsqm.BetweenTimesCalls()[0].From, from) // from time should match
	is.Equal(tsqm.BetweenTimesCalls()[0].To, to)     // to time should match
}

func TestThatAggregationSettingsAreExtractedFromGetParameters(t *testing.T) {
	is, log, rw := setup(t)
	svc, tsqm := defaultTempServiceMock()
	req, _ := http.NewRequest("GET", "?sensor=thatone&aggrMethods=avg,max,min&aggrPeriodDuration=P2H&options=aggregatedValues", nil)

	NewRetrieveTemperaturesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(len(tsqm.AggregateCalls()), 1) // Aggregate should have been called once
	is.Equal(tsqm.AggregateCalls()[0].Aggregates, "avg,max,min")
	is.Equal(tsqm.AggregateCalls()[0].Period, "P2H")
}

func TestThatBadStartTimeFails(t *testing.T) {
	is, log, rw := setup(t)
	svc, _ := defaultTempServiceMock()
	req, _ := http.NewRequest("GET", "?sensor=thatone&timeAt=gurka", nil)

	NewRetrieveTemperaturesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusBadRequest) // response status should be 400 bad request
}

func TestThatFailingGetGeneratesInternalServerError(t *testing.T) {
	is, log, rw := setup(t)
	svc, tsqm := defaultTempServiceMock()
	tsqm.GetFunc = func(context.Context, zerolog.Logger) ([]domain.Sensor, error) { return nil, errors.New("failure") }
	req, _ := http.NewRequest("GET", "?sensor=thatone", nil)

	NewRetrieveTemperaturesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusInternalServerError) // response status should be 500 ISE
}

// #################################################

func TestInvokeTempSensorsHandler(t *testing.T) {
	is, log, rw := setup(t)
	//svc := setupMockServiceThatReturns(http.StatusOK, "[]")
	req, _ := http.NewRequest("GET", "http://diwise.io/api/temperature/air/sensors", nil)

	NewRetrieveTemperatureSensorsHandler(log /*svc.URL*/, "https://diwise.io").ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK) // response status should be 200 OK
}

// #################################################

func setup(t *testing.T) (*is.I, zerolog.Logger, *httptest.ResponseRecorder) {
	return is.New(t), zerolog.Logger{}, httptest.NewRecorder()
}

/*func setupMockServiceThatReturns(responseCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseCode)
		w.Header().Add("Content-Type", "application/ld+json")
		if body != "" {
			w.Write([]byte(body))
		}
	}))
}*/

func defaultTempServiceMock() (*services.TempServiceMock, *services.TempServiceQueryMock) {
	tsqm := &services.TempServiceQueryMock{
		GetFunc: func(ctx context.Context, log zerolog.Logger) ([]domain.Sensor, error) {
			return []domain.Sensor{}, nil
		},
	}

	return &services.TempServiceMock{
		QueryFunc: func() services.TempServiceQuery {
			return tsqm
		},
	}, tsqm
}
