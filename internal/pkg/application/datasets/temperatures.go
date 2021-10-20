package datasets

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
)

type TempResponseValue struct {
	Value string `json:"val"`
}

type TempResponseItem struct {
	ID      string              `json:"id"`
	Values  []TempResponseValue `json:"values"`
	Average float64             `json:"average"`
}

type TempResponse struct {
	Items []TempResponseItem `json:"items"`
}

func NewRetrieveTemperaturesHandler(log logging.Logger, svc TempService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		response := &TempResponse{}

		temps, _ := svc.Get(time.Now().UTC().Add(-7*24*time.Hour), time.Now().UTC())

		tempRespItemMap := make(map[string]TempResponseItem)

		for _, t := range temps {

			temp, exist := tempRespItemMap[t.Id]

			if exist {
				temp.Values = append(temp.Values, TempResponseValue{Value: fmt.Sprintf("%.2f", t.Value)})
				tempRespItemMap[t.Id] = temp
			} else {
				tempRespItem := TempResponseItem{}
				tempRespItem.ID = t.Id
				tempRespItem.Average = 12
				tempRespItem.Values = []TempResponseValue{{Value: fmt.Sprintf("%.2f", t.Value)}}

				tempRespItemMap[t.Id] = tempRespItem
			}
		}

		for _, v := range tempRespItemMap {
			response.Items = append(response.Items, v)
		}

		w.Header().Add("Content-Type", "application/json")

		bytes, err := json.MarshalIndent(response, " ", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Println(string(bytes))

		w.Write(bytes)
	})
}

func calculateAverage(temps []Temp) (float64, error) {

	if len(temps) == 0 {
		return 0.0, fmt.Errorf("no temperatures available")
	}

	holder := 0.0

	for _, t := range temps {
		holder += t.Value
	}

	return holder / float64(len(temps)), fmt.Errorf("unexpected error while calculating average")
}

// TODO: Refaktorisera och flytta till domänlagret
type Temp struct {
	Id    string
	Value float64
}

type TempService interface {
	Get(from, to time.Time) ([]Temp, error)
}

type ts struct {
	contextBrokerURL string
}

func (svc ts) Get(from, to time.Time) ([]Temp, error) {
	return getSomeTemperatures(svc.contextBrokerURL, from, to)
}

func NewTempService(contextBrokerURL string) TempService {
	return &ts{contextBrokerURL: contextBrokerURL}
}

func getSomeTemperatures(contextBrokerURL string, from, to time.Time) ([]Temp, error) {
	var err error

	timeAt := from.Format(time.RFC3339)
	endTimeAt := to.Format(time.RFC3339)
	url := fmt.Sprintf("%s/ngsi-ld/v1/entities?type=WeatherObserved&attrs=temperature&timerel=between&timeAt=%s&endTimeAt=%s", contextBrokerURL, timeAt, endTimeAt)

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed, status code not ok: %s", err)
	}
	defer response.Body.Close()

	temps := []Temp{}
	wos := []fiware.WeatherObserved{}

	b, _ := io.ReadAll(response.Body)

	err = json.Unmarshal(b, &wos)

	for _, wo := range wos {
		t := Temp{
			Id:    wo.ID,
			Value: wo.Temperature.Value,
		}
		temps = append(temps, t)
	}

	return temps, err
}
