package citywork

import (
	"context"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestThatRefreshReturnsErrorOnNoValidHostNotFound(t *testing.T) {
	is := is.New(t)
	log := zerolog.Logger{}

	cwSvc := NewCityworksService(context.Background(), log, "http://lolcat:1234", "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	err := svc.refresh()
	is.True(err != nil) //should return err due to invalid host
}

func TestXxx(t *testing.T) {
	is := is.New(t)
	log := zerolog.Logger{}

	cwSvc := NewCityworksService(context.Background(), log, "http://lolcat:1234", "default")

	svc, ok := cwSvc.(*cityworksSvc)
	is.True(ok)

	svc.cityworks = []byte{}
	//test that refresh passes when citywork list is not empty
}
