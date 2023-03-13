// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package waterquality

import (
	"sync"
	"time"
)

// Ensure, that WaterQualityServiceMock does implement WaterQualityService.
// If this is not the case, regenerate this file with moq.
var _ WaterQualityService = &WaterQualityServiceMock{}

// WaterQualityServiceMock is a mock implementation of WaterQualityService.
//
// 	func TestSomethingThatUsesWaterQualityService(t *testing.T) {
//
// 		// make and configure a mocked WaterQualityService
// 		mockedWaterQualityService := &WaterQualityServiceMock{
// 			BetweenTimesFunc: func(from time.Time, to time.Time)  {
// 				panic("mock out the BetweenTimes method")
// 			},
// 			BrokerFunc: func() string {
// 				panic("mock out the Broker method")
// 			},
// 			DistanceFunc: func(distance int)  {
// 				panic("mock out the Distance method")
// 			},
// 			GetAllFunc: func() []byte {
// 				panic("mock out the GetAll method")
// 			},
// 			GetAllNearPointFunc: func(pt Point, distance int) (*[]WaterQualityTemporal, error) {
// 				panic("mock out the GetAllNearPoint method")
// 			},
// 			GetByIDFunc: func(id string) (*WaterQualityTemporal, error) {
// 				panic("mock out the GetByID method")
// 			},
// 			LocationFunc: func(latitude float64, longitude float64)  {
// 				panic("mock out the Location method")
// 			},
// 			ShutdownFunc: func()  {
// 				panic("mock out the Shutdown method")
// 			},
// 			StartFunc: func()  {
// 				panic("mock out the Start method")
// 			},
// 			TenantFunc: func() string {
// 				panic("mock out the Tenant method")
// 			},
// 		}
//
// 		// use mockedWaterQualityService in code that requires WaterQualityService
// 		// and then make assertions.
//
// 	}
type WaterQualityServiceMock struct {
	// BetweenTimesFunc mocks the BetweenTimes method.
	BetweenTimesFunc func(from time.Time, to time.Time)

	// BrokerFunc mocks the Broker method.
	BrokerFunc func() string

	// DistanceFunc mocks the Distance method.
	DistanceFunc func(distance int)

	// GetAllFunc mocks the GetAll method.
	GetAllFunc func() []byte

	// GetAllNearPointFunc mocks the GetAllNearPoint method.
	GetAllNearPointFunc func(pt Point, distance int) (*[]WaterQualityTemporal, error)

	// GetByIDFunc mocks the GetByID method.
	GetByIDFunc func(id string) (*WaterQualityTemporal, error)

	// LocationFunc mocks the Location method.
	LocationFunc func(latitude float64, longitude float64)

	// ShutdownFunc mocks the Shutdown method.
	ShutdownFunc func()

	// StartFunc mocks the Start method.
	StartFunc func()

	// TenantFunc mocks the Tenant method.
	TenantFunc func() string

	// calls tracks calls to the methods.
	calls struct {
		// BetweenTimes holds details about calls to the BetweenTimes method.
		BetweenTimes []struct {
			// From is the from argument value.
			From time.Time
			// To is the to argument value.
			To time.Time
		}
		// Broker holds details about calls to the Broker method.
		Broker []struct {
		}
		// Distance holds details about calls to the Distance method.
		Distance []struct {
			// Distance is the distance argument value.
			Distance int
		}
		// GetAll holds details about calls to the GetAll method.
		GetAll []struct {
		}
		// GetAllNearPoint holds details about calls to the GetAllNearPoint method.
		GetAllNearPoint []struct {
			// Pt is the pt argument value.
			Pt Point
			// Distance is the distance argument value.
			Distance int
		}
		// GetByID holds details about calls to the GetByID method.
		GetByID []struct {
			// ID is the id argument value.
			ID string
		}
		// Location holds details about calls to the Location method.
		Location []struct {
			// Latitude is the latitude argument value.
			Latitude float64
			// Longitude is the longitude argument value.
			Longitude float64
		}
		// Shutdown holds details about calls to the Shutdown method.
		Shutdown []struct {
		}
		// Start holds details about calls to the Start method.
		Start []struct {
		}
		// Tenant holds details about calls to the Tenant method.
		Tenant []struct {
		}
	}
	lockBetweenTimes    sync.RWMutex
	lockBroker          sync.RWMutex
	lockDistance        sync.RWMutex
	lockGetAll          sync.RWMutex
	lockGetAllNearPoint sync.RWMutex
	lockGetByID         sync.RWMutex
	lockLocation        sync.RWMutex
	lockShutdown        sync.RWMutex
	lockStart           sync.RWMutex
	lockTenant          sync.RWMutex
}

// BetweenTimes calls BetweenTimesFunc.
func (mock *WaterQualityServiceMock) BetweenTimes(from time.Time, to time.Time) {
	if mock.BetweenTimesFunc == nil {
		panic("WaterQualityServiceMock.BetweenTimesFunc: method is nil but WaterQualityService.BetweenTimes was just called")
	}
	callInfo := struct {
		From time.Time
		To   time.Time
	}{
		From: from,
		To:   to,
	}
	mock.lockBetweenTimes.Lock()
	mock.calls.BetweenTimes = append(mock.calls.BetweenTimes, callInfo)
	mock.lockBetweenTimes.Unlock()
	mock.BetweenTimesFunc(from, to)
}

// BetweenTimesCalls gets all the calls that were made to BetweenTimes.
// Check the length with:
//     len(mockedWaterQualityService.BetweenTimesCalls())
func (mock *WaterQualityServiceMock) BetweenTimesCalls() []struct {
	From time.Time
	To   time.Time
} {
	var calls []struct {
		From time.Time
		To   time.Time
	}
	mock.lockBetweenTimes.RLock()
	calls = mock.calls.BetweenTimes
	mock.lockBetweenTimes.RUnlock()
	return calls
}

// Broker calls BrokerFunc.
func (mock *WaterQualityServiceMock) Broker() string {
	if mock.BrokerFunc == nil {
		panic("WaterQualityServiceMock.BrokerFunc: method is nil but WaterQualityService.Broker was just called")
	}
	callInfo := struct {
	}{}
	mock.lockBroker.Lock()
	mock.calls.Broker = append(mock.calls.Broker, callInfo)
	mock.lockBroker.Unlock()
	return mock.BrokerFunc()
}

// BrokerCalls gets all the calls that were made to Broker.
// Check the length with:
//     len(mockedWaterQualityService.BrokerCalls())
func (mock *WaterQualityServiceMock) BrokerCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockBroker.RLock()
	calls = mock.calls.Broker
	mock.lockBroker.RUnlock()
	return calls
}

// Distance calls DistanceFunc.
func (mock *WaterQualityServiceMock) Distance(distance int) {
	if mock.DistanceFunc == nil {
		panic("WaterQualityServiceMock.DistanceFunc: method is nil but WaterQualityService.Distance was just called")
	}
	callInfo := struct {
		Distance int
	}{
		Distance: distance,
	}
	mock.lockDistance.Lock()
	mock.calls.Distance = append(mock.calls.Distance, callInfo)
	mock.lockDistance.Unlock()
	mock.DistanceFunc(distance)
}

// DistanceCalls gets all the calls that were made to Distance.
// Check the length with:
//     len(mockedWaterQualityService.DistanceCalls())
func (mock *WaterQualityServiceMock) DistanceCalls() []struct {
	Distance int
} {
	var calls []struct {
		Distance int
	}
	mock.lockDistance.RLock()
	calls = mock.calls.Distance
	mock.lockDistance.RUnlock()
	return calls
}

// GetAll calls GetAllFunc.
func (mock *WaterQualityServiceMock) GetAll() []byte {
	if mock.GetAllFunc == nil {
		panic("WaterQualityServiceMock.GetAllFunc: method is nil but WaterQualityService.GetAll was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetAll.Lock()
	mock.calls.GetAll = append(mock.calls.GetAll, callInfo)
	mock.lockGetAll.Unlock()
	return mock.GetAllFunc()
}

// GetAllCalls gets all the calls that were made to GetAll.
// Check the length with:
//     len(mockedWaterQualityService.GetAllCalls())
func (mock *WaterQualityServiceMock) GetAllCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetAll.RLock()
	calls = mock.calls.GetAll
	mock.lockGetAll.RUnlock()
	return calls
}

// GetAllNearPoint calls GetAllNearPointFunc.
func (mock *WaterQualityServiceMock) GetAllNearPoint(pt Point, distance int) (*[]WaterQualityTemporal, error) {
	if mock.GetAllNearPointFunc == nil {
		panic("WaterQualityServiceMock.GetAllNearPointFunc: method is nil but WaterQualityService.GetAllNearPoint was just called")
	}
	callInfo := struct {
		Pt       Point
		Distance int
	}{
		Pt:       pt,
		Distance: distance,
	}
	mock.lockGetAllNearPoint.Lock()
	mock.calls.GetAllNearPoint = append(mock.calls.GetAllNearPoint, callInfo)
	mock.lockGetAllNearPoint.Unlock()
	return mock.GetAllNearPointFunc(pt, distance)
}

// GetAllNearPointCalls gets all the calls that were made to GetAllNearPoint.
// Check the length with:
//     len(mockedWaterQualityService.GetAllNearPointCalls())
func (mock *WaterQualityServiceMock) GetAllNearPointCalls() []struct {
	Pt       Point
	Distance int
} {
	var calls []struct {
		Pt       Point
		Distance int
	}
	mock.lockGetAllNearPoint.RLock()
	calls = mock.calls.GetAllNearPoint
	mock.lockGetAllNearPoint.RUnlock()
	return calls
}

// GetByID calls GetByIDFunc.
func (mock *WaterQualityServiceMock) GetByID(id string) (*WaterQualityTemporal, error) {
	if mock.GetByIDFunc == nil {
		panic("WaterQualityServiceMock.GetByIDFunc: method is nil but WaterQualityService.GetByID was just called")
	}
	callInfo := struct {
		ID string
	}{
		ID: id,
	}
	mock.lockGetByID.Lock()
	mock.calls.GetByID = append(mock.calls.GetByID, callInfo)
	mock.lockGetByID.Unlock()
	return mock.GetByIDFunc(id)
}

// GetByIDCalls gets all the calls that were made to GetByID.
// Check the length with:
//     len(mockedWaterQualityService.GetByIDCalls())
func (mock *WaterQualityServiceMock) GetByIDCalls() []struct {
	ID string
} {
	var calls []struct {
		ID string
	}
	mock.lockGetByID.RLock()
	calls = mock.calls.GetByID
	mock.lockGetByID.RUnlock()
	return calls
}

// Location calls LocationFunc.
func (mock *WaterQualityServiceMock) Location(latitude float64, longitude float64) {
	if mock.LocationFunc == nil {
		panic("WaterQualityServiceMock.LocationFunc: method is nil but WaterQualityService.Location was just called")
	}
	callInfo := struct {
		Latitude  float64
		Longitude float64
	}{
		Latitude:  latitude,
		Longitude: longitude,
	}
	mock.lockLocation.Lock()
	mock.calls.Location = append(mock.calls.Location, callInfo)
	mock.lockLocation.Unlock()
	mock.LocationFunc(latitude, longitude)
}

// LocationCalls gets all the calls that were made to Location.
// Check the length with:
//     len(mockedWaterQualityService.LocationCalls())
func (mock *WaterQualityServiceMock) LocationCalls() []struct {
	Latitude  float64
	Longitude float64
} {
	var calls []struct {
		Latitude  float64
		Longitude float64
	}
	mock.lockLocation.RLock()
	calls = mock.calls.Location
	mock.lockLocation.RUnlock()
	return calls
}

// Shutdown calls ShutdownFunc.
func (mock *WaterQualityServiceMock) Shutdown() {
	if mock.ShutdownFunc == nil {
		panic("WaterQualityServiceMock.ShutdownFunc: method is nil but WaterQualityService.Shutdown was just called")
	}
	callInfo := struct {
	}{}
	mock.lockShutdown.Lock()
	mock.calls.Shutdown = append(mock.calls.Shutdown, callInfo)
	mock.lockShutdown.Unlock()
	mock.ShutdownFunc()
}

// ShutdownCalls gets all the calls that were made to Shutdown.
// Check the length with:
//     len(mockedWaterQualityService.ShutdownCalls())
func (mock *WaterQualityServiceMock) ShutdownCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockShutdown.RLock()
	calls = mock.calls.Shutdown
	mock.lockShutdown.RUnlock()
	return calls
}

// Start calls StartFunc.
func (mock *WaterQualityServiceMock) Start() {
	if mock.StartFunc == nil {
		panic("WaterQualityServiceMock.StartFunc: method is nil but WaterQualityService.Start was just called")
	}
	callInfo := struct {
	}{}
	mock.lockStart.Lock()
	mock.calls.Start = append(mock.calls.Start, callInfo)
	mock.lockStart.Unlock()
	mock.StartFunc()
}

// StartCalls gets all the calls that were made to Start.
// Check the length with:
//     len(mockedWaterQualityService.StartCalls())
func (mock *WaterQualityServiceMock) StartCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockStart.RLock()
	calls = mock.calls.Start
	mock.lockStart.RUnlock()
	return calls
}

// Tenant calls TenantFunc.
func (mock *WaterQualityServiceMock) Tenant() string {
	if mock.TenantFunc == nil {
		panic("WaterQualityServiceMock.TenantFunc: method is nil but WaterQualityService.Tenant was just called")
	}
	callInfo := struct {
	}{}
	mock.lockTenant.Lock()
	mock.calls.Tenant = append(mock.calls.Tenant, callInfo)
	mock.lockTenant.Unlock()
	return mock.TenantFunc()
}

// TenantCalls gets all the calls that were made to Tenant.
// Check the length with:
//     len(mockedWaterQualityService.TenantCalls())
func (mock *WaterQualityServiceMock) TenantCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockTenant.RLock()
	calls = mock.calls.Tenant
	mock.lockTenant.RUnlock()
	return calls
}
