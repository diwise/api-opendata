package datasets

import (
	"fmt"
	"net/http"

	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
)

func NewRetrieveTemperaturesHandler(log logging.Logger, contextBrokerURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		temps, _ := getSomeTemperatures()
		for _, t := range temps {
			fmt.Printf("temp: %f\n", t.Value)
		}

		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte{})

	})
}

// TODO: Refaktorisera och flytta till domänlagret
type Temp struct {
	Value float64
}

func getSomeTemperatures() ([]Temp, error) {
	return nil, fmt.Errorf("not implemented")
}
