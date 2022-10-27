package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/sportsfields"
	"github.com/diwise/api-opendata/internal/pkg/domain"
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
	is, log, rw := setup(t)
	svc := defaultSportsFieldsMock()
	req, err := http.NewRequest("GET", "/test0", nil)
	is.NoErr(err)

	NewRetrieveSportsFieldByIDHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)     // response status should be 200 OK
	is.Equal(len(svc.GetByIDCalls()), 1) // GetByID should have been called once
}

func defaultSportsFieldsMock() *services.SportsFieldServiceMock {
	dlp := "2019-10-15T16:15:32Z"
	sf0 := domain.SportsField{
		Name:       "test0",
		Categories: []string{"ice-rink"},
		Geometry: domain.MultiPolygon{
			Type: "MultiPolygon",
			Lines: [][][][]float64{
				{
					{
						{17.428771593881844, 62.42103804538807}, {17.428785133659883, 62.421037809376244}, {17.428821575900738, 62.42048396661722}, {17.428101436027845, 62.42046508568337}, {17.428025378913084, 62.42103219129709}, {17.428365400350206, 62.421045125144}, {17.428690864217362, 62.421045739009976}, {17.428771593881844, 62.42103804538807},
					},
				},
			},
		},
		DateLastPrepared: &dlp,
	}
	sf1 := domain.SportsField{
		Name:       "test1",
		Categories: []string{"ice-rink", "flood-lit"},
		Geometry: domain.MultiPolygon{
			Type: "MultiPolygon",
			Lines: [][][][]float64{
				{
					{
						{17.428771593881844, 62.42103804538807}, {17.428785133659883, 62.421037809376244}, {17.428821575900738, 62.42048396661722}, {17.428101436027845, 62.42046508568337}, {17.428025378913084, 62.42103219129709}, {17.428365400350206, 62.421045125144}, {17.428690864217362, 62.421045739009976}, {17.428771593881844, 62.42103804538807},
					},
				},
			},
		},
		DateLastPrepared: &dlp,
	}

	body, _ := json.Marshal(sf0)

	list := []domain.SportsField{}

	list = append(list, sf0, sf1)

	listBody, _ := json.Marshal(list)

	mock := &services.SportsFieldServiceMock{
		GetAllFunc: func() []byte {
			return listBody
		},
		GetByIDFunc: func(id string) ([]byte, error) {
			return body, nil
		},
	}
	return mock
}

const expectedOutput string = "{\n  \"data\": [{\"name\":\"test0\",\"categories\":[\"ice-rink\"],\"geometry\":{\"type\":\"MultiPolygon\",\"coordinates\":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]},\"dateLastPrepared\":\"2019-10-15T16:15:32Z\"},{\"name\":\"test1\",\"categories\":[\"ice-rink\",\"flood-lit\"],\"geometry\":{\"type\":\"MultiPolygon\",\"coordinates\":[[[[17.428771593881844,62.42103804538807],[17.428785133659883,62.421037809376244],[17.428821575900738,62.42048396661722],[17.428101436027845,62.42046508568337],[17.428025378913084,62.42103219129709],[17.428365400350206,62.421045125144],[17.428690864217362,62.421045739009976],[17.428771593881844,62.42103804538807]]]]},\"dateLastPrepared\":\"2019-10-15T16:15:32Z\"}]\n}"
