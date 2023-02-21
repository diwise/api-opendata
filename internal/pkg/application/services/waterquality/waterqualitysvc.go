package waterquality

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
)

type WaterQuality interface {
	Start()
	Shutdown()

	Tenant() string
	Broker() string

	GetAll() []byte
}

func NewWaterQualityService(ctx context.Context, log zerolog.Logger, url, tenant string, maxDistance int) WaterQuality {
	return &wqsvc{
		contextBrokerURL: url,
		tenant:           tenant,

		waterQualities:      []byte("[]"),
		waterQualityDetails: map[string][]byte{},
		maxDistance:         maxDistance,

		ctx: ctx,
		log: log,
	}
}

type wqsvc struct {
	contextBrokerURL string
	tenant           string

	wqoMutex            sync.Mutex
	waterQualities      []byte
	waterQualityDetails map[string][]byte
	maxDistance         int

	ctx context.Context
	log zerolog.Logger

	keepRunning bool
}

func (svc *wqsvc) Start() {
	svc.log.Info().Msg("starting water quality service")
	go svc.run()
}

func (svc *wqsvc) Shutdown() {
	svc.log.Info().Msg("shutting down water quality service")
	svc.keepRunning = false
}

func (svc *wqsvc) Broker() string {
	return svc.contextBrokerURL
}

func (svc *wqsvc) Tenant() string {
	return svc.tenant
}

func (svc *wqsvc) GetAll() []byte {
	svc.wqoMutex.Lock()
	defer svc.wqoMutex.Unlock()

	return svc.waterQualities
}

func (svc *wqsvc) run() {

}
