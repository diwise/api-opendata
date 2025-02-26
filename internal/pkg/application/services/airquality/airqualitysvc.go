package airquality

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diwise/api-opendata/internal/pkg/domain"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/context-broker/pkg/ngsild/geojson"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("api-opendata/svcs/airquality")

const (
	DefaultBrokerTenant string = "default"
)

type AirQualityService interface {
	Refresh(ctx context.Context) (int, error)
	Shutdown(ctx context.Context)
	Start(ctx context.Context)

	//Broker() string
	Tenant() string

	GetAll(ctx context.Context) []domain.AirQuality
	GetByID(ctx context.Context, id string) (*domain.AirQualityDetails, error)
}

var ErrNoSuchAirQuality error = errors.New("no such air quality")

func NewAirQualityService(ctx context.Context, cbClient client.ContextBrokerClient, ctxBrokerTenant string) AirQualityService {
	return &aqsvc{
		cbClient: cbClient,
		tenant:   ctxBrokerTenant,

		airQualities:   []domain.AirQuality{},
		airQualityByID: map[string]domain.AirQualityDetails{},

		queue:       make(chan func()),
		keepRunning: &atomic.Bool{},
	}
}

type aqsvc struct {
	cbClient client.ContextBrokerClient
	tenant   string

	airQualities   []domain.AirQuality
	airQualityByID map[string]domain.AirQualityDetails

	queue chan func()

	keepRunning *atomic.Bool
	wg          sync.WaitGroup
}

/*func (svc *aqsvc) Broker() string {
	return svc.cbClient
}*/

func (svc *aqsvc) Tenant() string {
	return svc.tenant
}

func (svc *aqsvc) GetAll(ctx context.Context) []domain.AirQuality {
	result := make(chan []domain.AirQuality)

	svc.queue <- func() {
		result <- svc.airQualities
	}

	return <-result
}

func (svc *aqsvc) GetByID(ctx context.Context, id string) (*domain.AirQualityDetails, error) {
	result := make(chan domain.AirQualityDetails)
	err := make(chan error)

	svc.queue <- func() {
		body, ok := svc.airQualityByID[id]
		if !ok {
			err <- ErrNoSuchAirQuality
		} else {
			result <- body
		}
	}

	select {
	case r := <-result:
		return &r, nil
	case e := <-err:
		return nil, e
	}
}

func (svc *aqsvc) Refresh(ctx context.Context) (int, error) {
	refreshDone := make(chan int)
	refreshFailed := make(chan error)

	svc.queue <- func() {
		count, err := svc.refresh(ctx)
		if err != nil {
			refreshFailed <- err
		} else {
			refreshDone <- count
		}
	}

	select {
	case c := <-refreshDone:
		return c, nil
	case e := <-refreshFailed:
		return 0, e
	}
}

func (svc *aqsvc) Start(ctx context.Context) {
	go svc.run(ctx)
}

func (svc *aqsvc) Shutdown(ctx context.Context) {
	svc.queue <- func() {
		svc.keepRunning.Store(false)
	}

	svc.wg.Wait()
}

func (svc *aqsvc) run(ctx context.Context) {
	svc.wg.Add(1)
	defer svc.wg.Done()

	logger := logging.GetFromContext(ctx)
	logger.Info("starting up air quality service")

	// use atomic swap to avoid startup races
	alreadyStarted := svc.keepRunning.Swap(true)
	if alreadyStarted {
		logger.Error("attempt to start the air quality service multiple times")
		return
	}

	const RefreshIntervalOnFail time.Duration = 5 * time.Second
	const RefreshIntervalOnSuccess time.Duration = 5 * time.Minute

	var refreshTimer *time.Timer
	count, err := svc.refresh(ctx)

	if err != nil {
		logger.Error("failed to refresh air qualities", "err", err.Error())
		refreshTimer = time.NewTimer(RefreshIntervalOnFail)
	} else {
		logger.Info("refreshed air qualities", "count", count)
		refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
	}

	for svc.keepRunning.Load() {
		select {
		case fn := <-svc.queue:
			fn()
		case <-refreshTimer.C:
			count, err := svc.refresh(ctx)
			if err != nil {
				logger.Error("failed to refresh air qualities", "err", err.Error())
				refreshTimer = time.NewTimer(RefreshIntervalOnFail)
			} else {
				logger.Info("refreshed air qualities", "count", count)
				refreshTimer = time.NewTimer(RefreshIntervalOnSuccess)
			}
		}
	}

	logger.Info("air quality service exiting")
}

func (svc *aqsvc) refresh(ctx context.Context) (count int, err error) {
	ctx, span := tracer.Start(ctx, "refresh-air-quality")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

	logger.Info("refreshing air quality info")

	headers := map[string][]string{
		"Accept": {"application/ld+json"},
		"Link":   {entities.LinkHeader},
	}

	params := url.Values{}
	params.Add("type", "AirQualityObserved")
	params.Add("count", "true")

	reqUrl := fmt.Sprintf("/ngsi-ld/v1/entities?%s", params.Encode())

	airqualities := []domain.AirQuality{}
	res, err := svc.cbClient.QueryEntities(ctx, nil, nil, reqUrl, headers)
	if err != nil {
		logger.Error("failed to retrieve air qualities from context broker", "err", err.Error())
		return
	}

	for {
		airquality := <-res.Found
		if airquality == nil {
			break
		}

		airqualities = append(airqualities, toAirQuality(airquality))
	}

	svc.airQualities = airqualities

	err = svc.getDetails(ctx, svc.cbClient, headers)
	if err != nil {
		return len(svc.airQualities), err
	}

	return len(svc.airQualities), nil
}

type Result struct {
	Found      []domain.AirQuality
	TotalCount int64
	Count      int
	Offset     int
	Limit      int
	Partial    bool
}

const (
	LocationPropertyName                  string = "location"
	AtmosphericPressurePropertyName       string = "atmosphericPressure"
	TemperaturePropertyName               string = "temperature"
	RelativeHumidityPropertyName          string = "relativeHumidity"
	ParticleCountPropertyName             string = "particleCount"
	PM1PropertyName                       string = "PM1"
	PM4PropertyName                       string = "PM4"
	PM10PropertyName                      string = "PM10"
	PM25PropertyName                      string = "PM25"
	TotalSuspendedParticulatePropertyName string = "totalSuspendedParticulate"
	CO2PropertyName                       string = "CO2"
	NOPropertyName                        string = "NO"
	NO2PropertyName                       string = "NO2"
	NOxPropertyName                       string = "NOx"
	VoltagePropertyName                   string = "voltage"
	WindDirectionPropertyName             string = "windDirection"
	WindSpeedPropertyName                 string = "windSpeed"
	DateObservedPropertyName              string = "dateObserved"
)

func toAirQuality(n types.Entity) domain.AirQuality {
	airquality := domain.AirQuality{}
	airquality.ID = n.ID()

	n.ForEachAttribute(func(attributeType, attributeName string, contents any) {
		switch attributeName {
		case DateObservedPropertyName:
			p := contents.(*properties.DateTimeProperty)
			timestamp, _ := time.Parse(time.RFC3339, p.Val.Value)
			airquality.DateObserved = *domain.NewDateTime(timestamp.UTC().Format(time.RFC3339))
		case LocationPropertyName:
			p := contents.(*geojson.GeoJSONProperty)
			point := p.GetAsPoint()
			airquality.Location.Coordinates = point.Coordinates[:]
			airquality.Location.Type = p.GeoPropertyType()
		case AtmosphericPressurePropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.AtmosphericPressure = &p.Val
		case TemperaturePropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.Temperature = &p.Val
		case RelativeHumidityPropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.RelativeHumidity = &p.Val
		case ParticleCountPropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.ParticleCount = &p.Val
		case PM1PropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.PM1 = &p.Val
		case PM4PropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.PM4 = &p.Val
		case PM10PropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.PM10 = &p.Val
		case PM25PropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.PM25 = &p.Val
		case TotalSuspendedParticulatePropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.TotalSuspendedParticulate = &p.Val
		case CO2PropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.CO2 = &p.Val
		case NOPropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.NO = &p.Val
		case NO2PropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.NO2 = &p.Val
		case NOxPropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.NOx = &p.Val
		case VoltagePropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.Voltage = &p.Val
		case WindDirectionPropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.WindDirection = &p.Val
		case WindSpeedPropertyName:
			p := contents.(*properties.NumberProperty)
			airquality.WindSpeed = &p.Val
		}
	})

	return airquality
}

func (svc *aqsvc) getDetails(ctx context.Context, c client.ContextBrokerClient, headers map[string][]string) error {
	logger := logging.GetFromContext(ctx)

	for _, aqo := range svc.airQualities {
		details := domain.AirQualityDetails{}
		pollutants := []domain.Pollutant{}

		details.ID = aqo.ID
		details.DateObserved = aqo.DateObserved
		details.Location = aqo.Location

		t, err := c.RetrieveTemporalEvolutionOfEntity(ctx, aqo.ID, headers, client.Between(time.Now().Add(-24*time.Hour), time.Now()))
		if err != nil || t.Found == nil {
			logger.Error("failed to retrieve temporal evolution of air qualities", "err", err.Error())
			return err
		}

		if len(t.Found.Property(AtmosphericPressurePropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("AtmosphericPressure", t.Found.Property(AtmosphericPressurePropertyName)))
		}
		if len(t.Found.Property(TemperaturePropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("Temperature", t.Found.Property(TemperaturePropertyName)))
		}
		if len(t.Found.Property(RelativeHumidityPropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("RelativeHumidity", t.Found.Property(RelativeHumidityPropertyName)))
		}
		if len(t.Found.Property(ParticleCountPropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("ParticleCount", t.Found.Property(ParticleCountPropertyName)))
		}
		if len(t.Found.Property(PM1PropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("PM1", t.Found.Property(PM1PropertyName)))
		}
		if len(t.Found.Property(PM4PropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("PM4", t.Found.Property(PM4PropertyName)))
		}
		if len(t.Found.Property(PM10PropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("PM10", t.Found.Property(PM10PropertyName)))
		}
		if len(t.Found.Property(PM25PropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("PM25", t.Found.Property(PM25PropertyName)))
		}
		if len(t.Found.Property(TotalSuspendedParticulatePropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("TotalSuspendedParticulate", t.Found.Property(TotalSuspendedParticulatePropertyName)))
		}
		if len(t.Found.Property(CO2PropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("CO2", t.Found.Property(CO2PropertyName)))
		}
		if len(t.Found.Property(NOPropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("NO", t.Found.Property(NOPropertyName)))
		}
		if len(t.Found.Property(NO2PropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("NO2", t.Found.Property(NO2PropertyName)))
		}
		if len(t.Found.Property(NOxPropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("NOx", t.Found.Property(NOxPropertyName)))
		}
		if len(t.Found.Property(WindDirectionPropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("WindDirection", t.Found.Property(WindDirectionPropertyName)))
		}
		if len(t.Found.Property(WindSpeedPropertyName)) > 0 {
			pollutants = append(pollutants, addPollutant("WindSpeed", t.Found.Property(WindSpeedPropertyName)))
		}

		details.Pollutants = pollutants

		svc.airQualityByID[aqo.ID] = details
	}

	return nil
}

func addPollutant(name string, temporal []types.TemporalProperty) domain.Pollutant {
	aqi := domain.Pollutant{
		Name: name,
	}

	if len(temporal) > 0 {
		for _, v := range temporal {
			aqi.Values = append(aqi.Values, domain.Value{
				Value:      v.Value().(float64),
				ObservedAt: v.ObservedAt(),
			})
		}
		return aqi
	}

	return aqi
}
