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
	Unit    string              `json:"unit"`
}

type TempResponse struct {
	Items []TempResponseItem `json:"items"`
}

func NewRetrieveTemperaturesHandler(log logging.Logger, svc TempService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		response := &TempResponse{}

		temps, _ := svc.Get()
		for _, t := range temps {
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

func (svc ts) Get() ([]Temp, error) {
	return getSomeTemperatures(svc.contextBrokerURL)
}

func NewTempService(contextBrokerURL string) TempService {
	return &ts{contextBrokerURL: contextBrokerURL}
}

func getSomeTemperatures(contextBrokerURL string) ([]Temp, error) {
	return nil, fmt.Errorf("not implemented")
}
