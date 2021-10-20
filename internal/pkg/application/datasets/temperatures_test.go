package datasets

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/matryer/is"
)

func TestInvokeTempHandler(t *testing.T) {
	is := is.New(t)
	l := logging.NewLogger()
	server := setupMockService(http.StatusOK, "")

	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", server.URL+"/api/temperatures", nil)

	NewRetrieveTemperaturesHandler(l, NewTempService(server.URL)).ServeHTTP(rw, req)

	is.Equal(rw.Code, http.StatusOK) // response status should be 200 OK
}

func setupMockService(responseCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/ld+json")
		w.WriteHeader(responseCode)
		w.Write([]byte(responseBody))
	}))
}
