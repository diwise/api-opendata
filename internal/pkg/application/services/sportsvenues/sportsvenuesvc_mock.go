// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package sportsvenues

import (
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"sync"
)

// Ensure, that SportsVenueServiceMock does implement SportsVenueService.
// If this is not the case, regenerate this file with moq.
var _ SportsVenueService = &SportsVenueServiceMock{}

// SportsVenueServiceMock is a mock implementation of SportsVenueService.
//
// 	func TestSomethingThatUsesSportsVenueService(t *testing.T) {
//
// 		// make and configure a mocked SportsVenueService
// 		mockedSportsVenueService := &SportsVenueServiceMock{
// 			BrokerFunc: func() string {
// 				panic("mock out the Broker method")
// 			},
// 			GetAllFunc: func() []domain.SportsVenue {
// 				panic("mock out the GetAll method")
// 			},
// 			GetByIDFunc: func(id string) (*domain.SportsVenue, error) {
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
// 		// use mockedSportsVenueService in code that requires SportsVenueService
// 		// and then make assertions.
//
// 	}
type SportsVenueServiceMock struct {
	// BrokerFunc mocks the Broker method.
	BrokerFunc func() string

	// GetAllFunc mocks the GetAll method.
	GetAllFunc func() []domain.SportsVenue

	// GetByIDFunc mocks the GetByID method.
	GetByIDFunc func(id string) (*domain.SportsVenue, error)

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
func (mock *SportsVenueServiceMock) Broker() string {
	if mock.BrokerFunc == nil {
		panic("SportsVenueServiceMock.BrokerFunc: method is nil but SportsVenueService.Broker was just called")
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
//     len(mockedSportsVenueService.BrokerCalls())
func (mock *SportsVenueServiceMock) BrokerCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockBroker.RLock()
	calls = mock.calls.Broker
	mock.lockBroker.RUnlock()
	return calls
}

// GetAll calls GetAllFunc.
func (mock *SportsVenueServiceMock) GetAll() []domain.SportsVenue {
	if mock.GetAllFunc == nil {
		panic("SportsVenueServiceMock.GetAllFunc: method is nil but SportsVenueService.GetAll was just called")
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
//     len(mockedSportsVenueService.GetAllCalls())
func (mock *SportsVenueServiceMock) GetAllCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetAll.RLock()
	calls = mock.calls.GetAll
	mock.lockGetAll.RUnlock()
	return calls
}

// GetByID calls GetByIDFunc.
func (mock *SportsVenueServiceMock) GetByID(id string) (*domain.SportsVenue, error) {
	if mock.GetByIDFunc == nil {
		panic("SportsVenueServiceMock.GetByIDFunc: method is nil but SportsVenueService.GetByID was just called")
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
//     len(mockedSportsVenueService.GetByIDCalls())
func (mock *SportsVenueServiceMock) GetByIDCalls() []struct {
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
func (mock *SportsVenueServiceMock) Shutdown() {
	if mock.ShutdownFunc == nil {
		panic("SportsVenueServiceMock.ShutdownFunc: method is nil but SportsVenueService.Shutdown was just called")
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
//     len(mockedSportsVenueService.ShutdownCalls())
func (mock *SportsVenueServiceMock) ShutdownCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockShutdown.RLock()
	calls = mock.calls.Shutdown
	mock.lockShutdown.RUnlock()
	return calls
}

// Start calls StartFunc.
func (mock *SportsVenueServiceMock) Start() {
	if mock.StartFunc == nil {
		panic("SportsVenueServiceMock.StartFunc: method is nil but SportsVenueService.Start was just called")
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
//     len(mockedSportsVenueService.StartCalls())
func (mock *SportsVenueServiceMock) StartCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockStart.RLock()
	calls = mock.calls.Start
	mock.lockStart.RUnlock()
	return calls
}

// Tenant calls TenantFunc.
func (mock *SportsVenueServiceMock) Tenant() string {
	if mock.TenantFunc == nil {
		panic("SportsVenueServiceMock.TenantFunc: method is nil but SportsVenueService.Tenant was just called")
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
//     len(mockedSportsVenueService.TenantCalls())
func (mock *SportsVenueServiceMock) TenantCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockTenant.RLock()
	calls = mock.calls.Tenant
	mock.lockTenant.RUnlock()
	return calls
}
