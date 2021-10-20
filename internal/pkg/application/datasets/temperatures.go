package datasets

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
)

type TempResponse struct {
	Items []struct {
	} `json:"items"`
}

func NewRetrieveTemperaturesHandler(log logging.Logger, contextBrokerURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		response := &TempResponse{}

		temps, _ := getSomeTemperatures()
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

func getSomeTemperatures() ([]Temp, error) {
	return nil, fmt.Errorf("not implemented")
}
