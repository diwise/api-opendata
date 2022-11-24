// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package sportsfields

import (
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"sync"
)

// Ensure, that SportsFieldServiceMock does implement SportsFieldService.
// If this is not the case, regenerate this file with moq.
var _ SportsFieldService = &SportsFieldServiceMock{}

// SportsFieldServiceMock is a mock implementation of SportsFieldService.
//
// 	func TestSomethingThatUsesSportsFieldService(t *testing.T) {
//
// 		// make and configure a mocked SportsFieldService
// 		mockedSportsFieldService := &SportsFieldServiceMock{
// 			BrokerFunc: func() string {
// 				panic("mock out the Broker method")
// 			},
// 			GetAllFunc: func(requiredCategories []string) []domain.SportsField {
// 				panic("mock out the GetAll method")
// 			},
// 			GetByIDFunc: func(id string) (*domain.SportsField, error) {
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
// 		// use mockedSportsFieldService in code that requires SportsFieldService
// 		// and then make assertions.
//
// 	}
type SportsFieldServiceMock struct {
	// BrokerFunc mocks the Broker method.
	BrokerFunc func() string

	// GetAllFunc mocks the GetAll method.
	GetAllFunc func(requiredCategories []string) []domain.SportsField

	// GetByIDFunc mocks the GetByID method.
	GetByIDFunc func(id string) (*domain.SportsField, error)

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
			// RequiredCategories is the requiredCategories argument value.
			RequiredCategories []string
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
func (mock *SportsFieldServiceMock) Broker() string {
	if mock.BrokerFunc == nil {
		panic("SportsFieldServiceMock.BrokerFunc: method is nil but SportsFieldService.Broker was just called")
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
//     len(mockedSportsFieldService.BrokerCalls())
func (mock *SportsFieldServiceMock) BrokerCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockBroker.RLock()
	calls = mock.calls.Broker
	mock.lockBroker.RUnlock()
	return calls
}

// GetAll calls GetAllFunc.
func (mock *SportsFieldServiceMock) GetAll(requiredCategories []string) []domain.SportsField {
	if mock.GetAllFunc == nil {
		panic("SportsFieldServiceMock.GetAllFunc: method is nil but SportsFieldService.GetAll was just called")
	}
	callInfo := struct {
		RequiredCategories []string
	}{
		RequiredCategories: requiredCategories,
	}
	mock.lockGetAll.Lock()
	mock.calls.GetAll = append(mock.calls.GetAll, callInfo)
	mock.lockGetAll.Unlock()
	return mock.GetAllFunc(requiredCategories)
}

// GetAllCalls gets all the calls that were made to GetAll.
// Check the length with:
//     len(mockedSportsFieldService.GetAllCalls())
func (mock *SportsFieldServiceMock) GetAllCalls() []struct {
	RequiredCategories []string
} {
	var calls []struct {
		RequiredCategories []string
	}
	mock.lockGetAll.RLock()
	calls = mock.calls.GetAll
	mock.lockGetAll.RUnlock()
	return calls
}

// GetByID calls GetByIDFunc.
func (mock *SportsFieldServiceMock) GetByID(id string) (*domain.SportsField, error) {
	if mock.GetByIDFunc == nil {
		panic("SportsFieldServiceMock.GetByIDFunc: method is nil but SportsFieldService.GetByID was just called")
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
//     len(mockedSportsFieldService.GetByIDCalls())
func (mock *SportsFieldServiceMock) GetByIDCalls() []struct {
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
func (mock *SportsFieldServiceMock) Shutdown() {
	if mock.ShutdownFunc == nil {
		panic("SportsFieldServiceMock.ShutdownFunc: method is nil but SportsFieldService.Shutdown was just called")
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
//     len(mockedSportsFieldService.ShutdownCalls())
func (mock *SportsFieldServiceMock) ShutdownCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockShutdown.RLock()
	calls = mock.calls.Shutdown
	mock.lockShutdown.RUnlock()
	return calls
}

// Start calls StartFunc.
func (mock *SportsFieldServiceMock) Start() {
	if mock.StartFunc == nil {
		panic("SportsFieldServiceMock.StartFunc: method is nil but SportsFieldService.Start was just called")
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
//     len(mockedSportsFieldService.StartCalls())
func (mock *SportsFieldServiceMock) StartCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockStart.RLock()
	calls = mock.calls.Start
	mock.lockStart.RUnlock()
	return calls
}

// Tenant calls TenantFunc.
func (mock *SportsFieldServiceMock) Tenant() string {
	if mock.TenantFunc == nil {
		panic("SportsFieldServiceMock.TenantFunc: method is nil but SportsFieldService.Tenant was just called")
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
//     len(mockedSportsFieldService.TenantCalls())
func (mock *SportsFieldServiceMock) TenantCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockTenant.RLock()
	calls = mock.calls.Tenant
	mock.lockTenant.RUnlock()
	return calls
}
