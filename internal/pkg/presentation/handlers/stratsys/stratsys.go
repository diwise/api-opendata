package stratsys

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
)

func NewRetrieveStratsysReportsHandler(log zerolog.Logger, companyCode, clientID, scope, loginUrl, defaultUrl string) http.HandlerFunc {
	if companyCode == "" || clientID == "" || scope == "" || loginUrl == "" || defaultUrl == "" {
		log.Fatal().Msg("all environment variables need to be set")
	}

	loginUrl = fmt.Sprintf("%s/%s/connect/token", loginUrl, companyCode)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := getTokenBearer(clientID, scope, loginUrl)
		if err != nil {
			log.Error().Err(err).Msg("failed to retrieve token")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		reportId := chi.URLParam(r, "id")

		if reportId != "" {
			response, err := getReportById(reportId, defaultUrl, companyCode, token)
			if err != nil {
				log.Error().Err(err).Msg("failed to get reports")
				w.WriteHeader(response.code)
				return
			}
			if response.contentType != "" {
				w.Header().Add("Content-Type", response.contentType)
			}
			w.Write(response.body)
		} else {
			response, err := getReports(defaultUrl, companyCode, token)
			if err != nil {
				log.Error().Err(err).Msg("failed to get reports")
				w.WriteHeader(response.code)
				return
			}
			if response.contentType != "" {
				w.Header().Add("Content-Type", response.contentType)
			}
			w.Write(response.body)
		}

	})
}

func getReportById(id, url, companyCode, token string) (stratsysResponse, error) {
	return getReportOrReports(url+"/api/publishedreports/v2/"+id, companyCode, token)
}

func getReports(url, companyCode, token string) (stratsysResponse, error) {
	return getReportOrReports(url+"/api/publishedreports/v2", companyCode, token)
}

func getReportOrReports(url, companyCode, token string) (stratsysResponse, error) {
	client := http.Client{}

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Stratsys-CompanyCode", companyCode)

	resp, err := client.Do(req)
	if err != nil {
		return stratsysResponse{code: http.StatusInternalServerError},
			fmt.Errorf("error when requesting report: %s", err.Error())
	}
	defer resp.Body.Close()

	ssresp := stratsysResponse{code: resp.StatusCode, contentType: resp.Header.Get("Content-Type")}

	if resp.StatusCode != http.StatusOK {
		return ssresp, fmt.Errorf("request failed, status code not ok: %d", resp.StatusCode)
	}

	ssresp.body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		ssresp.code = http.StatusInternalServerError
		return ssresp, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	return ssresp, nil
}

func getTokenBearer(clientID, scope, authUrl string) (string, error) {

	params := url.Values{}
	params.Add("grant_type", `client_credentials`)
	params.Add("scope", scope)
	params.Add("client_id", clientID)

	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest(http.MethodPost, authUrl, body)
	if err != nil {
		return "", fmt.Errorf("failed to create new token request: %s", err.Error())
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", fmt.Errorf("failed to get token: %s", err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response from token request: %d != %d", resp.StatusCode, http.StatusOK)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %s", err.Error())
	}

	token := tokenResponse{}

	err = json.Unmarshal(bodyBytes, &token)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal access token json: %s", err.Error())
	}

	return token.AccessToken, nil
}

type stratsysResponse struct {
	code        int
	contentType string
	body        []byte
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
