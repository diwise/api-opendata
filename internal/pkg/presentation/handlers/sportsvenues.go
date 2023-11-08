package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/application/services/sportsvenues"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
)

func NewRetrieveSportsVenueByIDHandler(ctx context.Context, sfsvc sportsvenues.SportsVenueService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-sportsvenue-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		sportsvenueID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if sportsvenueID == "" {
			err = fmt.Errorf("no sports venue is supplied in query")
			log.Error("bad request", slog.String("err", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		venue, err := sfsvc.GetByID(sportsvenueID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		responseBody, err := json.Marshal(venue)
		if err != nil {
			log.Error("failed to marshal sports venue to json", slog.String("err", err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		responseBody = []byte("{\"data\":" + string(responseBody) + "}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(responseBody)
	})
}

func NewRetrieveSportsVenuesHandler(ctx context.Context, sfsvc sportsvenues.SportsVenueService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-sportsvenues")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		categories := urlValueAsSlice(r.URL.Query(), "categories")
		fields := urlValueAsSlice(r.URL.Query(), "fields")

		sportsvenues := sfsvc.GetAll(categories)

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
				log.Error("failed to marshal sportsvenues list to geo json", slog.String("err", err.Error()))
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
				log.Error("failed to marshal sportsvenues list to json", slog.String("err", err.Error()))
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

	mappers := map[string]func(*domain.SportsVenue) (string, any){
		"id":           func(sv *domain.SportsVenue) (string, any) { return "id", sv.ID },
		"type":         func(sv *domain.SportsVenue) (string, any) { return "type", "SportsVenue" },
		"name":         func(sv *domain.SportsVenue) (string, any) { return "name", sv.Name },
		"description":  func(sv *domain.SportsVenue) (string, any) { return "description", sv.Description },
		"location":     func(sv *domain.SportsVenue) (string, any) { return "location", location(sv) },
		"categories":   func(sv *domain.SportsVenue) (string, any) { return "categories", sv.Categories },
		"datecreated":  func(sv *domain.SportsVenue) (string, any) { return "dateCreated", *sv.DateCreated },
		"datemodified": func(sv *domain.SportsVenue) (string, any) { return "dateModified", *sv.DateModified },
		"publicaccess": func(sv *domain.SportsVenue) (string, any) { return "publicAccess", omitempty(sv.PublicAccess) },
		"seealso":      func(sv *domain.SportsVenue) (string, any) { return "seeAlso", omitempty(sv.SeeAlso) },
		"source":       func(sv *domain.SportsVenue) (string, any) { return "source", sv.Source },
		"managedby":    func(sv *domain.SportsVenue) (string, any) { return "managedBy", sv.ManagedBy },
		"owner":        func(sv *domain.SportsVenue) (string, any) { return "owner", sv.Owner },
	}

	return func(sv *domain.SportsVenue) ([]byte, error) {
		result := map[string]any{}
		for _, f := range fields {
			mapper, ok := mappers[f]
			if !ok {
				return nil, fmt.Errorf("unknown field: %s", f)
			}
			key, value := mapper(sv)
			if propertyIsNotNil(value) {
				result[key] = value
			}
		}

		return json.Marshal(&result)
	}
}

// TODO: Explain the peculiarities of nil interfaces to Go newcomers ...
func propertyIsNotNil(v any) bool {
	if v == nil {
		return false
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return !reflect.ValueOf(v).IsNil()
	}
	return true
}
