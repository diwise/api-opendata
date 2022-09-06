package citywork

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatGetByIDReturnsErrorOnCityworkNotFound(t *testing.T) {
	is := is.New(t)
	server := setupMockServiceThatReturns(200, "")
	log := zerolog.Logger{}

	cityworksSvc := NewCityworksService(context.Background(), log, server.URL, "default")
	defer cityworksSvc.Shutdown()

	cityworkBytes, err := cityworksSvc.GetByID("cityworkID")
	is.True(err != nil)
	is.Equal(cityworkBytes, []byte{})
}

func setupMockServiceThatReturns(responseCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseCode)
		w.Header().Add("Content-Type", "application/ld+json")
		if body != "" {
			w.Write([]byte(body))
		}
	}))
}
