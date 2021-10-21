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
