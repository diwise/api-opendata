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
	responseBody := rw.Body.Bytes()

	is.Equal(string(responseBody), `{"data":[{"id":"aq1","location":{"type":"Point","coordinates":[17.1,62.1]},"dateObserved":{"@type":"DateTime","@value":"2022-10-20T13:10:00Z"},"atmosphericPressure":12.6,"temperature":12.6,"relativeHumidity":12.6,"particleCount":12.6,"PM1":12.6,"PM4":12.6,"PM10":12.6,"PM25":12.6,"totalSuspendedParticulate":12.6,"CO2":12.6,"NO":12.6,"NO2":12.6,"NOx":12.6,"voltage":12.6,"windDirection":12.6,"windSpeed":12.6},{"id":"aq2","location":{"type":"Point","coordinates":[17.2,62.2]},"dateObserved":{"@type":"DateTime","@value":"2022-10-21T13:10:00Z"}},{"id":"aq3","location":{"type":"Point","coordinates":[17.3,62.3]},"dateObserved":{"@type":"DateTime","@value":"2022-10-22T13:10:00Z"}}]}`)
}

func TestRetrieveAirQualityByID(t *testing.T) {
	is, r, ts := setupTest(t)
	svc := defaultAirQualityMock()

	r.Get("/{id}", NewRetrieveAirQualityByIDHandler(context.Background(), svc))
	response, responseBody := newGetRequest(is, ts, "application/ld+json", "/aq1", nil)

	is.Equal(response.StatusCode, http.StatusOK)
	is.Equal(len(svc.GetByIDCalls()), 1)

	is.Equal(responseBody, expectedAirQualityByIDOutput)
}

const expectedAirQualityByIDOutput string = `{"data": {"id":"aq1","location":{"type":"Point","coordinates":[17.1,62.1]},"dateObserved":{"@type":"DateTime","@value":"2022-10-21T13:10:00Z"},"pollutants":[{"name":"Temperature","values":[{"value":12.6,"observedAt":"2022-10-20T13:10:00Z"}]}]}}`

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

var value float64 = 12.6

var aqList = []domain.AirQuality{
	{
		ID:                        "aq1",
		Location:                  *domain.NewPoint(62.1, 17.1),
		DateObserved:              *domain.NewDateTime("2022-10-20T13:10:00Z"),
		AtmosphericPressure:       &value,
		Temperature:               &value,
		RelativeHumidity:          &value,
		ParticleCount:             &value,
		PM1:                       &value,
		PM4:                       &value,
		PM10:                      &value,
		PM25:                      &value,
		TotalSuspendedParticulate: &value,
		CO2:                       &value,
		NO:                        &value,
		NO2:                       &value,
		NOx:                       &value,
		Voltage:                   &value,
		WindDirection:             &value,
		WindSpeed:                 &value,
	},
	{
		ID:           "aq2",
		Location:     *domain.NewPoint(62.2, 17.2),
		DateObserved: *domain.NewDateTime("2022-10-21T13:10:00Z"),
	},
	{
		ID:           "aq3",
		Location:     *domain.NewPoint(62.3, 17.3),
		DateObserved: *domain.NewDateTime("2022-10-22T13:10:00Z"),
	},
}

var aqDetails = map[string]domain.AirQualityDetails{
	"aq1": {
		ID:           "aq1",
		Location:     *domain.NewPoint(62.1, 17.1),
		DateObserved: *domain.NewDateTime("2022-10-21T13:10:00Z"),
		Pollutants: []domain.Pollutant{
			{
				Name: "Temperature",
				Values: []domain.Value{
					{
						Value:      12.6,
						ObservedAt: "2022-10-20T13:10:00Z",
					},
				},
			},
		},
	},
}
