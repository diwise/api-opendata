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

func NewRetrieveTemperaturesHandler(log logging.Logger, contextBrokerURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		response := &TempResponse{}

		tempRespItem := &TempResponseItem{}
		tempRespItem.ID = "gurka"

		response.Items = append(response.Items, *tempRespItem)

		temps, _ := getSomeTemperatures()

		tempRespItem.Average, _ = calculateAverage(temps)

		for _, t := range temps {
			trv := &TempResponseValue{}
			trv.Value = fmt.Sprintf("%.2f", t.Value)

			tempRespItem.Values = append(tempRespItem.Values, *trv)

			//value of temp, add to TempResponseValue.Value, append to array of Values in TempResponseItem.
			//TempResponseItem added to TempResponse.

			fmt.Printf("temp: %f\n", t.Value)
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

func getSomeTemperatures() ([]Temp, error) {
	return []Temp{}, fmt.Errorf("not implemented")
}
