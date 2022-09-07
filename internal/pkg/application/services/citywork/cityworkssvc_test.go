package citywork

import (
	"context"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatGetByIDReturnsErrorOnCityworkNotFound(t *testing.T) {
	is := is.New(t)
	log := zerolog.Logger{}

	cityworksSvc := NewCityworksService(context.Background(), log, "http://lolcat:1234", "default")

	_, err := cityworksSvc.GetByID("cityworkID")
	is.True(err != nil)
}
