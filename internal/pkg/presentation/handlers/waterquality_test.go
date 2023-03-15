package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestGetWaterQuality(t *testing.T) {
	is := is.New(t)

	wqSvc := mockWaterQualitySvc(is)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/waterquality", nil)

	NewRetrieveWaterQualityHandler(zerolog.Logger{}, wqSvc).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK) // Request failed, status code not OK
}

func TestGetWaterQualityByID(t *testing.T) {
	is, router, testServer := testSetup(t)

	wqSvc := mockWaterQualitySvc(is)

	router.Get("/{id}", NewRetrieveWaterQualityByIDHandler(zerolog.Logger{}, wqSvc))
	resp, _ := newGetRequest(is, testServer, "application/ld+json", "/urn:ngsi-ld:WaterQualityObserved:testID", nil)

	is.Equal(resp.StatusCode, http.StatusOK) // Request failed, status code not OK
	is.Equal(len(wqSvc.GetByIDCalls()), 1)   // GetByID should have been called exactly once
}

func testSetup(t *testing.T) (*is.I, *chi.Mux, *httptest.Server) {
	is := is.New(t)
	r := chi.NewRouter()
	ts := httptest.NewServer(r)

	return is, r, ts
}

func mockWaterQualitySvc(is *is.I) *waterquality.WaterQualityServiceMock {
	return &waterquality.WaterQualityServiceMock{
		GetAllFunc: func() []domain.WaterQuality {
			dto := waterquality.WaterQualityDTO{}
			err := json.Unmarshal([]byte(waterqualityJson), &dto)
			is.NoErr(err)

			wq := domain.WaterQuality{
				ID:           dto.ID,
				Temperature:  dto.Temperature,
				DateObserved: dto.DateObserved.Value,
				Location:     *domain.NewPoint(dto.Location.Value.Coordinates[0], dto.Location.Value.Coordinates[1]),
			}

			return []domain.WaterQuality{wq}
		},
		GetByIDFunc: func(id string) (*domain.WaterQualityTemporal, error) {
			wqt := &domain.WaterQualityTemporal{}
			err := json.Unmarshal([]byte(waterqualityTemporalJson), wqt)
			is.NoErr(err)

			return wqt, nil
		},
	}
}

const waterqualityTemporalJson string = `{
	"@context": [
	  "https://schema.lab.fiware.org/ld/context",
	  "https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
	],
	"dateObserved": {
	  "type": "Property",
	  "value": {
		"@type": "DateTime",
		"@value": "2021-05-18T19:23:09Z"
	  }
	},
	"id": "urn:ngsi-ld:WaterQualityObserved:testID",
	"location": {
	  "type": "GeoProperty",
	  "value": {
		"coordinates": [
		  17.39364,
		  62.297684
		],
		"type": "Point"
	  }
	},
	"temperature": [{
		"type": "Property",
		"value": 10.8,
		"observedAt": "2021-05-18T19:23:09Z"
	}],
	"type": "WaterQualityObserved"
  }`

const waterqualityJson string = `{
	"@context": [
	  "https://schema.lab.fiware.org/ld/context",
	  "https://uri.etsi.org/ngsi-ld/v1/ngsi-ld-core-context.jsonld"
	],
	"dateObserved": {
	  "type": "Property",
	  "value": {
		"@type": "DateTime",
		"@value": "2021-05-18T19:23:09Z"
	  }
	},
	"id": "urn:ngsi-ld:WaterQualityObserved:testID",
	"location": {
	  "type": "GeoProperty",
	  "value": {
		"coordinates": [
		  17.39364,
		  62.297684
		],
		"type": "Point"
	  }
	},
	"temperature": 10.8,
	"type": "WaterQualityObserved"
  }`
