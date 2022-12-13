package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/exercisetrails"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
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

	const expectedResponse string = `{"data":[{"categories":["bike-track"],"description":"this is a description","id":"trail0","length":7,"name":"test0"}]}`
	is.Equal(string(response), expectedResponse)
}

func TestGetExerciseTrailsWithPublicAccess(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultTrailsMock()
	req, _ := http.NewRequest("GET", "?fields=publicaccess", nil)

	NewRetrieveExerciseTrailsHandler(log, svc).ServeHTTP(rw, req)
	response, _ := io.ReadAll(rw.Body)

	const expectedResponse string = `{"data":[{"categories":["bike-track"],"id":"trail0","length":7,"name":"test0","publicAccess":"no"}]}`
	is.Equal(string(response), expectedResponse)
}

func TestGetExerciseTrailsWithNoSpecificCategories(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultTrailsMock()
	req, err := http.NewRequest("GET", "", nil)
	is.NoErr(err)

	defaultGetAll := svc.GetAllFunc
	svc.GetAllFunc = func(c []string) []domain.ExerciseTrail {
		is.Equal(c, []string{})
		return defaultGetAll(c)
	}

	NewRetrieveExerciseTrailsHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)    // response status should be 200 OK
	is.Equal(len(svc.GetAllCalls()), 1) // Get should have been called once
}

func TestGetExerciseTrailsWithCertainCategories(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultTrailsMock()
	req, err := http.NewRequest("GET", "?categories=bike-track,floodlit", nil)
	is.NoErr(err)

	defaultGetAll := svc.GetAllFunc
	svc.GetAllFunc = func(c []string) []domain.ExerciseTrail {
		is.Equal(c, []string{"bike-track", "floodlit"})
		return defaultGetAll(c)
	}

	NewRetrieveExerciseTrailsHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)    // response status should be 200 OK
	is.Equal(len(svc.GetAllCalls()), 1) // Get should have been called once
}

const expectedGPXOutput string = `<?xml version="1.0" encoding="UTF-8"?>
<gpx creator="diwise cip" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.topografix.com/GPX/1/1 http://www.topografix.com/GPX/1/1/gpx.xsd" version="1.1" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>test0</name>
    <trkseg>
      <trkpt lat="62.368439" lon="17.313069">
        <ele>32.1</ele>
      </trkpt>
      <trkpt lat="62.368418" lon="17.313284">
        <ele>42.5</ele>
      </trkpt>
      <trkpt lat="62.368416" lon="17.313413">
        <ele>38.7</ele>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`

func TestGetExerciseTrailAsGPX(t *testing.T) {
	is, r, ts := setupTest(t)

	svc := defaultTrailsMock()
	oldfunc := svc.GetByIDFunc
	svc.GetByIDFunc = func(id string) (*domain.ExerciseTrail, error) {
		is.Equal(id, "expected-id")
		return oldfunc(id)
	}

	r.Get("/{id}", NewRetrieveExerciseTrailByIDHandler(zerolog.Logger{}, svc))
	response, responseBody := newGetRequest(is, ts, "application/gpx+xml", "/expected-id", nil)

	is.Equal(response.StatusCode, http.StatusOK) // response status should be 200 OK
	is.Equal(len(svc.GetByIDCalls()), 1)         // Get should have been called once

	is.Equal(responseBody, expectedGPXOutput)
}

const expectedGeoJSONOutput string = `{"type":"FeatureCollection", "features": [{"type":"Feature","id":"trail0","geometry":{"type":"LineString","coordinates":[[17.313069,62.368439,32.1],[17.313284,62.368418,42.5],[17.313413,62.368416,38.7]]},"properties":{"categories":["bike-track"],"length":7,"name":"test0","type":"ExerciseTrail"}}]}`

func TestGetExerciseTrailAsGeoJSON(t *testing.T) {
	is, r, ts := setupTest(t)

	svc := defaultTrailsMock()

	r.Get("/exercisetrails", NewRetrieveExerciseTrailsHandler(zerolog.Logger{}, svc))
	response, responseBody := newGetRequest(is, ts, "application/geo+json", "/exercisetrails", nil)

	is.Equal(response.StatusCode, http.StatusOK) // response status should be 200 OK
	is.Equal(responseBody, expectedGeoJSONOutput)
}

func defaultTrailsMock() *services.ExerciseTrailServiceMock {
	trail0 := domain.ExerciseTrail{
		ID:           "trail0",
		Name:         "test0",
		Description:  "this is a description",
		Categories:   []string{"bike-track"},
		PublicAccess: "no",
		Length:       7,
		AreaServed:   "southern part",
		Location:     *domain.NewLineString([][]float64{{17.313069, 62.368439, 32.1}, {17.313284, 62.368418, 42.5}, {17.313413, 62.368416, 38.7}}),
	}

	mock := &services.ExerciseTrailServiceMock{
		GetAllFunc: func(c []string) []domain.ExerciseTrail {
			return []domain.ExerciseTrail{trail0}
		},
		GetByIDFunc: func(id string) (*domain.ExerciseTrail, error) {
			return &trail0, nil
		},
	}
	return mock
}

func newGetRequest(is *is.I, ts *httptest.Server, accept, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, body)
	is.NoErr(err)

	req.Header.Add("Accept", accept)

	resp, err := http.DefaultClient.Do(req)
	is.NoErr(err) // http request failed
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	is.NoErr(err) // failed to read response body

	return resp, string(respBody)
}

func setupTest(t *testing.T) (*is.I, *chi.Mux, *httptest.Server) {
	is := is.New(t)
	r := chi.NewRouter()
	ts := httptest.NewServer(r)

	return is, r, ts
}
