// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package services

import (
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"sync"
	"time"
)

// Ensure, that TempServiceQueryMock does implement TempServiceQuery.
// If this is not the case, regenerate this file with moq.
var _ TempServiceQuery = &TempServiceQueryMock{}

// TempServiceQueryMock is a mock implementation of TempServiceQuery.
//
// 	func TestSomethingThatUsesTempServiceQuery(t *testing.T) {
//
// 		// make and configure a mocked TempServiceQuery
// 		mockedTempServiceQuery := &TempServiceQueryMock{
// 			AggregateFunc: func(period string, aggregates string) TempServiceQuery {
// 				panic("mock out the Aggregate method")
// 			},
// 			BetweenTimesFunc: func(from time.Time, to time.Time) TempServiceQuery {
// 				panic("mock out the BetweenTimes method")
// 			},
// 			GetFunc: func() ([]domain.Temperature, error) {
// 				panic("mock out the Get method")
// 			},
// 		}
//
// 		// use mockedTempServiceQuery in code that requires TempServiceQuery
// 		// and then make assertions.
//
// 	}
type TempServiceQueryMock struct {
	// AggregateFunc mocks the Aggregate method.
	AggregateFunc func(period string, aggregates string) TempServiceQuery

	// BetweenTimesFunc mocks the BetweenTimes method.
	BetweenTimesFunc func(from time.Time, to time.Time) TempServiceQuery

	// GetFunc mocks the Get method.
	GetFunc func() ([]domain.Temperature, error)

	// calls tracks calls to the methods.
	calls struct {
		// Aggregate holds details about calls to the Aggregate method.
		Aggregate []struct {
			// Period is the period argument value.
			Period string
			// Aggregates is the aggregates argument value.
			Aggregates string
		}
		// BetweenTimes holds details about calls to the BetweenTimes method.
		BetweenTimes []struct {
			// From is the from argument value.
			From time.Time
			// To is the to argument value.
			To time.Time
		}
		// Get holds details about calls to the Get method.
		Get []struct {
		}
	}
	lockAggregate    sync.RWMutex
	lockBetweenTimes sync.RWMutex
	lockGet          sync.RWMutex
}

// Aggregate calls AggregateFunc.
func (mock *TempServiceQueryMock) Aggregate(period string, aggregates string) TempServiceQuery {
	callInfo := struct {
		Period     string
		Aggregates string
	}{
		Period:     period,
		Aggregates: aggregates,
	}
	mock.lockAggregate.Lock()
	mock.calls.Aggregate = append(mock.calls.Aggregate, callInfo)
	mock.lockAggregate.Unlock()
	if mock.AggregateFunc == nil {
		return mock
	}
	return mock.AggregateFunc(period, aggregates)
}

// AggregateCalls gets all the calls that were made to Aggregate.
// Check the length with:
//     len(mockedTempServiceQuery.AggregateCalls())
func (mock *TempServiceQueryMock) AggregateCalls() []struct {
	Period     string
	Aggregates string
} {
	var calls []struct {
		Period     string
		Aggregates string
	}
	mock.lockAggregate.RLock()
	calls = mock.calls.Aggregate
	mock.lockAggregate.RUnlock()
	return calls
}

// BetweenTimes calls BetweenTimesFunc.
func (mock *TempServiceQueryMock) BetweenTimes(from time.Time, to time.Time) TempServiceQuery {
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
	if mock.BetweenTimesFunc == nil {
		return mock
	}
	return mock.BetweenTimesFunc(from, to)
}

// BetweenTimesCalls gets all the calls that were made to BetweenTimes.
// Check the length with:
//     len(mockedTempServiceQuery.BetweenTimesCalls())
func (mock *TempServiceQueryMock) BetweenTimesCalls() []struct {
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

// Get calls GetFunc.
func (mock *TempServiceQueryMock) Get() ([]domain.Temperature, error) {
	callInfo := struct {
	}{}
	mock.lockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	mock.lockGet.Unlock()
	if mock.GetFunc == nil {
		var (
			temperaturesOut []domain.Temperature
			errOut          error
		)
		return temperaturesOut, errOut
	}
	return mock.GetFunc()
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//     len(mockedTempServiceQuery.GetCalls())
func (mock *TempServiceQueryMock) GetCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGet.RLock()
	calls = mock.calls.Get
	mock.lockGet.RUnlock()
	return calls
}
