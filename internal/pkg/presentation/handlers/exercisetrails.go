package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/diwise/api-opendata/internal/pkg/application/services/exercisetrails"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func NewRetrieveExerciseTrailByIDHandler(logger zerolog.Logger, trailService exercisetrails.ExerciseTrailService) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-trail-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		trailID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if trailID == "" {
			err = fmt.Errorf("no exercise trail is supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		trail, err := trailService.GetByID(trailID)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		const gpxContentType string = "application/gpx+xml"

		acceptedContentType := "application/json"
		acceptHeader := r.Header["Accept"][0]
		if acceptHeader != "" && strings.HasPrefix(acceptHeader, gpxContentType) {
			acceptedContentType = gpxContentType
		}

		responseBody := []byte{}

		if acceptedContentType == "application/json" {
			responseBody, err = json.Marshal(trail)
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal trail to json")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			responseBody = []byte("{\"data\":" + string(responseBody) + "}")
		} else if acceptedContentType == gpxContentType {
			responseBody, err = convertTrailToGPX(trail)
			if err != nil {
				log.Error().Err(err).Msg("failed to create gpx file from trail")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			filename := strings.ReplaceAll(strings.ToLower(trail.Name), " ", "_")
			w.Header().Add("Content-Disposition", "attachment; filename=\""+filename+".gpx\"")
		}

		w.Header().Add("Content-Type", acceptedContentType)
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(responseBody)
	})
}

func urlValueAsSlice(query url.Values, param string) []string {
	value := query.Get(param)
	if value == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func NewRetrieveExerciseTrailsHandler(logger zerolog.Logger, trailService exercisetrails.ExerciseTrailService) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-trails")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		categories := urlValueAsSlice(r.URL.Query(), "categories")
		fields := urlValueAsSlice(r.URL.Query(), "fields")

		trails := trailService.GetAll(categories)

		const geoJSONContentType string = "application/geo+json"

		acceptedContentType := "application/json"
		if len(r.Header["Accept"]) > 0 {
			acceptHeader := r.Header["Accept"][0]
			if acceptHeader != "" && strings.HasPrefix(acceptHeader, geoJSONContentType) {
				acceptedContentType = geoJSONContentType
			}
		}

		if acceptedContentType == geoJSONContentType {
			locationMapper := func(t *domain.ExerciseTrail) any { return t.Location }

			fields := append([]string{"type", "name", "categories", "length"}, fields...)
			trailsGeoJSON, err := marshalTrailsToJSON(
				trails,
				newGeoJSONMapper(
					newTrailMapper(fields, locationMapper),
				))
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal trail list to geo json")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := "{\"type\":\"FeatureCollection\", \"features\": " + string(trailsGeoJSON) + "}"

			w.Header().Add("Content-Type", acceptedContentType)
			w.Header().Add("Cache-Control", "max-age=600")
			w.Write([]byte(response))
		} else {
			locationMapper := func(t *domain.ExerciseTrail) any {
				return domain.NewPoint(t.Location.Coordinates[0][1], t.Location.Coordinates[0][0])
			}

			fields := append([]string{"id", "name", "categories", "length"}, fields...)
			trailsJSON, err := marshalTrailsToJSON(trails, newTrailMapper(fields, locationMapper))

			if err != nil {
				log.Error().Err(err).Msg("failed to marshal trail list to json")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := "{\"data\":" + string(trailsJSON) + "}"

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(response))
		}
	})
}

type TrailMapperFunc func(*domain.ExerciseTrail) ([]byte, error)

func newGeoJSONMapper(baseMapper TrailMapperFunc) TrailMapperFunc {

	return func(t *domain.ExerciseTrail) ([]byte, error) {
		body, err := baseMapper(t)
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
		}{"Feature", t.ID, t.Location, props}

		return json.Marshal(&feature)
	}

}

func newTrailMapper(fields []string, location func(*domain.ExerciseTrail) any) TrailMapperFunc {

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

	mappers := map[string]func(*domain.ExerciseTrail) (string, any){
		"id":              func(t *domain.ExerciseTrail) (string, any) { return "id", t.ID },
		"type":            func(t *domain.ExerciseTrail) (string, any) { return "type", "ExerciseTrail" },
		"name":            func(t *domain.ExerciseTrail) (string, any) { return "name", t.Name },
		"description":     func(t *domain.ExerciseTrail) (string, any) { return "description", t.Description },
		"location":        func(t *domain.ExerciseTrail) (string, any) { return "location", location(t) },
		"categories":      func(t *domain.ExerciseTrail) (string, any) { return "categories", t.Categories },
		"length":          func(t *domain.ExerciseTrail) (string, any) { return "length", t.Length },
		"difficulty":      func(t *domain.ExerciseTrail) (string, any) { return "difficulty", t.Difficulty },
		"paymentrequired": func(t *domain.ExerciseTrail) (string, any) { return "paymentRequired", t.PaymentRequired },
		"publicaccess":    func(t *domain.ExerciseTrail) (string, any) { return "publicAccess", omitempty(t.PublicAccess) },
		"status":          func(t *domain.ExerciseTrail) (string, any) { return "status", t.Status },
		"datelastpreparation": func(t *domain.ExerciseTrail) (string, any) {
			return "dateLastPreparation", omitempty(t.DateLastPreparation)
		},
		"source":     func(t *domain.ExerciseTrail) (string, any) { return "source", t.Source },
		"areaserved": func(t *domain.ExerciseTrail) (string, any) { return "areaServed", t.AreaServed },
		"managedby":  func(t *domain.ExerciseTrail) (string, any) { return "managedBy", t.ManagedBy },
		"owner":      func(t *domain.ExerciseTrail) (string, any) { return "owner", t.Owner },
		"seealso":    func(t *domain.ExerciseTrail) (string, any) { return "seeAlso", omitempty(t.SeeAlso) },
	}

	return func(t *domain.ExerciseTrail) ([]byte, error) {
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

func marshalTrailsToJSON(trails []domain.ExerciseTrail, mapper TrailMapperFunc) ([]byte, error) {
	trailCount := len(trails)

	if trailCount == 0 {
		return []byte("[]"), nil
	}

	backingBuffer := make([]byte, 0, 1024*1024)
	buffer := bytes.NewBuffer(backingBuffer)

	trailBytes, err := mapper(&trails[0])
	if err != nil {
		return nil, err
	}

	buffer.Write([]byte("["))
	buffer.Write(trailBytes)

	for index := 1; index < trailCount; index++ {
		trailBytes, err := mapper(&trails[index])
		if err != nil {
			return nil, err
		}

		buffer.Write([]byte(","))
		buffer.Write(trailBytes)
	}

	buffer.Write([]byte("]"))

	return buffer.Bytes(), nil
}
