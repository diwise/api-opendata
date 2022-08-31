package handlers

import (
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/application/services/citywork"
	"github.com/rs/zerolog"
)

func NewRetrieveCityworkHandler(logger zerolog.Logger, cityworkSvc citywork.CityworkService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := cityworkSvc.GetAll()

		roadworksJSON := "{\n  \"data\": " + string(body) + "\n}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(roadworksJSON))
	})
}

func NewRetrieveCityworkByIDHandler(logger zerolog.Logger, cityworkSvc citywork.CityworkService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}
