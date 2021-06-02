// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// BasicStorage is an autogenerated mock type for the BasicStorage type
type BasicStorage struct {
	mock.Mock
}

// Get provides a mock function with given fields: key
func (_m *BasicStorage) Get(key []byte) ([]byte, error) {
	ret := _m.Called(key)

	var r0 []byte
	if rf, ok := ret.Get(0).(func([]byte) []byte); ok {
		r0 = rf(key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Put provides a mock function with given fields: key, value
func (_m *BasicStorage) Put(key []byte, value []byte) error {
	ret := _m.Called(key, value)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, []byte) error); ok {
		r0 = rf(key, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
