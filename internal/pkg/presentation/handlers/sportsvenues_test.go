package handlers

import (
	"io"
	"net/http"
	"testing"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/sportsvenues"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/rs/zerolog"
)

func TestInvokeSportsVenuesHandler(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultSportsVenuesMock()
	req, err := http.NewRequest("GET", "?fields=seealso", nil)
	is.NoErr(err)

	NewRetrieveSportsVenuesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)    // response status should be 200 OK
	is.Equal(len(svc.GetAllCalls()), 1) // GetAll should have been called once

	body, err := io.ReadAll(rw.Body)
	is.NoErr(err)
	is.Equal(expectedVenuesOutput, string(body))
}

func TestInvokeSportsVenuesByIDHandler(t *testing.T) {
	is, r, ts := setupTest(t)
	svc := defaultSportsVenuesMock()
	oldfunc := svc.GetByIDFunc
	svc.GetByIDFunc = func(id string) (*domain.SportsVenue, error) {
		is.Equal(id, "test0")
		return oldfunc(id)
	}

	r.Get("/{id}", NewRetrieveSportsVenueByIDHandler(zerolog.Logger{}, svc))
	response, _ := newGetRequest(is, ts, "application/ld+json", "/test0", nil)

	is.Equal(response.StatusCode, http.StatusOK)
	is.Equal(len(svc.GetByIDCalls()), 1)
}

func TestGetSportsVenuesAsGeoJSON(t *testing.T) {
	is, r, ts := setupTest(t)

	svc := defaultSportsVenuesMock()

	r.Get("/sportsvenues", NewRetrieveSportsVenuesHandler(zerolog.Logger{}, svc))
	response, responseBody := newGetRequest(is, ts, "application/geo+json", "/sportsvenues?fields=description", nil)

	is.Equal(response.StatusCode, http.StatusOK) // response status should be 200 OK
	is.Equal(responseBody, sportsvenueGeoJSON)
}

func defaultSportsVenuesMock() *services.SportsVenueServiceMock {

	sf0 := domain.SportsVenue{
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
		Description: "cool description",
		SeeAlso:     []string{"https://a.com"},
	}
	sf1 := domain.SportsVenue{
		ID:         "id1",
		Name:       "test1",
		Categories: []string{"sports-hall"},
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
		Description: "even cooler description",
		SeeAlso:     []string{"https://b.com"},
	}

	list := []domain.SportsVenue{}

	list = append(list, sf0, sf1)

	mock := &services.SportsVenueServiceMock{
		GetAllFunc: func(c []string) []domain.SportsVenue {
			return list
		},
		GetByIDFunc: func(id string) (*domain.SportsVenue, error) {
			return &sf0, nil
		},
	}
	return mock
}

const expectedVenuesOutput string = `{"data":[{"categories":["ice-rink"],"id":"id0","location":{"type":"Point","coordinates":[17.428771593881844,62.42103804538807]},"name":"test0","seeAlso":["https://a.com"]},{"categories":["sports-hall"],"id":"id1","location":{"type":"Point","coordinates":[17.428771593881844,62.42103804538807]},"name":"test1","seeAlso":["https://b.com"]}]}`

const sportsvenueGeoJSON string = `{"type":"FeatureCollection", "features": [{"type":"Feature","id":"id0","geometry":{"type":"MultiPolygon","coordinates":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]},"properties":{"categories":["ice-rink"],"description":"cool description","name":"test0","type":"SportsVenue"}},{"type":"Feature","id":"id1","geometry":{"type":"MultiPolygon","coordinates":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]},"properties":{"categories":["sports-hall"],"description":"even cooler description","name":"test1","type":"SportsVenue"}}]}`
