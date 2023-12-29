package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/diwise/api-opendata/internal/pkg/application/services/airquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
)

func NewRetrieveAirQualitiesHandler(ctx context.Context, aqsvc airquality.AirQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-air-qualities")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		fields := urlValueAsSlice(r.URL.Query(), "fields")

		const geoJSONContentType string = "application/geo+json"

		acceptedContentType := "application/json"
		if len(r.Header["Accept"]) > 0 {
			acceptHeader := r.Header["Accept"][0]
			if acceptHeader != "" && strings.HasPrefix(acceptHeader, geoJSONContentType) {
				acceptedContentType = geoJSONContentType
			}
		}

		aqos := aqsvc.GetAll(ctx)

		if acceptedContentType == geoJSONContentType {
			locationMapper := func(aqo *domain.AirQuality) any { return aqo.Location }

			fields := append([]string{"type", "location", "dateobserved"}, fields...)
			aqoGeoJSON, err := marshalAQOToJSON(
				aqos,
				newAQOGeoJSONMapper(
					newAQOMapper(fields, locationMapper),
				))
			if err != nil {
				log.Error("failed to marshal air quality list to geo json", slog.String("err", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			body := "{\"type\":\"FeatureCollection\", \"features\": " + string(aqoGeoJSON) + "}"

			w.Header().Add("Content-Type", acceptedContentType)
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(body))

		} else {

			aqosBytes, err := json.Marshal(aqos)
			if err != nil {
				log.Error("failed to marshal air quality list to json", slog.String("err", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			airQualityJSON := "{\"data\":" + string(aqosBytes) + "}"

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(airQualityJSON))
		}
	})
}

func NewRetrieveAirQualityByIDHandler(ctx context.Context, aqsvc airquality.AirQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-air-quality-by-id")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		airQualityID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if airQualityID == "" {
			err = fmt.Errorf("no air quality id supplied in query")
			log.Error("bad request", "err", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		aq, err := aqsvc.GetByID(ctx, airQualityID)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bodyBytes, _ := json.Marshal(aq)

		body := []byte("{\"data\": " + string(bodyBytes) + "}")

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=600")
		w.Write(body)
	})
}

type AirQualityMapperFunc func(*domain.AirQuality) ([]byte, error)

func newAQOGeoJSONMapper(baseMapper AirQualityMapperFunc) AirQualityMapperFunc {
	return func(sf *domain.AirQuality) ([]byte, error) {
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

func marshalAQOToJSON(aqos []domain.AirQuality, mapper AirQualityMapperFunc) ([]byte, error) {
	aqoCount := len(aqos)

	if aqoCount == 0 {
		return []byte("[]"), nil
	}

	backingBuffer := make([]byte, 0, 1024*1024)
	buffer := bytes.NewBuffer(backingBuffer)

	aqoBytes, err := mapper(&aqos[0])
	if err != nil {
		return nil, err
	}

	buffer.Write([]byte("["))
	buffer.Write(aqoBytes)

	for index := 1; index < aqoCount; index++ {
		aqoBytes, err := mapper(&aqos[index])
		if err != nil {
			return nil, err
		}

		buffer.Write([]byte(","))
		buffer.Write(aqoBytes)
	}

	buffer.Write([]byte("]"))

	return buffer.Bytes(), nil
}

func newAQOMapper(fields []string, location func(*domain.AirQuality) any) AirQualityMapperFunc {
	mappers := map[string]func(*domain.AirQuality) (string, any){}

	return func(aq *domain.AirQuality) ([]byte, error) {
		result := map[string]any{}
		for _, f := range fields {
			mapper, ok := mappers[f]
			if !ok {
				return nil, fmt.Errorf("unknown field: %s", f)
			}
			key, value := mapper(aq)
			if propertyIsNotNil(value) {
				result[key] = value
			}
		}

		return json.Marshal(&result)
	}
}
