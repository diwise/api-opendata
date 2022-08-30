package roadworks

import (
	"context"

	"github.com/rs/zerolog"
)

type RoadWorkService interface {
	Broker() string
	Tenant() string

	GetAll() []byte

	Start()
	Shutdown()
}

func NewRoadWorksService(ctx context.Context, logger zerolog.Logger, contextBrokerUrl, tenant string) RoadWorkService {
	svc := &roadWorksSvc{
		ctx: ctx,
		log: logger,

		tenant:           tenant,
		contextBrokerURL: contextBrokerUrl,

		keepRunning: true,
	}

	return svc
}

type roadWorksSvc struct {
	contextBrokerURL string
	tenant           string

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *roadWorksSvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *roadWorksSvc) Tenant() string {
	return svc.tenant
}

func (svc *roadWorksSvc) GetAll() []byte {
	return []byte{}
}

func (svc *roadWorksSvc) Start() {
	svc.log.Info().Msg("starting road-works service")
	go svc.run()
}

func (svc *roadWorksSvc) Shutdown() {

}

func (svc *roadWorksSvc) run() {

}

func (svc *roadWorksSvc) refresh() error {
	return nil
}
