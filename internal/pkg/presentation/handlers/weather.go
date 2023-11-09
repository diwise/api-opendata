package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"log/slog"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/weather"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
)

func getPointFromURL(r *http.Request) (int64, float64, float64, error) {
	var distance int64 = 1000
	var lon, lat float64
	var err error

	maxDistance := r.URL.Query().Get("maxDistance")
	if maxDistance != "" {
		distance, _ = strconv.ParseInt(maxDistance, 0, 64)
	}

	coordinates := r.URL.Query().Get("coordinates")
	if coordinates != "" {
		coords := strings.Split(coordinates, ",")
		if len(coords) != 2 {
			return 0, 0, 0, fmt.Errorf("invalid coordinates specified")
		}
		lon, err = strconv.ParseFloat(strings.Replace(coords[0], "[", "", 1), 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid coordinates specified")
		}
		lat, err = strconv.ParseFloat(strings.Replace(coords[1], "]", "", 1), 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid coordinates specified")
		}
	} else {
		return 0, 0, 0, fmt.Errorf("no coordinates specified")
	}

	return distance, lat, lon, nil
}

func getTimeParamsFromURL(r *http.Request) (time.Time, time.Time, error) {
	var err error

	startTime := time.Now().UTC().Add(-1 * 24 * time.Hour)
	endTime := time.Now().UTC()

	from := r.URL.Query().Get("timeAt")
	if from != "" {
		startTime, err = time.Parse(time.RFC3339, from)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	to := r.URL.Query().Get("endTimeAt")
	if to != "" {
		endTime, err = time.Parse(time.RFC3339, to)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	return startTime, endTime, nil
}

func NewRetrieveWeatherHandler(ctx context.Context, svc services.WeatherService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-weather")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		dist, lat, lon, err := getPointFromURL(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			err = fmt.Errorf("unable to get point (%w)", err)
			log.Error("bad request", slog.String("err", err.Error()))
			return
		}

		timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		weather, err := svc.Query().NearPoint(dist, lat, lon).Get(timeout)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("unable to get weather")
			log.Error("internal error", slog.String("err", err.Error()))
			return
		}

		w.Header().Add("Content-Type", "application/json")

		bytes, err := json.MarshalIndent(weather, " ", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("unable to marshal results to json (%w)", err)
			log.Error("internal error", slog.String("err", err.Error()))
			return
		}

		w.Write([]byte("{\"data\": " + string(bytes) + "}"))
	})
}

func NewRetrieveWeatherByIDHandler(ctx context.Context, svc services.WeatherService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "retrieve-weather-byid")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

		woID, err := url.QueryUnescape(chi.URLParam(r, "id"))
		if woID == "" {
			err = fmt.Errorf("no weather id is supplied in query")
			log.Error("bad request", slog.String("err", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		from, to, err := getTimeParamsFromURL(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			err = fmt.Errorf("unable to get time range (%w)", err)
			log.Error("bad request", slog.String("err", err.Error()))
			return
		}

		timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		weather, err := svc.Query().ID(woID).BetweenTimes(from, to).GetByID(timeout)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("unable to get weather")
			log.Error("internal error", slog.String("err", err.Error()))
			return
		}

		w.Header().Add("Content-Type", "application/json")

		bytes, err := json.MarshalIndent(weather, " ", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("unable to marshal results to json (%w)", err)
			log.Error("internal error", slog.String("err", err.Error()))
			return
		}

		w.Write([]byte("{\"data\": " + string(bytes) + "}"))
	})
}
