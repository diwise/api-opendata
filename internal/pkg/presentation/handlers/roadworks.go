package handlers

import (
	"net/http"

	"github.com/rs/zerolog"
)

func NewRetrieveRoadWorksHandler(log zerolog.Logger, ctxBrokerURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}
