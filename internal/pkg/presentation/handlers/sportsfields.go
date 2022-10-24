package handlers

import (
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsfields"
	"github.com/rs/zerolog"
)

func NewRetrieveSportsFieldByIDHandler(logger zerolog.Logger, sfsvc sportsfields.SportsFieldService) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write([]byte("not finished"))
	})
}

func NewRetrieveSportsFieldsHandler(logger zerolog.Logger, sfsvc sportsfields.SportsFieldService) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte("not finished"))

	})
}
