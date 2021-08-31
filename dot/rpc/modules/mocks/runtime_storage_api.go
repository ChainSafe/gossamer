// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// MockRuntimeStorageAPI is an autogenerated mock type for the RuntimeStorageAPI type
type MockRuntimeStorageAPI struct {
	mock.Mock
}

// GetLocal provides a mock function with given fields: k
func (_m *MockRuntimeStorageAPI) GetLocal(k []byte) ([]byte, error) {
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
func (_m *MockRuntimeStorageAPI) GetPersistent(k []byte) ([]byte, error) {
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
func (_m *MockRuntimeStorageAPI) SetLocal(k, v []byte) error {
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
func (_m *MockRuntimeStorageAPI) SetPersistent(k, v []byte) error {
	ret := _m.Called(k, v)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, []byte) error); ok {
		r0 = rf(k, v)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
