package weather

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	testutils "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/expects"
	"github.com/diwise/service-chassis/pkg/test/http/response"

	"github.com/matryer/is"
)

var Expects = testutils.Expects
var Returns = testutils.Returns
var anyInput = expects.AnyInput

func TestCalc(t *testing.T) {
	is := is.New(t)

	v1 := 7.0
	v2 := 3.0

	w1 := time.Now().UTC().Add(-1 * time.Hour)
	w2 := time.Now().UTC()

	w := domain.Weather{
		Temperature: domain.Temperature{
			Values: &[]domain.Temperature{
				{
					Value: &v1,
					When:  &w1,
				},
				{
					Value: &v2,
					When:  &w2,
				},
			},
		},
	}
	
	calc(&w)

	is.Equal(5.0, *w.Temperature.Average)
	is.Equal(7.0, *w.Temperature.Max)
	is.Equal(3.0, *w.Temperature.Min)
	is.Equal(w1, *w.Temperature.From)
	is.Equal(w2, *w.Temperature.To)
}

func testSetup(t *testing.T, statusCode int, responseBody string) (*is.I, context.Context, testutils.MockService, WeatherService) {
	is := is.New(t)
	ctx := context.Background()

	ms := testutils.NewMockServiceThat(
		Expects(is, anyInput()),
		Returns(
			response.Code(statusCode),
			response.ContentType("application/ld+json"),
			response.Body([]byte(responseBody)),
		),
	)

	weatherService := NewWeatherService(ctx, ms.URL(), "")

	return is, ctx, ms, weatherService
}
