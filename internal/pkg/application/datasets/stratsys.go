package datasets

import (
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
)

func NewRetrieveStratsysReportsHandler(log logging.Logger, companyCode, clientID, scope string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}
