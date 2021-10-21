package datasets

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/application/services"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/matryer/is"
)

func TestInvokeTempHandler(t *testing.T) {
	is := is.New(t)
	l := logging.NewLogger()

	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://diwise.io/api/temperatures", nil)

	svc := &services.TempServiceMock{
		GetFunc: func(time.Time, time.Time) ([]domain.Temperature, error) {
			return []domain.Temperature{}, nil
		},
	}

	NewRetrieveTemperaturesHandler(l, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK) // response status should be 200 OK
}

func TestTemperaturesFromBroker(t *testing.T) {
	is := is.New(t)
	log := logging.NewLogger()
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "diwise.io", nil)

	svc := &services.TempServiceMock{
		GetFunc: func(time.Time, time.Time) ([]domain.Temperature, error) {
			return []domain.Temperature{}, nil
		},
	}

	NewRetrieveTemperaturesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK) // response status should be 200 OK
}
