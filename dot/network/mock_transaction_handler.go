// Code generated by mockery v2.8.0. DO NOT EDIT.

package network

import mock "github.com/stretchr/testify/mock"

// MockTransactionHandler is an autogenerated mock type for the TransactionHandler type
type MockTransactionHandler struct {
	mock.Mock
}

// HandleTransactionMessage provides a mock function with given fields: _a0
func (_m *MockTransactionHandler) HandleTransactionMessage(_a0 *TransactionMessage) (bool, error) {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(*TransactionMessage) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*TransactionMessage) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
