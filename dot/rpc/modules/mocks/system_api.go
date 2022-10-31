// Code generated by mockery v2.14.1. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// SystemAPI is an autogenerated mock type for the SystemAPI type
type SystemAPI struct {
	mock.Mock
}

// ChainName provides a mock function with given fields:
func (_m *SystemAPI) ChainName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// ChainType provides a mock function with given fields:
func (_m *SystemAPI) ChainType() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Properties provides a mock function with given fields:
func (_m *SystemAPI) Properties() map[string]interface{} {
	ret := _m.Called()

	var r0 map[string]interface{}
	if rf, ok := ret.Get(0).(func() map[string]interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]interface{})
		}
	}

	return r0
}

// SystemName provides a mock function with given fields:
func (_m *SystemAPI) SystemName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// SystemVersion provides a mock function with given fields:
func (_m *SystemAPI) SystemVersion() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

type mockConstructorTestingTNewSystemAPI interface {
	mock.TestingT
	Cleanup(func())
}

// NewSystemAPI creates a new instance of SystemAPI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSystemAPI(t mockConstructorTestingTNewSystemAPI) *SystemAPI {
	mock := &SystemAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
