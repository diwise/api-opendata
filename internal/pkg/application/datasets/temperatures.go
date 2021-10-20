package datasets

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
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

		tempRespItem := &TempResponseItem{}
		tempRespItem.ID = "gurka"

		temps, _ := svc.Get()

		tempRespItem.Average, _ = calculateAverage(temps)

		for _, t := range temps {
			trv := &TempResponseValue{}
			trv.Value = fmt.Sprintf("%.2f", t.Value)

			tempRespItem.Values = append(tempRespItem.Values, *trv)
		}

		response.Items = append(response.Items, *tempRespItem)

		w.Header().Add("Content-Type", "application/json")

		bytes, err := json.MarshalIndent(response, " ", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

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
	Value float64
}

type TempService interface {
	Get() ([]Temp, error)
}

type ts struct {
	contextBrokerURL string
}

func NewTempService(contextBrokerURL string) TempService {
	return &ts{contextBrokerURL: contextBrokerURL}
}

func (svc ts) Get() ([]Temp, error) {
	return getSomeTemperatures(svc.contextBrokerURL)
}

func getSomeTemperatures(contextBrokerURL string) ([]Temp, error) {
	temps := []Temp{
		{
			Value: 1.1,
		},
		{
			Value: 2.1,
		},
		{
			Value: 3.1,
		},
	}

	return temps, fmt.Errorf("not implemented")
}
