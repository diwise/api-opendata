package handlers

import (
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/application/services/airquality"
	"github.com/rs/zerolog"
)

func NewRetrieveAirQualitiesHandler(log zerolog.Logger, aqsvc airquality.AirQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Write([]byte("air quality"))
	})
}

func NewRetrieveAirQualityByIDHandler(log zerolog.Logger, aqsvc airquality.AirQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Write([]byte("air quality by id"))
	})
}
