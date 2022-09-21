package handlers

import (
	"io"
	"net/http"
	"strings"
	"testing"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/exercisetrails"
	"github.com/diwise/api-opendata/internal/pkg/domain"
)

func TestInvokeExerciseTrailsHandler(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultTrailsMock()
	req, err := http.NewRequest("GET", "", nil)
	is.NoErr(err)

	NewRetrieveExerciseTrailsHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)    // response status should be 200 OK
	is.Equal(len(svc.GetAllCalls()), 1) // Get should have been called once
}

func TestGetExerciseTrailsDoesNotContainDescriptionByDefault(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultTrailsMock()
	req, err := http.NewRequest("GET", "", nil)
	is.NoErr(err)

	NewRetrieveExerciseTrailsHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)    // response status should be 200 OK
	is.Equal(len(svc.GetAllCalls()), 1) // Get should have been called once

	response, err := io.ReadAll(rw.Body)
	is.NoErr(err)

	is.True(!strings.Contains(string(response), "description"))
}

func TestGetExerciseTrailsWithDescription(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultTrailsMock()
	req, err := http.NewRequest("GET", "?fields=description", nil)
	is.NoErr(err)

	NewRetrieveExerciseTrailsHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)    // response status should be 200 OK
	is.Equal(len(svc.GetAllCalls()), 1) // Get should have been called once

	response, err := io.ReadAll(rw.Body)
	is.NoErr(err)

	const expectedResponse string = `{"data":[{"categories":["bike-track"],"description":"this is a description","id":"trail","length":7,"name":"test0"}]}`
	is.Equal(string(response), expectedResponse)
}

func defaultTrailsMock() *services.ExerciseTrailServiceMock {
	mock := &services.ExerciseTrailServiceMock{
		GetAllFunc: func() []domain.ExerciseTrail {
			return []domain.ExerciseTrail{
				{
					ID:          "trail",
					Name:        "test0",
					Description: "this is a description",
					Categories:  []string{"bike-track"},
					Length:      7,
					AreaServed:  "southern part",
				},
			}
		},
	}
	return mock
}
