package datasets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
)

func NewRetrieveStratsysReportsHandler(log logging.Logger, companyCode, clientID, scope, loginUrl, defaultUrl string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if companyCode == "" || clientID == "" || scope == "" || loginUrl == "" || defaultUrl == "" {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error("all environment variables need to be set")
			return
		}

		token, err := getTokenBearer(clientID, scope, loginUrl)
		if err != nil {
			log.Errorf("failed to retrieve token: %s", err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		reports, err := getReports(defaultUrl, companyCode, token)
		if err != nil {
			log.Errorf("failed to get reports: %s", err.Error())
			return
		}

		w.Write([]byte(reports))

	})
}

func getReports(url, companyCode, token string) ([]byte, error) {
	client := http.Client{}

	req, _ := http.NewRequest(http.MethodGet, url+"/api/publishedreports/v2", nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Stratsys-CompanyCode", companyCode)

	resp, err := client.Do(req)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed, status code not ok: %d", resp.StatusCode)
	}
	if err != nil {
		return nil, fmt.Errorf("request failed: %s", err.Error())
	}

	reports, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	return reports, nil
}

func getTokenBearer(clientID, scope, authUrl string) (string, error) {

	params := url.Values{}
	params.Add("grant_type", `client_credentials`)
	params.Add("scope", scope)
	params.Add("client_id", clientID)

	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest(http.MethodPost, authUrl, body)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %s", err.Error())
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: %s", err.Error())
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %s", err.Error())
	}

	defer resp.Body.Close()

	token := tokenResponse{}

	err = json.Unmarshal(bodyBytes, &token)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal access token json: %s", err.Error())
	}

	return token.AccessToken, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
