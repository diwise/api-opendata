package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	NUTSCodePrefix      string = "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/"
	WikidataPrefix      string = "https://www.wikidata.org/wiki/"
	YearMonthDayISO8601 string = "2006-01-02"
)

func NewRetrieveBeachByIDHandler(logger zerolog.Logger, contextBroker string) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-beach-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		beachID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if beachID == "" {
			err = fmt.Errorf("no beach id supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		inBeach, err := getBeachByIDFromContextBroker(ctx, log, contextBroker, "default", beachID)
		if err != nil {
			if err == ErrNoSuchBeach {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			err = fmt.Errorf("failed to request beach by ID: (%w)", err)
			log.Error().Err(err).Msg("internal error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		latitude, longitude := inBeach.LatLon()
		wq, err := getWaterQualitiesNearBeach(ctx, contextBroker, latitude, longitude)

		outBeach := &BeachOut{
			ID:           inBeach.ID,
			Name:         inBeach.Name,
			Description:  inBeach.Description,
			Location:     *NewPoint(latitude, longitude),
			WaterQuality: wq,
			RefSeeAlso:   inBeach.RefSeeAlso,
		}

		json, err := json.MarshalIndent(outBeach, "", "  ")
		w.Header().Add("Content-Type", "application/json")
		w.Write(json)
	})
}

func getWaterQualitiesNearBeach(ctx context.Context, contextBroker string, latitude, longitude float64) ([]WaterQuality, error) {
	return []WaterQuality{{Temperature: 15.2, DateObserved: time.Now().UTC().Format(time.RFC3339)}}, nil
}

func NewRetrieveBeachesHandler(logger zerolog.Logger, contextBroker string) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		acceptedContentType := r.Header.Get("Accept")
		if strings.HasPrefix(acceptedContentType, "application/json") {
			serveBeachesAsJSON(logger, contextBroker, w, r)
		} else {
			serveBeachesAsTextCSV(logger, contextBroker, w, r)
		}
	})
}

const beachJSONFormat string = `{"id": "%s", "name": "%s", "location": {"type": "Point", "coordinates": [%f, %f]}}"`

func serveBeachesAsJSON(logger zerolog.Logger, contextBroker string, w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, span := tracer.Start(r.Context(), "retrieve-beaches")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	beaches := []string{}

	err = getBeachesFromContextBroker(ctx, log, contextBroker, "default", func(b beach) {
		latitude, longitude := b.LatLon()
		beaches = append(beaches, fmt.Sprintf(beachJSONFormat, b.ID, b.Name, longitude, latitude))
	})

	if err != nil {
		log.Error().Err(err).Msgf("failed to get beaches from %s", contextBroker)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	beachJSON := "{\"data\": [" + strings.Join(beaches, ",") + "]}"

	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(beachJSON))
}

func serveBeachesAsTextCSV(logger zerolog.Logger, contextBroker string, w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, span := tracer.Start(r.Context(), "retrieve-beaches-csv")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	beachesCsv := bytes.NewBufferString("place_id;name;latitude;longitude;hov_ref;wikidata;updated;temp_url;description")

	err = getBeachesFromContextBroker(ctx, log, contextBroker, "default", func(b beach) {
		latitude, longitude := b.LatLon()

		time := getDateModifiedFromBeach(&b)
		nutsCode := getNutsCodeFromBeach(&b)
		wiki := getWikiRefFromBeach(&b)

		tempURL := fmt.Sprintf(
			"\"%s/ngsi-ld/v1/entities?type=WaterQualityObserved&georel=near%%3BmaxDistance==1000&geometry=Point&coordinates=[%f,%f]\"",
			contextBroker, longitude, latitude,
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
}

var ErrNoSuchBeach error = fmt.Errorf("beach not found")

func getBeachByIDFromContextBroker(ctx context.Context, logger zerolog.Logger, brokerURL, tenant, beachID string) (*beach, error) {
	var err error

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	beachID = url.QueryEscape(beachID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, brokerURL+"/ngsi-ld/v1/entities/"+beachID+"?options=keyValues", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Add("Accept", "application/ld+json")
	req.Header.Add("Link", "<https://schema.lab.fiware.org/ld/context>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"")
	req.Header.Add("NGSILD-Tenant", tenant)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNoSuchBeach
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	if resp.StatusCode >= http.StatusBadRequest {
		reqbytes, _ := httputil.DumpRequest(req, false)
		respbytes, _ := httputil.DumpResponse(resp, false)

		logger.Error().Str("request", string(reqbytes)).Str("response", string(respbytes)).Msg("request failed")
		return nil, fmt.Errorf("request failed")
	}

	if resp.StatusCode != http.StatusOK {
		contentType := resp.Header.Get("Content-Type")
		return nil, fmt.Errorf("context source returned status code %d (content-type: %s, body: %s)", resp.StatusCode, contentType, string(respBody))
	}

	theBeach := &beach{}
	err = json.Unmarshal(respBody, theBeach)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	return theBeach, nil
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

type WaterQuality struct {
	Temperature  float64 `json:"temperature"`
	DateObserved string  `json:"dateObserved"`
}

type BeachOut struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Location     Point          `json:"location"`
	WaterQuality []WaterQuality `json:"waterquality"`
	RefSeeAlso   []string       `json:"refSeeAlso"`
}

type Point struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

func NewPoint(latitude, longitude float64) *Point {
	return &Point{
		"Point",
		[]float64{longitude, latitude},
	}
}
