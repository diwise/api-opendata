package datasets

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
	var err error

	url := fmt.Sprintf("%s/ngsi-ld/v1/entities?type=WeatherObserved&attrs=temperature&timerel=between&timeAt=2021-10-01T00:00:00Z&endTimeAt=2021-10-20T00:00:00Z", contextBrokerURL)

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
			Value: wo.Temperature.Value,
		}
		temps = append(temps, t)
	}

	return temps, err
}
