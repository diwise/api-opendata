package datasets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/application/services"
	"github.com/diwise/api-opendata/internal/pkg/infrastructure/logging"
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

func NewRetrieveTemperaturesHandler(log logging.Logger, svc services.TempService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		response := &TempResponse{
			Items: []TempResponseItem{},
		}

		tempsFromCtxBroker, _ := svc.Get(time.Now().UTC().Add(-1*24*time.Hour), time.Now().UTC())

		tempRespItemMap := make(map[string]TempResponseItem)
		sumOfTemperatures := make(map[string]float64)

		for _, t := range tempsFromCtxBroker {

			tempRespItem, exist := tempRespItemMap[t.Id]

			if exist {
				tempRespItem.Values = append(tempRespItem.Values, TempResponseValue{
					Value: fmt.Sprintf("%.2f", t.Value),
					When:  t.When,
				})

				sumOfTemperatures[t.Id] = sumOfTemperatures[t.Id] + t.Value

				tempRespItemMap[t.Id] = tempRespItem

			} else {
				newTempRespItem := TempResponseItem{
					ID: t.Id,
					Values: []TempResponseValue{
						{
							Value: fmt.Sprintf("%.2f", t.Value),
							When:  t.When,
						}},
				}

				sumOfTemperatures[t.Id] = t.Value

				tempRespItemMap[t.Id] = newTempRespItem
			}

		}

		for _, v := range tempRespItemMap {
			v.Average = fmt.Sprintf("%.2f", sumOfTemperatures[v.ID]/float64(len(v.Values)))
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
