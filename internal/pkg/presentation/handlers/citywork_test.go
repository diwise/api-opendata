package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/citywork"
	"github.com/matryer/is"
)

func TestGetCitywork(t *testing.T) {
	is := is.New(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/cityworks", nil)
	req.Header.Add("Accept", "application/json")

	cityworkSvc := &citywork.CityworksServiceMock{
		GetAllFunc: func() []byte {
			return nil
		},
	}

	NewRetrieveCityworksHandler(context.Background(), cityworkSvc).ServeHTTP(w, req)
	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}
