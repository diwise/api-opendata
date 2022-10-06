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
			err = fmt.Errorf("no exerciset trail id supplied in query")
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

func NewRetrieveExerciseTrailsHandler(logger zerolog.Logger, trailService exercisetrails.ExerciseTrailService) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-trails")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		requestedFields := r.URL.Query().Get("fields")
		fields := []string{}
		if requestedFields != "" {
			fields = strings.Split(requestedFields, ",")
		}

		trails := trailService.GetAll()
		locationMapper := func(t *domain.ExerciseTrail) any {
			return domain.NewPoint(t.Location.Coordinates[0][1], t.Location.Coordinates[0][0])
		}
		trailsJSON, err := marshalTrailsToJSON(trails, newTrailMapper(fields, locationMapper))

		if err != nil {
			_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)
			log.Error().Err(err).Msg("failed to marshal trail list to json")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := "{\"data\":" + string(trailsJSON) + "}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(response))
	})
}

type TrailMapperFunc func(*domain.ExerciseTrail) ([]byte, error)

func newTrailMapper(extraFields []string, location func(*domain.ExerciseTrail) any) TrailMapperFunc {

	omitempty := func(s string) any {
		if s == "" {
			return nil
		}

		return s
	}

	mappers := map[string]func(*domain.ExerciseTrail) (string, any){
		"id":              func(t *domain.ExerciseTrail) (string, any) { return "id", t.ID },
		"name":            func(t *domain.ExerciseTrail) (string, any) { return "name", t.Name },
		"description":     func(t *domain.ExerciseTrail) (string, any) { return "description", t.Description },
		"location":        func(t *domain.ExerciseTrail) (string, any) { return "location", location(t) },
		"categories":      func(t *domain.ExerciseTrail) (string, any) { return "categories", t.Categories },
		"length":          func(t *domain.ExerciseTrail) (string, any) { return "length", t.Length },
		"difficulty":      func(t *domain.ExerciseTrail) (string, any) { return "difficulty", t.Difficulty },
		"paymentrequired": func(t *domain.ExerciseTrail) (string, any) { return "paymentRequired", t.PaymentRequired },
		"status":          func(t *domain.ExerciseTrail) (string, any) { return "status", t.Status },
		"datelastpreparation": func(t *domain.ExerciseTrail) (string, any) {
			return "dateLastPreparation", omitempty(t.DateLastPreparation)
		},
		"source":     func(t *domain.ExerciseTrail) (string, any) { return "source", t.Source },
		"areaserved": func(t *domain.ExerciseTrail) (string, any) { return "areaServed", t.AreaServed },
	}

	fields := append(
		[]string{"id", "name", "categories", "length"},
		extraFields...,
	)

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
