package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
)

type TempService interface {
	Get(from, to time.Time) ([]domain.Temperature, error)
}

func NewTempService(contextBrokerURL string) TempService {
	return &ts{contextBrokerURL: contextBrokerURL}
}

type ts struct {
	contextBrokerURL string
}

func (svc ts) Get(from, to time.Time) ([]domain.Temperature, error) {

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

	temps := []domain.Temperature{}

	for _, wo := range wos {
		t := domain.Temperature{
			Id:    wo.RefDevice.Object,
			Value: wo.Temperature.Value,
			When:  wo.DateObserved.Value.Value,
		}
		temps = append(temps, t)
	}

	return temps, err
}
