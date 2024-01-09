package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/airquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
)

func TestRetrieveAirQuality(t *testing.T) {
	is, log, rw := setup(t)
	svc := defaultAirQualityMock()
	req, err := http.NewRequest("GET", "", nil)
	is.NoErr(err)

	NewRetrieveAirQualitiesHandler(log, svc).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK)
	is.Equal(len(svc.GetAllCalls()), 1)
}

func TestRetrieveAirQualityByID(t *testing.T) {
	is, r, ts := setupTest(t)
	svc := defaultAirQualityMock()

	r.Get("/{id}", NewRetrieveAirQualityByIDHandler(context.Background(), svc))
	response, responseBody := newGetRequest(is, ts, "application/ld+json", "/aq1", nil)

	is.Equal(response.StatusCode, http.StatusOK)
	is.Equal(len(svc.GetByIDCalls()), 1)

	is.Equal(responseBody, expectedAirQualityOutput)
}

const expectedAirQualityOutput string = `{"data": {"id":"aq1","location":{"type":"Point","coordinates":[17.1,62.1]},"dateObserved":{"@type":"Property","@value":"2022-10-20T13:10:00Z"},"PM1":0.6}}`

func defaultAirQualityMock() *services.AirQualityServiceMock {
	mock := &services.AirQualityServiceMock{
		GetAllFunc: func(ctx context.Context) []domain.AirQuality {
			return aqList
		},
		GetByIDFunc: func(ctx context.Context, id string) (*domain.AirQualityDetails, error) {
			aq, ok := aqDetails[id]
			if ok {
				return &aq, nil
			} else {
				return nil, fmt.Errorf("no such air quality")
			}
		},
	}

	return mock
}

var aqList = []domain.AirQuality{
	{
		ID:       "aq1",
		Location: *domain.NewPoint(62.1, 17.1),
		DateObserved: domain.DateTime{
			Value: "2022-10-20T13:10:00Z",
		},
	},
	{
		ID:       "aq2",
		Location: *domain.NewPoint(62.2, 17.2),
		DateObserved: domain.DateTime{
			Value: "2022-10-21T13:10:00Z",
		},
	},
	{
		ID:       "aq3",
		Location: *domain.NewPoint(62.3, 17.3),
		DateObserved: domain.DateTime{
			Value: "2022-10-22T13:10:00Z",
		},
	},
}

var aqDetails = map[string]domain.AirQualityDetails{
	"aq1": {
		ID:       "aq1",
		Location: *domain.NewPoint(62.1, 17.1),
		DateObserved: domain.DateTime{
			Type:  "Property",
			Value: "2022-10-20T13:10:00Z",
		},
	},
}
