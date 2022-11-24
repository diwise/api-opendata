package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsvenues"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func NewRetrieveSportsVenueByIDHandler(logger zerolog.Logger, sfsvc sportsvenues.SportsVenueService) http.HandlerFunc {
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
			log.Error().Err(err).Msg("failed to marshal sports field to json")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		responseBody = []byte("{\"data\":" + string(responseBody) + "}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(responseBody)
	})
}

func NewRetrieveSportsVenuesHandler(logger zerolog.Logger, sfsvc sportsvenues.SportsVenueService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-sportsvenues")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		requestedFields := r.URL.Query().Get("fields")
		fields := []string{}
		if requestedFields != "" {
			fields = strings.Split(requestedFields, ",")
		}

		sportsvenues := sfsvc.GetAll()

		const geoJSONContentType string = "application/geo+json"

		acceptedContentType := "application/json"
		if len(r.Header["Accept"]) > 0 {
			acceptHeader := r.Header["Accept"][0]
			if acceptHeader != "" && strings.HasPrefix(acceptHeader, geoJSONContentType) {
				acceptedContentType = geoJSONContentType
			}
		}

		if acceptedContentType == geoJSONContentType {
			locationMapper := func(sf *domain.SportsVenue) any { return sf.Location }

			fields := append([]string{"type", "name", "categories"}, fields...)
			sportsvenuesGeoJSON, err := marshalSportsVenuesToJSON(
				sportsvenues,
				newSportsVenuesGeoJSONMapper(
					newSportsVenuesMapper(fields, locationMapper),
				))
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal sportsvenues list to geo json")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := "{\"type\":\"FeatureCollection\", \"features\": " + string(sportsvenuesGeoJSON) + "}"

			w.Header().Add("Content-Type", acceptedContentType)
			w.Header().Add("Cache-Control", "max-age=600")
			w.Write([]byte(response))
		} else {
			locationMapper := func(t *domain.SportsVenue) any {
				return domain.NewPoint(t.Location.Coordinates[0][0][0][1], t.Location.Coordinates[0][0][0][0])
			}

			fields := append([]string{"id", "name", "categories", "location"}, fields...)
			sportsvenuesJSON, err := marshalSportsVenuesToJSON(sportsvenues, newSportsVenuesMapper(fields, locationMapper))

			if err != nil {
				log.Error().Err(err).Msg("failed to marshal sportsvenues list to json")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := "{\"data\":" + string(sportsvenuesJSON) + "}"

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(response))
		}
	})
}

type SportsVenuesMapperFunc func(*domain.SportsVenue) ([]byte, error)

func newSportsVenuesGeoJSONMapper(baseMapper SportsVenuesMapperFunc) SportsVenuesMapperFunc {

	return func(sf *domain.SportsVenue) ([]byte, error) {
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

func marshalSportsVenuesToJSON(sportsvenues []domain.SportsVenue, mapper SportsVenuesMapperFunc) ([]byte, error) {
	sportsvenuesCount := len(sportsvenues)

	if sportsvenuesCount == 0 {
		return []byte("[]"), nil
	}

	backingBuffer := make([]byte, 0, 1024*1024)
	buffer := bytes.NewBuffer(backingBuffer)

	sportsvenuesBytes, err := mapper(&sportsvenues[0])
	if err != nil {
		return nil, err
	}

	buffer.Write([]byte("["))
	buffer.Write(sportsvenuesBytes)

	for index := 1; index < sportsvenuesCount; index++ {
		sportsvenuesBytes, err := mapper(&sportsvenues[index])
		if err != nil {
			return nil, err
		}

		buffer.Write([]byte(","))
		buffer.Write(sportsvenuesBytes)
	}

	buffer.Write([]byte("]"))

	return buffer.Bytes(), nil
}

func newSportsVenuesMapper(fields []string, location func(*domain.SportsVenue) any) SportsVenuesMapperFunc {

	mappers := map[string]func(*domain.SportsVenue) (string, any){
		"id":           func(sf *domain.SportsVenue) (string, any) { return "id", sf.ID },
		"type":         func(sf *domain.SportsVenue) (string, any) { return "type", "SportsVenue" },
		"name":         func(sf *domain.SportsVenue) (string, any) { return "name", sf.Name },
		"description":  func(sf *domain.SportsVenue) (string, any) { return "description", sf.Description },
		"location":     func(sf *domain.SportsVenue) (string, any) { return "location", location(sf) },
		"categories":   func(sf *domain.SportsVenue) (string, any) { return "categories", sf.Categories },
		"datecreated":  func(sf *domain.SportsVenue) (string, any) { return "dateCreated", *sf.DateCreated },
		"datemodified": func(sf *domain.SportsVenue) (string, any) { return "dateModified", *sf.DateModified },
		"seealso":      func(sf *domain.SportsVenue) (string, any) { return "seeAlso", sf.SeeAlso },
		"source":       func(t *domain.SportsVenue) (string, any) { return "source", t.Source },
	}

	return func(t *domain.SportsVenue) ([]byte, error) {
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
