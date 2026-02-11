package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

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

	re := regexp.MustCompile(`https?://[^/]+`)
	cleanedBody := re.ReplaceAllString(responseBody, "")
	is.Equal(cleanedBody, expectedAirQualityByIDOutput)
}

func TestRetrieveAirQualityByIDParsesTimeParamsCorrectly(t *testing.T) {
	is, r, ts := setupTest(t)
	svc := defaultAirQualityMock()

	r.Get("/{id}", NewRetrieveAirQualityByIDHandler(context.Background(), svc))
	response, _ := newGetRequest(is, ts, "application/ld+json", "/aq1?from=2022-10-21T13:10:00Z&to=2022-10-21T13:10:00Z", nil)

	is.Equal(response.StatusCode, http.StatusOK)
	is.Equal(len(svc.GetByIDWithTimespanCalls()), 1)
}

func TestRetrieveAirQualityByIDCorrectLinks(t *testing.T) {
	is, r, ts := setupTest(t)
	svc := defaultAirQualityMock()

	r.Get("/{id}", NewRetrieveAirQualityByIDHandler(context.Background(), svc))
	response, responseBody := newGetRequest(is, ts, "application/ld+json", "/aq1?from=2022-10-19T13:10:00Z&to=2022-10-22T13:10:00Z", nil)

	is.Equal(response.StatusCode, http.StatusOK)
	is.Equal(len(svc.GetByIDWithTimespanCalls()), 1)

	re := regexp.MustCompile(`https?://[^/]+`)
	cleanedBody := re.ReplaceAllString(responseBody, "")
	is.Equal(cleanedBody, expectedAirQualityByIDLinksOutput)
}

const expectedAirQualityByIDOutput string = `{"data":{"id":"aq1","location":{"type":"Point","coordinates":[17.1,62.1]},"dateObserved":{"@type":"DateTime","@value":"2022-10-21T13:10:00Z"},"pollutants":[{"name":"Temperature","values":[{"value":12.6,"observedAt":"2022-10-20T13:10:00Z"}]}]},"links":{"self":"/aq1"}}`

const expectedAirQualityByIDLinksOutput string = `{"data":{"id":"aq1","location":{"type":"Point","coordinates":[17.1,62.1]},"dateObserved":{"@type":"DateTime","@value":"2022-10-21T13:10:00Z"},"pollutants":[{"name":"Temperature","values":[{"value":12.6,"observedAt":"2022-10-20T13:10:00Z"}]}]},"links":{"next":"/aq1?from=2022-10-20T13%3A10%3A00Z&to=2022-10-22T13%3A10%3A00Z","self":"/aq1?from=2022-10-19T13%3A10%3A00Z&to=2022-10-22T13%3A10%3A00Z"}}`

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
		GetByIDWithTimespanFunc: func(ctx context.Context, id string, from, to time.Time) (*domain.AirQualityDetails, *services.NextTimespan, error) {
			aq, ok := aqDetails[id]
			nextFrom := from.Add(time.Hour * 24)
			if ok {
				return &aq, &services.NextTimespan{From: &nextFrom, To: &to}, nil
			}

			return nil, nil, fmt.Errorf("no such air quality")
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
