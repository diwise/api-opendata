package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/diwise/api-opendata/internal/pkg/application/services/waterquality"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/api")

func NewRetrieveWaterQualityHandler(ctx context.Context, svc waterquality.WaterQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-water-qualities")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		fields := urlValueAsSlice(r.URL.Query(), "fields")

		maxDistance := r.URL.Query().Get("maxDistance")
		var distance int64
		if maxDistance != "" {
			distance, err = strconv.ParseInt(maxDistance, 0, 64)
			if err != nil {
				log.Error("failed to parse distance from query parameters", slog.String("error", err.Error()))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		coordinates := r.URL.Query().Get("coordinates")
		var longitude, latitude float64
		if coordinates != "" {
			coords := strings.Split(coordinates, ",")

			longitude, _ = strconv.ParseFloat(coords[0], 64)
			latitude, _ = strconv.ParseFloat(coords[1], 64)
		}

		const geoJSONContentType string = "application/geo+json"

		acceptedContentType := "application/json"
		if len(r.Header["Accept"]) > 0 {
			acceptHeader := r.Header["Accept"][0]
			if acceptHeader != "" && strings.HasPrefix(acceptHeader, geoJSONContentType) {
				acceptedContentType = geoJSONContentType
			}
		}

		var wqos []domain.WaterQuality

		if distance != 0 {
			wqos, err = svc.GetAllNearPointWithinTimespan(ctx, waterquality.NewPoint(latitude, longitude), int(distance), time.Time{}, time.Time{})
		} else {
			wqos = svc.GetAll(ctx)
		}

		if acceptedContentType == geoJSONContentType {
			locationMapper := func(wqo *domain.WaterQuality) any { return wqo.Location }

			fields := append([]string{"type", "location", "temperature", "dateobserved"}, fields...)
			wqoGeoJSON, err := marshalWQOToJSON(
				wqos,
				newWQOGeoJSONMapper(
					newWQOMapper(fields, locationMapper),
				))
			if err != nil {
				log.Error("failed to marshal beach list to GeoJson", slog.String("error", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			body := "{\"type\":\"FeatureCollection\", \"features\": " + string(wqoGeoJSON) + "}"

			w.Header().Add("Content-Type", acceptedContentType)
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(body))

		} else {

			wqosBytes, err := json.Marshal(wqos)
			if err != nil {
				log.Error("failed to marshal water quality into json", slog.String("error", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			waterQualityJSON := "{\"data\":" + string(wqosBytes) + "}"

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Cache-Control", "max-age=3600")
			w.Write([]byte(waterQualityJSON))
		}
	})
}

func NewRetrieveWaterQualityByIDHandler(ctx context.Context, svc waterquality.WaterQualityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-water-qualities")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		waterqualityID, err := url.QueryUnescape(chi.URLParam(r, "id"))
		if waterqualityID == "" {
			err = fmt.Errorf("no water quality id is supplied in query")
			log.Error("bad request", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			log.Error("failed to parse parameters from query", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		from := time.Time{}
		to := time.Time{}

		if len(values) != 0 {
			if values.Get("from") != "" {
				from, err = time.Parse(time.RFC3339, values.Get("from"))
				if err != nil {
					log.Error("time parameter from is incorrect format", slog.String("error", err.Error()))
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}
			if values.Get("to") != "" {
				to, err = time.Parse(time.RFC3339, values.Get("to"))
				if err != nil {
					log.Error("time parameter to is incorrect format", slog.String("error", err.Error()))
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}
		}

		wqo, err := svc.GetByID(ctx, waterqualityID, from, to)
		if err != nil {
			log.Error("no water quality found", slog.String("error", err.Error()), "id", waterqualityID)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		body, err := json.Marshal(wqo)
		if err != nil {
			log.Error("failed to marshal water quality", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		waterQualityJSON := "{\"data\":" + string(body) + "}"

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Cache-Control", "max-age=3600")
		w.Write([]byte(waterQualityJSON))

	})
}

type WaterQualityMapperFunc func(*domain.WaterQuality) ([]byte, error)

func newWQOGeoJSONMapper(baseMapper WaterQualityMapperFunc) WaterQualityMapperFunc {

	return func(sf *domain.WaterQuality) ([]byte, error) {
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

func marshalWQOToJSON(wqos []domain.WaterQuality, mapper WaterQualityMapperFunc) ([]byte, error) {
	wqoCount := len(wqos)

	if wqoCount == 0 {
		return []byte("[]"), nil
	}

	backingBuffer := make([]byte, 0, 1024*1024)
	buffer := bytes.NewBuffer(backingBuffer)

	wqoBytes, err := mapper(&wqos[0])
	if err != nil {
		return nil, err
	}

	buffer.Write([]byte("["))
	buffer.Write(wqoBytes)

	for index := 1; index < wqoCount; index++ {
		wqoBytes, err := mapper(&wqos[index])
		if err != nil {
			return nil, err
		}

		buffer.Write([]byte(","))
		buffer.Write(wqoBytes)
	}

	buffer.Write([]byte("]"))

	return buffer.Bytes(), nil
}

func newWQOMapper(fields []string, location func(*domain.WaterQuality) any) WaterQualityMapperFunc {

	mappers := map[string]func(*domain.WaterQuality) (string, any){
		"id":           func(sf *domain.WaterQuality) (string, any) { return "id", sf.ID },
		"type":         func(sf *domain.WaterQuality) (string, any) { return "type", "WaterQualityObserved" },
		"location":     func(sf *domain.WaterQuality) (string, any) { return "location", location(sf) },
		"source":       func(t *domain.WaterQuality) (string, any) { return "source", t.Source },
		"temperature":  func(t *domain.WaterQuality) (string, any) { return "temperature", t.Temperature },
		"dateobserved": func(t *domain.WaterQuality) (string, any) { return "dateObserved", t.DateObserved },
	}

	return func(t *domain.WaterQuality) ([]byte, error) {
		result := map[string]any{}
		for _, f := range fields {
			mapper, ok := mappers[f]
			if !ok {
				return nil, fmt.Errorf("unknown field: %s", f)
			}
			key, value := mapper(t)
			if propertyIsNotNil(value) {
				result[key] = value
			}
		}

		return json.Marshal(&result)
	}
}
