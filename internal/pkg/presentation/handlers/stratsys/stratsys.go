package stratsys

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/api/stratsys")

func NewRetrieveStratsysReportsHandler(logger zerolog.Logger, companyCode, clientID, scope, loginUrl, defaultUrl string) http.HandlerFunc {
	if companyCode == "" || clientID == "" || scope == "" || loginUrl == "" || defaultUrl == "" {
		logger.Fatal().Msg("all environment variables need to be set")
	}

	loginUrl = fmt.Sprintf("%s/%s/connect/token", loginUrl, companyCode)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-stratsys-reports")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		token, err := getTokenBearer(ctx, log, clientID, scope, loginUrl)
		if err != nil {
			log.Error().Err(err).Msg("failed to retrieve token")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		reportId := chi.URLParam(r, "id")

		if reportId != "" {
			response, err := getReportById(ctx, log, reportId, defaultUrl, companyCode, token)
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
			response, err := getReports(ctx, log, defaultUrl, companyCode, token)
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

func getReportById(ctx context.Context, log zerolog.Logger, id, url, companyCode, token string) (stratsysResponse, error) {
	return getReportOrReports(ctx, log, url+"/api/publishedreports/v2/"+id, companyCode, token)
}

func getReports(ctx context.Context, log zerolog.Logger, url, companyCode, token string) (stratsysResponse, error) {
	return getReportOrReports(ctx, log, url+"/api/publishedreports/v2", companyCode, token)
}

func getReportOrReports(ctx context.Context, log zerolog.Logger, url, companyCode, token string) (stratsysResponse, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Stratsys-CompanyCode", companyCode)

	resp, err := httpClient.Do(req)
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

func getTokenBearer(ctx context.Context, log zerolog.Logger, clientID, scope, authUrl string) (string, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	ctx, span := tracer.Start(ctx, "retrieve-token")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	params := url.Values{}
	params.Add("grant_type", `client_credentials`)
	params.Add("scope", scope)
	params.Add("client_id", clientID)

	body := strings.NewReader(params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authUrl, body)
	if err != nil {
		err = fmt.Errorf("failed to create new token request: %w", err)
		return "", err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to get token: %w", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected response from token request: %d != %d", resp.StatusCode, http.StatusOK)
		return "", err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response body: %w", err)
		return "", err
	}

	token := tokenResponse{}

	err = json.Unmarshal(bodyBytes, &token)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal access token json: %w", err)
		return "", err
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
