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
	When  string `json:"when"`
}

type TempResponseItem struct {
	ID      string              `json:"id"`
	Values  []TempResponseValue `json:"values"`
	Average string              `json:"average"`
}

type TempResponse struct {
	Items []TempResponseItem `json:"items"`
}

func NewRetrieveTemperaturesHandler(log logging.Logger, svc TempService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		response := &TempResponse{}

		temps, _ := svc.Get(time.Now().UTC().Add(-1*24*time.Hour), time.Now().UTC())

		tempRespItemMap := make(map[string]TempResponseItem)

		for _, t := range temps {

			temp, exist := tempRespItemMap[t.Id]

			if exist {
				temp.Values = append(temp.Values, TempResponseValue{
					Value: fmt.Sprintf("%.2f", t.Value),
					When:  t.When,
				})
				tempRespItemMap[t.Id] = temp
			} else {
				tempRespItem := TempResponseItem{
					ID: t.Id,
					Values: []TempResponseValue{
						{
							Value: fmt.Sprintf("%.2f", t.Value),
							When:  t.When,
						}},
				}

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

		w.Write(bytes)
	})
}

// TODO: Refaktorisera och flytta till domänlagret
type Temp struct {
	Id    string
	Value float64
	When  string
}

type TempService interface {
	Get(from, to time.Time) ([]Temp, error)
}

func NewTempService(contextBrokerURL string) TempService {
	return &ts{contextBrokerURL: contextBrokerURL}
}

type ts struct {
	contextBrokerURL string
}

func (svc ts) Get(from, to time.Time) ([]Temp, error) {

	timeAt := from.Format(time.RFC3339)
	endTimeAt := to.Format(time.RFC3339)

	url := fmt.Sprintf(
		"%s/ngsi-ld/v1/entities?type=WeatherObserved&attrs=temperature&georel=near%%3BmaxDistance==2000&geometry=Point&coordinates=[17.3051555,62.3908926]&timerel=between&timeAt=%s&endTimeAt=%s",
		svc.contextBrokerURL, timeAt, endTimeAt,
	)

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed, status code not ok: %d", response.StatusCode)
	}

	wos := []fiware.WeatherObserved{}
	b, _ := io.ReadAll(response.Body)
	err = json.Unmarshal(b, &wos)

	temps := []Temp{}

	for _, wo := range wos {
		t := Temp{
			Id:    wo.RefDevice.Object,
			Value: wo.Temperature.Value,
			When:  wo.DateObserved.Value.Value,
		}
		temps = append(temps, t)
	}

	return temps, err
}
