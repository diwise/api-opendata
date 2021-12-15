package stratsys

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
)

func TestThatWeGetATokenFromStratsysHandler(t *testing.T) {

	log := logging.NewLogger()
	server := setupTokenMockService(http.StatusOK, accessTokenResp)

	companyCode := "companyCode"
	clientId := "clientId"
	scope := "scope"
	loginUrl := server.URL + "/token"
	defaultUrl := server.URL

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, loginUrl, nil)

	NewRetrieveStratsysReportsHandler(log, companyCode, clientId, scope, loginUrl, defaultUrl).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", w.Code)
	}
}

func TestThatWeCanRetrieveASingleReportFromStratsysHandler(t *testing.T) {

	log := logging.NewLogger()
	server := setupTokenMockService(http.StatusOK, accessTokenResp)

	companyCode := "companyCode"
	clientId := "clientId"
	scope := "scope"
	loginUrl := server.URL + "/token"
	defaultUrl := server.URL

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/stratsys/1337", nil)

	NewRetrieveStratsysReportsHandler(log, companyCode, clientId, scope, loginUrl, defaultUrl).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Request failed, status code not OK: %d", w.Code)
	}
}

func setupTokenMockService(responseCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if strings.Contains(r.URL.Path, "token") {
			w.Header().Add("Content-Type", "application/ld+json")
			w.WriteHeader(responseCode)
			w.Write([]byte(responseBody))
		} else {
			w.Header().Add("Content-Type", "application/ld+json")
			w.WriteHeader(responseCode)
			w.Write([]byte(nil))
		}
	}))
}

const accessTokenResp string = `{"access_token":"ncjklhclabclksabclac",
"scope":"am_application_scope default",
"token_type":"Bearer",
"expires_in":3600}`
