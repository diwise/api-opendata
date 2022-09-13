// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package roadaccidents

import (
	"sync"
)

// Ensure, that RoadAccidentServiceMock does implement RoadAccidentService.
// If this is not the case, regenerate this file with moq.
var _ RoadAccidentService = &RoadAccidentServiceMock{}

// RoadAccidentServiceMock is a mock implementation of RoadAccidentService.
//
// 	func TestSomethingThatUsesRoadAccidentService(t *testing.T) {
//
// 		// make and configure a mocked RoadAccidentService
// 		mockedRoadAccidentService := &RoadAccidentServiceMock{
// 			BrokerFunc: func() string {
// 				panic("mock out the Broker method")
// 			},
// 			GetAllFunc: func() []byte {
// 				panic("mock out the GetAll method")
// 			},
// 			GetByIDFunc: func(id string) ([]byte, error) {
// 				panic("mock out the GetByID method")
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
// 		// use mockedRoadAccidentService in code that requires RoadAccidentService
// 		// and then make assertions.
//
// 	}
type RoadAccidentServiceMock struct {
	// BrokerFunc mocks the Broker method.
	BrokerFunc func() string

	// GetAllFunc mocks the GetAll method.
	GetAllFunc func() []byte

	// GetByIDFunc mocks the GetByID method.
	GetByIDFunc func(id string) ([]byte, error)

	// ShutdownFunc mocks the Shutdown method.
	ShutdownFunc func()

	// StartFunc mocks the Start method.
	StartFunc func()

	// TenantFunc mocks the Tenant method.
	TenantFunc func() string

	// calls tracks calls to the methods.
	calls struct {
		// Broker holds details about calls to the Broker method.
		Broker []struct {
		}
		// GetAll holds details about calls to the GetAll method.
		GetAll []struct {
		}
		// GetByID holds details about calls to the GetByID method.
		GetByID []struct {
			// ID is the id argument value.
			ID string
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
	lockBroker   sync.RWMutex
	lockGetAll   sync.RWMutex
	lockGetByID  sync.RWMutex
	lockShutdown sync.RWMutex
	lockStart    sync.RWMutex
	lockTenant   sync.RWMutex
}

// Broker calls BrokerFunc.
func (mock *RoadAccidentServiceMock) Broker() string {
	if mock.BrokerFunc == nil {
		panic("RoadAccidentServiceMock.BrokerFunc: method is nil but RoadAccidentService.Broker was just called")
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
//     len(mockedRoadAccidentService.BrokerCalls())
func (mock *RoadAccidentServiceMock) BrokerCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockBroker.RLock()
	calls = mock.calls.Broker
	mock.lockBroker.RUnlock()
	return calls
}

// GetAll calls GetAllFunc.
func (mock *RoadAccidentServiceMock) GetAll() []byte {
	if mock.GetAllFunc == nil {
		panic("RoadAccidentServiceMock.GetAllFunc: method is nil but RoadAccidentService.GetAll was just called")
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
//     len(mockedRoadAccidentService.GetAllCalls())
func (mock *RoadAccidentServiceMock) GetAllCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetAll.RLock()
	calls = mock.calls.GetAll
	mock.lockGetAll.RUnlock()
	return calls
}

// GetByID calls GetByIDFunc.
func (mock *RoadAccidentServiceMock) GetByID(id string) ([]byte, error) {
	if mock.GetByIDFunc == nil {
		panic("RoadAccidentServiceMock.GetByIDFunc: method is nil but RoadAccidentService.GetByID was just called")
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
//     len(mockedRoadAccidentService.GetByIDCalls())
func (mock *RoadAccidentServiceMock) GetByIDCalls() []struct {
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

// Shutdown calls ShutdownFunc.
func (mock *RoadAccidentServiceMock) Shutdown() {
	if mock.ShutdownFunc == nil {
		panic("RoadAccidentServiceMock.ShutdownFunc: method is nil but RoadAccidentService.Shutdown was just called")
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
//     len(mockedRoadAccidentService.ShutdownCalls())
func (mock *RoadAccidentServiceMock) ShutdownCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockShutdown.RLock()
	calls = mock.calls.Shutdown
	mock.lockShutdown.RUnlock()
	return calls
}

// Start calls StartFunc.
func (mock *RoadAccidentServiceMock) Start() {
	if mock.StartFunc == nil {
		panic("RoadAccidentServiceMock.StartFunc: method is nil but RoadAccidentService.Start was just called")
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
//     len(mockedRoadAccidentService.StartCalls())
func (mock *RoadAccidentServiceMock) StartCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockStart.RLock()
	calls = mock.calls.Start
	mock.lockStart.RUnlock()
	return calls
}

// Tenant calls TenantFunc.
func (mock *RoadAccidentServiceMock) Tenant() string {
	if mock.TenantFunc == nil {
		panic("RoadAccidentServiceMock.TenantFunc: method is nil but RoadAccidentService.Tenant was just called")
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
//     len(mockedRoadAccidentService.TenantCalls())
func (mock *RoadAccidentServiceMock) TenantCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockTenant.RLock()
	calls = mock.calls.Tenant
	mock.lockTenant.RUnlock()
	return calls
}
