// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package weather

import (
	"context"
	"github.com/diwise/api-opendata/internal/pkg/domain"
	"sync"
	"time"
)

// Ensure, that WeatherServiceQueryMock does implement WeatherServiceQuery.
// If this is not the case, regenerate this file with moq.
var _ WeatherServiceQuery = &WeatherServiceQueryMock{}

// WeatherServiceQueryMock is a mock implementation of WeatherServiceQuery.
//
//	func TestSomethingThatUsesWeatherServiceQuery(t *testing.T) {
//
//		// make and configure a mocked WeatherServiceQuery
//		mockedWeatherServiceQuery := &WeatherServiceQueryMock{
//			AggrFunc: func(res string) WeatherServiceQuery {
//				panic("mock out the Aggr method")
//			},
//			BetweenTimesFunc: func(from time.Time, to time.Time) WeatherServiceQuery {
//				panic("mock out the BetweenTimes method")
//			},
//			GetFunc: func(ctx context.Context) ([]domain.Weather, error) {
//				panic("mock out the Get method")
//			},
//			GetByIDFunc: func(ctx context.Context) (domain.Weather, error) {
//				panic("mock out the GetByID method")
//			},
//			IDFunc: func(id string) WeatherServiceQuery {
//				panic("mock out the ID method")
//			},
//			NearPointFunc: func(distance int64, lat float64, lon float64) WeatherServiceQuery {
//				panic("mock out the NearPoint method")
//			},
//		}
//
//		// use mockedWeatherServiceQuery in code that requires WeatherServiceQuery
//		// and then make assertions.
//
//	}
type WeatherServiceQueryMock struct {
	// AggrFunc mocks the Aggr method.
	AggrFunc func(res string) WeatherServiceQuery

	// BetweenTimesFunc mocks the BetweenTimes method.
	BetweenTimesFunc func(from time.Time, to time.Time) WeatherServiceQuery

	// GetFunc mocks the Get method.
	GetFunc func(ctx context.Context) ([]domain.Weather, error)

	// GetByIDFunc mocks the GetByID method.
	GetByIDFunc func(ctx context.Context) (domain.Weather, error)

	// IDFunc mocks the ID method.
	IDFunc func(id string) WeatherServiceQuery

	// NearPointFunc mocks the NearPoint method.
	NearPointFunc func(distance int64, lat float64, lon float64) WeatherServiceQuery

	// calls tracks calls to the methods.
	calls struct {
		// Aggr holds details about calls to the Aggr method.
		Aggr []struct {
			// Res is the res argument value.
			Res string
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
		// GetByID holds details about calls to the GetByID method.
		GetByID []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// ID holds details about calls to the ID method.
		ID []struct {
			// ID is the id argument value.
			ID string
		}
		// NearPoint holds details about calls to the NearPoint method.
		NearPoint []struct {
			// Distance is the distance argument value.
			Distance int64
			// Lat is the lat argument value.
			Lat float64
			// Lon is the lon argument value.
			Lon float64
		}
	}
	lockAggr         sync.RWMutex
	lockBetweenTimes sync.RWMutex
	lockGet          sync.RWMutex
	lockGetByID      sync.RWMutex
	lockID           sync.RWMutex
	lockNearPoint    sync.RWMutex
}

// Aggr calls AggrFunc.
func (mock *WeatherServiceQueryMock) Aggr(res string) WeatherServiceQuery {
	if mock.AggrFunc == nil {
		panic("WeatherServiceQueryMock.AggrFunc: method is nil but WeatherServiceQuery.Aggr was just called")
	}
	callInfo := struct {
		Res string
	}{
		Res: res,
	}
	mock.lockAggr.Lock()
	mock.calls.Aggr = append(mock.calls.Aggr, callInfo)
	mock.lockAggr.Unlock()
	return mock.AggrFunc(res)
}

// AggrCalls gets all the calls that were made to Aggr.
// Check the length with:
//
//	len(mockedWeatherServiceQuery.AggrCalls())
func (mock *WeatherServiceQueryMock) AggrCalls() []struct {
	Res string
} {
	var calls []struct {
		Res string
	}
	mock.lockAggr.RLock()
	calls = mock.calls.Aggr
	mock.lockAggr.RUnlock()
	return calls
}

// BetweenTimes calls BetweenTimesFunc.
func (mock *WeatherServiceQueryMock) BetweenTimes(from time.Time, to time.Time) WeatherServiceQuery {
	if mock.BetweenTimesFunc == nil {
		panic("WeatherServiceQueryMock.BetweenTimesFunc: method is nil but WeatherServiceQuery.BetweenTimes was just called")
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
//	len(mockedWeatherServiceQuery.BetweenTimesCalls())
func (mock *WeatherServiceQueryMock) BetweenTimesCalls() []struct {
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
func (mock *WeatherServiceQueryMock) Get(ctx context.Context) ([]domain.Weather, error) {
	if mock.GetFunc == nil {
		panic("WeatherServiceQueryMock.GetFunc: method is nil but WeatherServiceQuery.Get was just called")
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
//	len(mockedWeatherServiceQuery.GetCalls())
func (mock *WeatherServiceQueryMock) GetCalls() []struct {
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

// GetByID calls GetByIDFunc.
func (mock *WeatherServiceQueryMock) GetByID(ctx context.Context) (domain.Weather, error) {
	if mock.GetByIDFunc == nil {
		panic("WeatherServiceQueryMock.GetByIDFunc: method is nil but WeatherServiceQuery.GetByID was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockGetByID.Lock()
	mock.calls.GetByID = append(mock.calls.GetByID, callInfo)
	mock.lockGetByID.Unlock()
	return mock.GetByIDFunc(ctx)
}

// GetByIDCalls gets all the calls that were made to GetByID.
// Check the length with:
//
//	len(mockedWeatherServiceQuery.GetByIDCalls())
func (mock *WeatherServiceQueryMock) GetByIDCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockGetByID.RLock()
	calls = mock.calls.GetByID
	mock.lockGetByID.RUnlock()
	return calls
}

// ID calls IDFunc.
func (mock *WeatherServiceQueryMock) ID(id string) WeatherServiceQuery {
	if mock.IDFunc == nil {
		panic("WeatherServiceQueryMock.IDFunc: method is nil but WeatherServiceQuery.ID was just called")
	}
	callInfo := struct {
		ID string
	}{
		ID: id,
	}
	mock.lockID.Lock()
	mock.calls.ID = append(mock.calls.ID, callInfo)
	mock.lockID.Unlock()
	return mock.IDFunc(id)
}

// IDCalls gets all the calls that were made to ID.
// Check the length with:
//
//	len(mockedWeatherServiceQuery.IDCalls())
func (mock *WeatherServiceQueryMock) IDCalls() []struct {
	ID string
} {
	var calls []struct {
		ID string
	}
	mock.lockID.RLock()
	calls = mock.calls.ID
	mock.lockID.RUnlock()
	return calls
}

// NearPoint calls NearPointFunc.
func (mock *WeatherServiceQueryMock) NearPoint(distance int64, lat float64, lon float64) WeatherServiceQuery {
	if mock.NearPointFunc == nil {
		panic("WeatherServiceQueryMock.NearPointFunc: method is nil but WeatherServiceQuery.NearPoint was just called")
	}
	callInfo := struct {
		Distance int64
		Lat      float64
		Lon      float64
	}{
		Distance: distance,
		Lat:      lat,
		Lon:      lon,
	}
	mock.lockNearPoint.Lock()
	mock.calls.NearPoint = append(mock.calls.NearPoint, callInfo)
	mock.lockNearPoint.Unlock()
	return mock.NearPointFunc(distance, lat, lon)
}

// NearPointCalls gets all the calls that were made to NearPoint.
// Check the length with:
//
//	len(mockedWeatherServiceQuery.NearPointCalls())
func (mock *WeatherServiceQueryMock) NearPointCalls() []struct {
	Distance int64
	Lat      float64
	Lon      float64
} {
	var calls []struct {
		Distance int64
		Lat      float64
		Lon      float64
	}
	mock.lockNearPoint.RLock()
	calls = mock.calls.NearPoint
	mock.lockNearPoint.RUnlock()
	return calls
}
