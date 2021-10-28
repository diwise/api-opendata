package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
)

type TempService interface {
	Query() TempServiceQuery
}

const (
	aggrMethodAverage string = "avg"
	aggrMethodMaximum string = "max"
	aggrMethodMinimum string = "min"
)

type aggrFunc func([]domain.Temperature, int, int) float64

func average(data []domain.Temperature, from, to int) float64 {
	sum := 0.0
	for i := from; i <= to; i++ {
		sum += data[i].Value
	}
	return sum / float64(to-from+1)
}

type TempServiceQuery interface {
	Aggregate(period, aggregates string) TempServiceQuery
	BetweenTimes(from, to time.Time) TempServiceQuery
	Get() ([]domain.Temperature, error)
}

func NewTempService(contextBrokerURL string) TempService {
	return &ts{contextBrokerURL: contextBrokerURL}
}

type ts struct {
	contextBrokerURL string
}

type tsq struct {
	ts
	from         time.Time
	to           time.Time
	aggregations []aggrFunc
}

func (svc ts) Query() TempServiceQuery {
	return &tsq{ts: svc}
}

func (q tsq) Aggregate(period, aggregates string) TempServiceQuery {
	supportedAggregates := map[string]aggrFunc{
		aggrMethodAverage: average,
	}

	for _, aggrName := range strings.Split(aggregates, ",") {
		if aggrFn, ok := supportedAggregates[aggrName]; ok {
			q.aggregations = append(q.aggregations, aggrFn)
		}
	}

	return q
}

func (q tsq) BetweenTimes(from, to time.Time) TempServiceQuery {
	q.from = from
	q.to = to
	return q
}

func (q tsq) Get() ([]domain.Temperature, error) {

	timeAt := q.from.Format(time.RFC3339)
	endTimeAt := q.to.Format(time.RFC3339)

	url := fmt.Sprintf(
		"%s/ngsi-ld/v1/entities?type=WeatherObserved&attrs=temperature&georel=near%%3BmaxDistance==2000&geometry=Point&coordinates=[17.3051555,62.3908926]&timerel=between&timeAt=%s&endTimeAt=%s",
		q.ts.contextBrokerURL, timeAt, endTimeAt,
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
	fmt.Printf("received response: %s\n", string(b))

	err = json.Unmarshal(b, &wos)
	if err != nil {
		return nil, err
	}

	temps := []domain.Temperature{}

	for _, wo := range wos {
		t := domain.Temperature{
			Id:    wo.RefDevice.Object,
			Value: wo.Temperature.Value,
			When:  wo.DateObserved.Value.Value,
		}
		temps = append(temps, t)
	}

	return temps, nil
}
