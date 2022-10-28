package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsfields"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func NewRetrieveSportsFieldByIDHandler(logger zerolog.Logger, sfsvc sportsfields.SportsFieldService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-sportsfield-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		sportsfieldID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if sportsfieldID == "" {
			err = fmt.Errorf("no sports field is supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		sportsfield, err := sfsvc.GetByID(sportsfieldID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		responseBody, err := json.Marshal(sportsfield)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal trail to json")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		responseBody = []byte("{\"data\":" + string(responseBody) + "}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(responseBody)
	})
}

func NewRetrieveSportsFieldsHandler(logger zerolog.Logger, sfsvc sportsfields.SportsFieldService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-sportsfields")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		requestedFields := r.URL.Query().Get("fields")
		fields := []string{}
		if requestedFields != "" {
			fields = strings.Split(requestedFields, ",")
		}

		sportsfields := sfsvc.GetAll()

		const geoJSONContentType string = "application/geo+json"

		acceptedContentType := "application/json"
		if len(r.Header["Accept"]) > 0 {
			acceptHeader := r.Header["Accept"][0]
			if acceptHeader != "" && strings.HasPrefix(acceptHeader, geoJSONContentType) {
				acceptedContentType = geoJSONContentType
			}
		}

		if acceptedContentType == geoJSONContentType {
			locationMapper := func(sf *domain.SportsField) any { return sf.Location }

			fields := append([]string{"type", "name", "categories"}, fields...)
			sportsfieldsGeoJSON, err := marshalSportsFieldsToJSON(
				sportsfields,
				newSportsFieldsGeoJSONMapper(
					newSportsFieldsMapper(fields, locationMapper),
				))
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal sportsfields list to geo json")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := "{\"type\":\"FeatureCollection\", \"features\": " + string(sportsfieldsGeoJSON) + "}"

			w.Header().Add("Content-Type", acceptedContentType)
			w.Header().Add("Cache-Control", "max-age=600")
			w.Write([]byte(response))
		} else {
			locationMapper := func(t *domain.SportsField) any {
				return domain.NewPoint(t.Location.Coordinates[0][0][0][1], t.Location.Coordinates[0][0][0][0])
			}

			fields := append([]string{"id", "name", "categories"}, fields...)
			sportsfieldsJSON, err := marshalSportsFieldsToJSON(sportsfields, newSportsFieldsMapper(fields, locationMapper))

			if err != nil {
				log.Error().Err(err).Msg("failed to marshal sportsfields list to json")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := "{\"data\":" + string(sportsfieldsJSON) + "}"

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(response))
		}
	})
}

type SportsFieldsMapperFunc func(*domain.SportsField) ([]byte, error)

func newSportsFieldsGeoJSONMapper(baseMapper SportsFieldsMapperFunc) SportsFieldsMapperFunc {

	return func(sf *domain.SportsField) ([]byte, error) {
		body, err := baseMapper(sf)
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
		}{"Feature", sf.ID, sf.Location, props}

		return json.Marshal(&feature)
	}

}

func marshalSportsFieldsToJSON(sportsfields []domain.SportsField, mapper SportsFieldsMapperFunc) ([]byte, error) {
	sportsfieldsCount := len(sportsfields)

	if sportsfieldsCount == 0 {
		return []byte("[]"), nil
	}

	backingBuffer := make([]byte, 0, 1024*1024)
	buffer := bytes.NewBuffer(backingBuffer)

	sportsfieldsBytes, err := mapper(&sportsfields[0])
	if err != nil {
		return nil, err
	}

	buffer.Write([]byte("["))
	buffer.Write(sportsfieldsBytes)

	for index := 1; index < sportsfieldsCount; index++ {
		sportsfieldsBytes, err := mapper(&sportsfields[index])
		if err != nil {
			return nil, err
		}

		buffer.Write([]byte(","))
		buffer.Write(sportsfieldsBytes)
	}

	buffer.Write([]byte("]"))

	return buffer.Bytes(), nil
}

func newSportsFieldsMapper(fields []string, location func(*domain.SportsField) any) SportsFieldsMapperFunc {

	omitempty := func(s string) any {
		if s == "" {
			return nil
		}

		return s
	}

	mappers := map[string]func(*domain.SportsField) (string, any){
		"id":          func(sf *domain.SportsField) (string, any) { return "id", sf.ID },
		"type":        func(sf *domain.SportsField) (string, any) { return "type", "SportsField" },
		"name":        func(sf *domain.SportsField) (string, any) { return "name", sf.Name },
		"description": func(sf *domain.SportsField) (string, any) { return "description", sf.Description },
		"location":    func(sf *domain.SportsField) (string, any) { return "location", location(sf) },
		"categories":  func(sf *domain.SportsField) (string, any) { return "categories", sf.Categories },
		"dateLastPrepared": func(sf *domain.SportsField) (string, any) {
			return "dateLastPrepared", omitempty(*sf.DateLastPrepared)
		},
		"source": func(t *domain.SportsField) (string, any) { return "source", t.Source },
	}

	return func(t *domain.SportsField) ([]byte, error) {
		result := map[string]any{}
		for _, f := range fields {
			mapper, ok := mappers[f]
			if !ok {
				return nil, fmt.Errorf("unknown field: %s", f)
			}
			key, value := mapper(t)
			if value != nil {
				result[key] = value
			}
		}

		return json.Marshal(&result)
	}
}
