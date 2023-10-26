// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package temperature

import (
	"context"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"sync"
	"time"
)

// Ensure, that TempServiceQueryMock does implement TempServiceQuery.
// If this is not the case, regenerate this file with moq.
var _ TempServiceQuery = &TempServiceQueryMock{}

// TempServiceQueryMock is a mock implementation of TempServiceQuery.
//
//	func TestSomethingThatUsesTempServiceQuery(t *testing.T) {
//
//		// make and configure a mocked TempServiceQuery
//		mockedTempServiceQuery := &TempServiceQueryMock{
//			AggregateFunc: func(period string, aggregates string) TempServiceQuery {
//				panic("mock out the Aggregate method")
//			},
//			BetweenTimesFunc: func(from time.Time, to time.Time) TempServiceQuery {
//				panic("mock out the BetweenTimes method")
//			},
//			GetFunc: func(ctx context.Context) ([]domain.Sensor, error) {
//				panic("mock out the Get method")
//			},
//			SensorFunc: func(sensor string) TempServiceQuery {
//				panic("mock out the Sensor method")
//			},
//		}
//
//		// use mockedTempServiceQuery in code that requires TempServiceQuery
//		// and then make assertions.
//
//	}
type TempServiceQueryMock struct {
	// AggregateFunc mocks the Aggregate method.
	AggregateFunc func(period string, aggregates string) TempServiceQuery

	// BetweenTimesFunc mocks the BetweenTimes method.
	BetweenTimesFunc func(from time.Time, to time.Time) TempServiceQuery

	// GetFunc mocks the Get method.
	GetFunc func(ctx context.Context) ([]domain.Sensor, error)

	// SensorFunc mocks the Sensor method.
	SensorFunc func(sensor string) TempServiceQuery

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
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// Sensor holds details about calls to the Sensor method.
		Sensor []struct {
			// Sensor is the sensor argument value.
			Sensor string
		}
	}
	lockAggregate    sync.RWMutex
	lockBetweenTimes sync.RWMutex
	lockGet          sync.RWMutex
	lockSensor       sync.RWMutex
}

// Aggregate calls AggregateFunc.
func (mock *TempServiceQueryMock) Aggregate(period string, aggregates string) TempServiceQuery {
	if mock.AggregateFunc == nil {
		panic("TempServiceQueryMock.AggregateFunc: method is nil but TempServiceQuery.Aggregate was just called")
	}
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
	return mock.AggregateFunc(period, aggregates)
}

// AggregateCalls gets all the calls that were made to Aggregate.
// Check the length with:
//
//	len(mockedTempServiceQuery.AggregateCalls())
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
	if mock.BetweenTimesFunc == nil {
		panic("TempServiceQueryMock.BetweenTimesFunc: method is nil but TempServiceQuery.BetweenTimes was just called")
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
	return mock.BetweenTimesFunc(from, to)
}

// BetweenTimesCalls gets all the calls that were made to BetweenTimes.
// Check the length with:
//
//	len(mockedTempServiceQuery.BetweenTimesCalls())
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
func (mock *TempServiceQueryMock) Get(ctx context.Context) ([]domain.Sensor, error) {
	if mock.GetFunc == nil {
		panic("TempServiceQueryMock.GetFunc: method is nil but TempServiceQuery.Get was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	mock.lockGet.Unlock()
	return mock.GetFunc(ctx)
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//
//	len(mockedTempServiceQuery.GetCalls())
func (mock *TempServiceQueryMock) GetCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockGet.RLock()
	calls = mock.calls.Get
	mock.lockGet.RUnlock()
	return calls
}

// Sensor calls SensorFunc.
func (mock *TempServiceQueryMock) Sensor(sensor string) TempServiceQuery {
	if mock.SensorFunc == nil {
		panic("TempServiceQueryMock.SensorFunc: method is nil but TempServiceQuery.Sensor was just called")
	}
	callInfo := struct {
		Sensor string
	}{
		Sensor: sensor,
	}
	mock.lockSensor.Lock()
	mock.calls.Sensor = append(mock.calls.Sensor, callInfo)
	mock.lockSensor.Unlock()
	return mock.SensorFunc(sensor)
}

// SensorCalls gets all the calls that were made to Sensor.
// Check the length with:
//
//	len(mockedTempServiceQuery.SensorCalls())
func (mock *TempServiceQueryMock) SensorCalls() []struct {
	Sensor string
} {
	var calls []struct {
		Sensor string
	}
	mock.lockSensor.RLock()
	calls = mock.calls.Sensor
	mock.lockSensor.RUnlock()
	return calls
}
