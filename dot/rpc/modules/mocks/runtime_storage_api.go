// Code generated by mockery v2.14.1. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// RuntimeStorageAPI is an autogenerated mock type for the RuntimeStorageAPI type
type RuntimeStorageAPI struct {
	mock.Mock
}

// GetLocal provides a mock function with given fields: k
func (_m *RuntimeStorageAPI) GetLocal(k []byte) ([]byte, error) {
	ret := _m.Called(k)

	var r0 []byte
	if rf, ok := ret.Get(0).(func([]byte) []byte); ok {
		r0 = rf(k)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(k)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPersistent provides a mock function with given fields: k
func (_m *RuntimeStorageAPI) GetPersistent(k []byte) ([]byte, error) {
	ret := _m.Called(k)

	var r0 []byte
	if rf, ok := ret.Get(0).(func([]byte) []byte); ok {
		r0 = rf(k)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(k)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetLocal provides a mock function with given fields: k, v
func (_m *RuntimeStorageAPI) SetLocal(k []byte, v []byte) error {
	ret := _m.Called(k, v)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, []byte) error); ok {
		r0 = rf(k, v)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetPersistent provides a mock function with given fields: k, v
func (_m *RuntimeStorageAPI) SetPersistent(k []byte, v []byte) error {
	ret := _m.Called(k, v)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, []byte) error); ok {
		r0 = rf(k, v)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewRuntimeStorageAPI interface {
	mock.TestingT
	Cleanup(func())
}

// NewRuntimeStorageAPI creates a new instance of RuntimeStorageAPI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewRuntimeStorageAPI(t mockConstructorTestingTNewRuntimeStorageAPI) *RuntimeStorageAPI {
	mock := &RuntimeStorageAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
