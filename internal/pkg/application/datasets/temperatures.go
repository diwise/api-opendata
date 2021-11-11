package datasets

import (
	"encoding/json"
	"net/http"
	"time"

	services "github.com/diwise/api-opendata/internal/pkg/application/services/temperature"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
)

type TempResponseValue struct {
	Average *float64   `json:"avg,omitempty"`
	Max     *float64   `json:"max,omitempty"`
	Min     *float64   `json:"min,omitempty"`
	Value   *float64   `json:"val,omitempty"`
	When    *time.Time `json:"when,omitempty"`
	From    *time.Time `json:"from,omitempty"`
	To      *time.Time `json:"to,omitempty"`
}

type TempResponseSensor struct {
	ID     string              `json:"id"`
	Values []TempResponseValue `json:"values"`
}

type TempResponse struct {
	Sensors []TempResponseSensor `json:"sensors"`
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

func NewRetrieveTemperaturesHandler(log logging.Logger, svc services.TempService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		query := svc.Query().Sensor(r.URL.Query().Get("sensor"))

		from, to, err := getTimeParamsFromURL(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("unable to get time range: %s", err.Error())
			return
		}

		query = query.BetweenTimes(from, to)

		if r.URL.Query().Get("options") == "aggregatedValues" {
			methods := r.URL.Query().Get("aggrMethods")
			duration := r.URL.Query().Get("aggrPeriodDuration")
			query = query.Aggregate(duration, methods)
		}

		sensors, err := query.Get()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("unable to get temperatures: %s", err.Error())
			return
		}

		response := &TempResponse{
			Sensors: []TempResponseSensor{},
		}

		for _, s := range sensors {
			sensor := TempResponseSensor{
				ID:     s.Id,
				Values: []TempResponseValue{},
			}

			for _, t := range s.Temperatures {
				value := TempResponseValue{
					Average: t.Average,
					Max:     t.Max,
					Min:     t.Min,
					Value:   t.Value,
					When:    t.When,
					From:    t.From,
					To:      t.To,
				}

				sensor.Values = append(sensor.Values, value)
			}

			response.Sensors = append(response.Sensors, sensor)
		}

		w.Header().Add("Content-Type", "application/json")

		bytes, err := json.MarshalIndent(response, " ", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("unable to marshal results to json: %s", err.Error())
			return
		}

		w.Write(bytes)
	})
}
