// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package temperature

import (
	"sync"
)

// Ensure, that TempServiceMock does implement TempService.
// If this is not the case, regenerate this file with moq.
var _ TempService = &TempServiceMock{}

// TempServiceMock is a mock implementation of TempService.
//
// 	func TestSomethingThatUsesTempService(t *testing.T) {
//
// 		// make and configure a mocked TempService
// 		mockedTempService := &TempServiceMock{
// 			QueryFunc: func() TempServiceQuery {
// 				panic("mock out the Query method")
// 			},
// 		}
//
// 		// use mockedTempService in code that requires TempService
// 		// and then make assertions.
//
// 	}
type TempServiceMock struct {
	// QueryFunc mocks the Query method.
	QueryFunc func() TempServiceQuery

	// calls tracks calls to the methods.
	calls struct {
		// Query holds details about calls to the Query method.
		Query []struct {
		}
	}
	lockQuery sync.RWMutex
}

// Query calls QueryFunc.
func (mock *TempServiceMock) Query() TempServiceQuery {
	callInfo := struct {
	}{}
	mock.lockQuery.Lock()
	mock.calls.Query = append(mock.calls.Query, callInfo)
	mock.lockQuery.Unlock()
	if mock.QueryFunc == nil {
		var (
			tempServiceQueryOut TempServiceQuery
		)
		return tempServiceQueryOut
	}
	return mock.QueryFunc()
}

// QueryCalls gets all the calls that were made to Query.
// Check the length with:
//     len(mockedTempService.QueryCalls())
func (mock *TempServiceMock) QueryCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockQuery.RLock()
	calls = mock.calls.Query
	mock.lockQuery.RUnlock()
	return calls
}
