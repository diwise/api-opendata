package datasets

import (
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
)

func NewRetrieveTemperaturesHandler(log logging.Logger, contextBrokerURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte{})

	})
}
