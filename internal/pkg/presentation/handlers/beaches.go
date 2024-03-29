package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/application/services/beaches"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/diwise/service-chassis/pkg/presentation/api/http/errors"
	"github.com/go-chi/chi/v5"
)

const (
	NUTSCodePrefix      string = "https://badplatsen.havochvatten.se/badplatsen/karta/#/bath/"
	WikidataPrefix      string = "https://www.wikidata.org/wiki/"
	YearMonthDayISO8601 string = "2006-01-02"
)

func NewRetrieveBeachByIDHandler(ctx context.Context, beachService beaches.BeachService) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-beach-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		traceID, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		beachID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if beachID == "" {
			err = fmt.Errorf("no beach id supplied in query")
			log.Error("bad request", slog.String("err", err.Error()))
			problem := errors.NewProblemReport(http.StatusBadRequest, "badrequest", errors.Detail(err.Error()), errors.TraceID(traceID))
			problem.WriteResponse(w)
			return
		}

		beach, err := beachService.GetByID(ctx, beachID)
		if err != nil {
			problem := errors.NewProblemReport(http.StatusNotFound, "notfound", errors.Detail("no such beach"), errors.TraceID(traceID))
			problem.WriteResponse(w)
			return
		}

		beachJSON, err := json.Marshal(beach)

		body := []byte("{\"data\":" + string(beachJSON) + "}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(body)
	})
}

func NewRetrieveBeachesHandler(ctx context.Context, beachService beaches.BeachService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		_, span := tracer.Start(r.Context(), "retrieve-beaches")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		traceID, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), r.Context())

		fields := urlValueAsSlice(r.URL.Query(), "fields")

		allBeaches := beachService.GetAll(ctx)

		const geoJSONContentType string = "application/geo+json"

		acceptedContentType := "application/json"
		if len(r.Header["Accept"]) > 0 {
			acceptHeader := r.Header["Accept"][0]
			if acceptHeader != "" && strings.HasPrefix(acceptHeader, geoJSONContentType) {
				acceptedContentType = geoJSONContentType
			}
		}

		waterqualityMapper := func(b *beaches.Beach) any {
			if b.WaterQuality != nil && len(*b.WaterQuality) > 0 {
				return &(*b.WaterQuality)[0]
			}

			return nil
		}

		if acceptedContentType == geoJSONContentType {
			locationMapper := func(b *beaches.Beach) any { return b.Location }

			fields = append([]string{"type", "name", "location"}, fields...)
			beachGeoJSON, err := marshalBeachToJSON(
				allBeaches,
				newBeachGeoJSONMapper(
					newBeachMapper(fields, locationMapper, waterqualityMapper),
				))
			if err != nil {
				err := fmt.Errorf("failed to marshal beach list to geo json: %s", err.Error())
				logger.Error("marshalling error", slog.String("err", err.Error()))
				problem := errors.NewProblemReport(http.StatusInternalServerError, "internalerror", errors.Detail(err.Error()), errors.TraceID(traceID))
				problem.WriteResponse(w)
				return
			}

			body := "{\"type\":\"FeatureCollection\", \"features\": " + string(beachGeoJSON) + "}"

			w.Header().Add("Content-Type", acceptedContentType)
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(body))

		} else {
			locationMapper := func(b *beaches.Beach) any {
				return domain.NewPoint(b.Location.Coordinates[0][0][0][1], b.Location.Coordinates[0][0][0][0])
			}

			fields := append([]string{"id", "name", "location"}, fields...)
			beachJSON, err := marshalBeachToJSON(
				allBeaches,
				newBeachMapper(fields, locationMapper, waterqualityMapper),
			)
			if err != nil {
				err := fmt.Errorf("failed to marshal beach list to json: %s", err.Error())
				logger.Error("marshalling error", slog.String("err", err.Error()))
				problem := errors.NewProblemReport(http.StatusInternalServerError, "internalerror", errors.Detail(err.Error()), errors.TraceID(traceID))
				problem.WriteResponse(w)
				return
			}

			body := "{\"data\":" + string(beachJSON) + "}"

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(body))
		}
	})
}

type BeachMapperFunc func(*beaches.Beach) ([]byte, error)

func newBeachGeoJSONMapper(baseMapper BeachMapperFunc) BeachMapperFunc {
	return func(b *beaches.Beach) ([]byte, error) {
		body, err := baseMapper(b)
		if err != nil {
			return nil, err
		}

		var props any
		json.Unmarshal(body, &props)

		feature := struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Geometry   any    `json:"geometry"`
			Properties any    `json:"properties"`
		}{"Feature", b.ID, b.Location, props}

		return json.Marshal(&feature)
	}
}

func marshalBeachToJSON(allBeaches []beaches.Beach, mapper BeachMapperFunc) ([]byte, error) {
	beachCount := len(allBeaches)

	if beachCount == 0 {
		return []byte("[]"), nil
	}

	backingBuffer := make([]byte, 0, 1024*1024)
	buffer := bytes.NewBuffer(backingBuffer)

	beachBytes, err := mapper(&allBeaches[0])
	if err != nil {
		return nil, err
	}

	buffer.Write([]byte("["))
	buffer.Write(beachBytes)

	for index := 1; index < beachCount; index++ {
		beachBytes, err := mapper(&allBeaches[index])
		if err != nil {
			return nil, err
		}

		buffer.Write([]byte(","))
		buffer.Write(beachBytes)
	}

	buffer.Write([]byte("]"))

	return buffer.Bytes(), nil
}

func newBeachMapper(fields []string, location, wq func(*beaches.Beach) any) BeachMapperFunc {

	omitempty := func(v any) any {
		switch value := v.(type) {
		case []string:
			if len(value) == 0 || (len(value) == 1 && len(value[0]) == 0) {
				return nil
			}
		case string:
			if len(value) == 0 {
				return nil
			}
		}

		return v
	}

	mappers := map[string]func(*beaches.Beach) (string, any){
		"id":           func(b *beaches.Beach) (string, any) { return "id", b.ID },
		"type":         func(b *beaches.Beach) (string, any) { return "type", "Beach" },
		"name":         func(b *beaches.Beach) (string, any) { return "name", b.Name },
		"description":  func(b *beaches.Beach) (string, any) { return "description", b.Description },
		"location":     func(b *beaches.Beach) (string, any) { return "location", location(b) },
		"waterquality": func(b *beaches.Beach) (string, any) { return "waterQuality", wq(b) },
		"seealso":      func(b *beaches.Beach) (string, any) { return "seeAlso", omitempty(b.SeeAlso) },
		"source":       func(b *beaches.Beach) (string, any) { return "source", omitempty(b.Source) },
	}

	return func(b *beaches.Beach) ([]byte, error) {
		result := map[string]any{}
		for _, f := range fields {
			mapper, ok := mappers[f]
			if !ok {
				return nil, fmt.Errorf("unknown field: %s", f)
			}
			key, value := mapper(b)
			if propertyIsNotNil(value) {
				result[key] = value
			}
		}

		return json.Marshal(&result)
	}

}
