package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/roadaccidents"
	"github.com/matryer/is"
)

func TestGetRoadAccidents(t *testing.T) {
	is := is.New(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/roadaccidents", nil)
	req.Header.Add("Accept", "application/json")

	roadAccidentSvc := &roadaccidents.RoadAccidentServiceMock{
		GetAllFunc: func() []byte { return nil },
	}

	NewRetrieveRoadAccidentsHandler(context.Background(), roadAccidentSvc).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)                 // Request failed, status code not OK
	is.Equal(len(roadAccidentSvc.GetAllCalls()), 1) // should have been called once
}
