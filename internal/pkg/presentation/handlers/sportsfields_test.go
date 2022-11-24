package handlers

import (
	"io"
	"net/http"
	"testing"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/sportsfields"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/rs/zerolog"
)

func TestInvokeSportsFieldsHandler(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultSportsFieldsMock()
	req, err := http.NewRequest("GET", "", nil)
	is.NoErr(err)

	NewRetrieveSportsFieldsHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)    // response status should be 200 OK
	is.Equal(len(svc.GetAllCalls()), 1) // GetAll should have been called once

	body, err := io.ReadAll(rw.Body)
	is.NoErr(err)
	is.Equal(expectedOutput, string(body))
}

func TestInvokeSportsFieldsByIDHandler(t *testing.T) {
	is, r, ts := setupTest(t)
	svc := defaultSportsFieldsMock()
	oldfunc := svc.GetByIDFunc
	svc.GetByIDFunc = func(id string) (*domain.SportsField, error) {
		is.Equal(id, "test0")
		return oldfunc(id)
	}

	r.Get("/{id}", NewRetrieveSportsFieldByIDHandler(zerolog.Logger{}, svc))
	response, _ := newGetRequest(is, ts, "application/ld+json", "/test0", nil)

	is.Equal(response.StatusCode, http.StatusOK)
	is.Equal(len(svc.GetByIDCalls()), 1)
}

func TestGetSportsFieldsAsGeoJSON(t *testing.T) {
	is, r, ts := setupTest(t)

	svc := defaultSportsFieldsMock()

	r.Get("/sportsfields", NewRetrieveSportsFieldsHandler(zerolog.Logger{}, svc))
	response, responseBody := newGetRequest(is, ts, "application/geo+json", "/sportsfields?fields=description", nil)

	is.Equal(response.StatusCode, http.StatusOK) // response status should be 200 OK
	is.Equal(responseBody, sportsfieldGeoJSON)
}

func defaultSportsFieldsMock() *services.SportsFieldServiceMock {
	dlp := "2019-10-15T16:15:32Z"
	sf0 := domain.SportsField{
		ID:         "id0",
		Name:       "test0",
		Categories: []string{"ice-rink"},
		Location: domain.MultiPolygon{
			Type: "MultiPolygon",
			Coordinates: [][][][]float64{
				{
					{
						{17.428771593881844, 62.42103804538807}, {17.428785133659883, 62.421037809376244}, {17.428821575900738, 62.42048396661722}, {17.428101436027845, 62.42046508568337}, {17.428025378913084, 62.42103219129709}, {17.428365400350206, 62.421045125144}, {17.428690864217362, 62.421045739009976}, {17.428771593881844, 62.42103804538807},
					},
				},
			},
		},
		DateLastPreparation: &dlp,
		Description:         "cool description",
	}
	sf1 := domain.SportsField{
		ID:         "id1",
		Name:       "test1",
		Categories: []string{"ice-rink", "flood-lit"},
		Location: domain.MultiPolygon{
			Type: "MultiPolygon",
			Coordinates: [][][][]float64{
				{
					{
						{17.428771593881844, 62.42103804538807}, {17.428785133659883, 62.421037809376244}, {17.428821575900738, 62.42048396661722}, {17.428101436027845, 62.42046508568337}, {17.428025378913084, 62.42103219129709}, {17.428365400350206, 62.421045125144}, {17.428690864217362, 62.421045739009976}, {17.428771593881844, 62.42103804538807},
					},
				},
			},
		},
		DateLastPreparation: &dlp,
		Description:         "even cooler description",
	}

	list := []domain.SportsField{}

	list = append(list, sf0, sf1)

	mock := &services.SportsFieldServiceMock{
		GetAllFunc: func(c []string) []domain.SportsField {
			return list
		},
		GetByIDFunc: func(id string) (*domain.SportsField, error) {
			return &sf0, nil
		},
	}
	return mock
}

const expectedOutput string = `{"data":[{"categories":["ice-rink"],"id":"id0","location":{"type":"Point","coordinates":[17.428771593881844,62.42103804538807]},"name":"test0"},{"categories":["ice-rink","flood-lit"],"id":"id1","location":{"type":"Point","coordinates":[17.428771593881844,62.42103804538807]},"name":"test1"}]}`

const sportsfieldGeoJSON string = `{"type":"FeatureCollection", "features": [{"type":"Feature","id":"id0","geometry":{"type":"MultiPolygon","coordinates":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]},"properties":{"categories":["ice-rink"],"description":"cool description","name":"test0","type":"SportsField"}},{"type":"Feature","id":"id1","geometry":{"type":"MultiPolygon","coordinates":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]},"properties":{"categories":["ice-rink","flood-lit"],"description":"even cooler description","name":"test1","type":"SportsField"}}]}`
