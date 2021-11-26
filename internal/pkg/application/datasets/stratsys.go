package datasets

import (
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
)

func NewRetrieveStratsysReportsHandler(log logging.Logger, companyCode, clientID, scope string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if companyCode == "" || clientID == "" || scope == "" {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error("all environment variables need to be set")
			return
		}

		// http.Post with clientID and scope to get token

		// use token and company code to get reports

	})
}
