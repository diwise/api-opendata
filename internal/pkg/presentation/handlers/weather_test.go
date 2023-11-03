package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/weather"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/matryer/is"
)

func TestInvokeTempHandler(t *testing.T) {
	is, _, rw := setup(t)
	svc, tsqm := defaultTempServiceMock()
	req, err := http.NewRequest("GET", "http://diwise.io/api/weather?coordinates=[0.0,0.0]", nil)
	is.NoErr(err)

	NewRetrieveWeatherHandler(context.Background(), svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)  // response status should be 200 OK
	is.Equal(len(tsqm.GetCalls()), 1) // Get should have been called once
}

func TestThatTimeSpanIsExtractedFromGetParameters(t *testing.T) {
	is, _, rw := setup(t)
	svc, tsqm := defaultTempServiceMock()
	from, _ := time.Parse(time.RFC3339, "2010-01-01T12:13:14Z")
	to, _ := time.Parse(time.RFC3339, "2010-01-01T22:23:24Z")
	getParams := fmt.Sprintf("?sensor=thatone&timeAt=%s&endTimeAt=%s", from.Format(time.RFC3339), to.Format(time.RFC3339))
	req, _ := http.NewRequest("GET", getParams, nil)

	NewRetrieveWeatherHandler(context.Background(), svc).ServeHTTP(rw, req)

	is.Equal(len(tsqm.BetweenTimesCalls()), 1)       // BetweenTimes should have been called once
	is.Equal(tsqm.BetweenTimesCalls()[0].From, from) // from time should match
	is.Equal(tsqm.BetweenTimesCalls()[0].To, to)     // to time should match
}

func TestThatBadStartTimeFails(t *testing.T) {
	is, _, rw := setup(t)
	svc, _ := defaultTempServiceMock()
	req, _ := http.NewRequest("GET", "?sensor=thatone&timeAt=gurka", nil)

	NewRetrieveWeatherHandler(context.Background(), svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusBadRequest) // response status should be 400 bad request
}

func TestThatFailingGetGeneratesInternalServerError(t *testing.T) {
	is, _, rw := setup(t)
	svc, tsqm := defaultTempServiceMock()
	tsqm.GetFunc = func(ctx context.Context) ([]domain.Weather, error) { return nil, errors.New("failure") }
	req, _ := http.NewRequest("GET", "?sensor=thatone", nil)

	NewRetrieveWeatherHandler(context.Background(), svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusInternalServerError) // response status should be 500 ISE
}

// #################################################

func setup(t *testing.T) (*is.I, context.Context, *httptest.ResponseRecorder) {
	return is.New(t), context.Background(), httptest.NewRecorder()
}
func defaultTempServiceMock() (*services.WeatherServiceMock, *services.WeatherServiceQueryMock) {
	tsqm := &services.WeatherServiceQueryMock{
		GetFunc: func(ctx context.Context) ([]domain.Weather, error) {
			return []domain.Weather{}, nil
		},
		NearPointFunc: func(distance int64, lat, lon float64) services.WeatherServiceQuery {
			return &services.WeatherServiceQueryMock{
			}
		},
	}

	return &services.WeatherServiceMock{
		QueryFunc: func() services.WeatherServiceQuery {
			return tsqm
		},
	}, tsqm
}
