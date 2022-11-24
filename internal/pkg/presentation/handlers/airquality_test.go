package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/airquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/rs/zerolog"
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
	oldfunc := svc.GetByIDFunc
	svc.GetByIDFunc = func(id string) ([]byte, error) {
		is.Equal(id, "aq0")
		return oldfunc(id)
	}

	r.Get("/{id}", NewRetrieveAirQualityByIDHandler(zerolog.Logger{}, svc))
	response, responseBody := newGetRequest(is, ts, "application/ld+json", "/aq0", nil)

	is.Equal(response.StatusCode, http.StatusOK)
	is.Equal(len(svc.GetByIDCalls()), 1)

	is.Equal(responseBody, expectedAirQualityOutput)
}

const expectedAirQualityOutput string = "{\n  \"data\": {\"id\":\"aq1\",\"location\":{\"type\":\"GeoProperty\",\"value\":{\"type\":\"Point\",\"coordinates\":[17.1,62.1]}}}\n}"

func defaultAirQualityMock() *services.AirQualityServiceMock {
	aqList := []domain.AirQuality{
		{
			ID: "aq1",
			Location: domain.LocationPoint{
				Type:  "GeoProperty",
				Value: *domain.NewPoint(62.1, 17.1),
			},
		},
		{
			ID: "aq2",
			Location: domain.LocationPoint{
				Type:  "GeoProperty",
				Value: *domain.NewPoint(62.2, 17.2),
			},
		},
		{
			ID: "aq3",
			Location: domain.LocationPoint{
				Type:  "GeoProperty",
				Value: *domain.NewPoint(62.3, 17.3),
			},
		},
	}

	aqListBytes, _ := json.Marshal(aqList)
	aq0Bytes, _ := json.Marshal(aqList[0])

	mock := &services.AirQualityServiceMock{
		GetAllFunc: func() []byte {
			return aqListBytes
		},
		GetByIDFunc: func(id string) ([]byte, error) {
			return aq0Bytes, nil
		},
	}

	return mock
}
