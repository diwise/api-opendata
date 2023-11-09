package stratsys

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestThatWeGetATokenFromStratsysHandler(t *testing.T) {
	is := is.New(t)
	server := setupTokenMockService(http.StatusOK, accessTokenResp)

	loginUrl := server.URL + "/token"
	defaultUrl := server.URL

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, loginUrl, nil)
	is.NoErr(err)

	NewRetrieveStratsysReportsHandler(context.Background(), "companyCode", "clientId", "scope", loginUrl, defaultUrl).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
}

func TestThatWeCanRetrieveASingleReportFromStratsysHandler(t *testing.T) {
	is := is.New(t)
	server := setupTokenMockService(http.StatusOK, accessTokenResp)

	loginUrl := server.URL + "/token"
	defaultUrl := server.URL

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/stratsys/1337", nil)
	is.NoErr(err)

	NewRetrieveStratsysReportsHandler(context.Background(), "companyCode", "clientId", "scope", loginUrl, defaultUrl).ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	is.Equal(w.Header().Get("Content-Type"), "application/json")
}

func setupTokenMockService(responseCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if strings.Contains(r.URL.Path, "token") {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(responseCode)
			w.Write([]byte(responseBody))
		} else {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(responseCode)
			w.Write([]byte(nil))
		}
	}))
}

const accessTokenResp string = `{"access_token":"ncjklhclabclksabclac",
"scope":"am_application_scope default",
"token_type":"Bearer",
"expires_in":3600}`
