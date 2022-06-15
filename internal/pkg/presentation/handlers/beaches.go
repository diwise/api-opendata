package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	NUTSCodePrefix      string = "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/"
	WikidataPrefix      string = "https://www.wikidata.org/wiki/"
	YearMonthDayISO8601 string = "2006-01-02"
)

func NewRetrieveBeachesHandler(logger zerolog.Logger, contextBroker string) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-beaches")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		beachesCsv := bytes.NewBufferString("place_id;name;latitude;longitude;hov_ref;wikidata;updated;temp_url;description")

		err = getBeachesFromContextBroker(ctx, log, contextBroker, "default", func(b beach) {
			latitude, longitude := b.LatLon()

			time := getDateModifiedFromBeach(&b)
			nutsCode := getNutsCodeFromBeach(&b)
			wiki := getWikiRefFromBeach(&b)

			tempURL := fmt.Sprintf(
				"\"%s/ngsi-ld/v1/entities?type=WaterQualityObserved&georel=near%%3BmaxDistance==1000&geometry=Point&coordinates=[%f,%f]\"", contextBroker, longitude, latitude,
			)

			beachInfo := fmt.Sprintf("\r\n%s;%s;%f;%f;%s;%s;%s;%s;\"%s\"",
				b.ID, b.Name, latitude, longitude,
				nutsCode,
				wiki,
				time,
				tempURL,
				strings.ReplaceAll(b.Description, "\"", "\"\""),
			)

			beachesCsv.Write([]byte(beachInfo))
		})

		if err != nil {
			log.Error().Err(err).Msgf("failed to get beaches from %s", contextBroker)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "text/csv")
		w.Write(beachesCsv.Bytes())
	})
}

func getBeachesFromContextBroker(ctx context.Context, logger zerolog.Logger, brokerURL, tenant string, callback func(b beach)) error {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, brokerURL+"/ngsi-ld/v1/entities?type=Beach&options=keyValues", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Add("Accept", "application/ld+json")
	req.Header.Add("Link", "<https://schema.lab.fiware.org/ld/context>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"")
	req.Header.Add("NGSILD-Tenant", tenant)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %s", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %s", err.Error())
	}

	if resp.StatusCode >= http.StatusBadRequest {
		reqbytes, _ := httputil.DumpRequest(req, false)
		respbytes, _ := httputil.DumpResponse(resp, false)

		logger.Error().Str("request", string(reqbytes)).Str("response", string(respbytes)).Msg("request failed")
		return fmt.Errorf("request failed")
	}

	if resp.StatusCode != http.StatusOK {
		contentType := resp.Header.Get("Content-Type")
		return fmt.Errorf("context source returned status code %d (content-type: %s, body: %s)", resp.StatusCode, contentType, string(respBody))
	}

	logger.Info().Msgf("response: %s", respBody)

	var beaches []beach
	err = json.Unmarshal(respBody, &beaches)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	for _, b := range beaches {
		callback(b)
	}

	return nil
}

func getNutsCodeFromBeach(b *beach) string {
	for _, ref := range b.RefSeeAlso {
		if strings.HasPrefix(ref, NUTSCodePrefix) {
			return strings.TrimPrefix(ref, NUTSCodePrefix)
		}
	}

	return ""
}

func getDateModifiedFromBeach(b *beach) string {
	if b.DateModified == "" {
		return ""
	}

	timestamp, err := time.Parse(time.RFC3339, b.DateModified)
	if err != nil {
		return ""
	}

	return timestamp.Format(YearMonthDayISO8601)
}

func getWikiRefFromBeach(b *beach) string {
	for _, ref := range b.RefSeeAlso {
		if strings.HasPrefix(ref, WikidataPrefix) {
			return strings.TrimPrefix(ref, WikidataPrefix)
		}
	}

	return ""
}

type beach struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    struct {
		Type        string          `json:"type"`
		Coordinates [][][][]float64 `json:"coordinates"`
	} `json:"location"`
	RefSeeAlso   []string `json:"refSeeAlso"`
	DateModified string   `json:"dateModified"`
}

func (b *beach) LatLon() (float64, float64) {
	// TODO: A more fancy calculation of midpoint or something?
	return b.Location.Coordinates[0][0][0][1], b.Location.Coordinates[0][0][0][0]
}
