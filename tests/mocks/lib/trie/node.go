// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// node is an autogenerated mock type for the node type
type node struct {
	mock.Mock
}

// String provides a mock function with given fields:
func (_m *node) String() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// decode provides a mock function with given fields: r, h
func (_m *node) decode(r io.Reader, h byte) error {
	ret := _m.Called(r, h)

	var r0 error
	if rf, ok := ret.Get(0).(func(io.Reader, byte) error); ok {
		r0 = rf(r, h)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// encode provides a mock function with given fields:
func (_m *node) encode() ([]byte, error) {
	ret := _m.Called()

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// encodeAndHash provides a mock function with given fields:
func (_m *node) encodeAndHash() ([]byte, []byte, error) {
	ret := _m.Called()

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 []byte
	if rf, ok := ret.Get(1).(func() []byte); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// getGeneration provides a mock function with given fields:
func (_m *node) getGeneration() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// getHash provides a mock function with given fields:
func (_m *node) getHash() []byte {
	ret := _m.Called()

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0
}

// isDirty provides a mock function with given fields:
func (_m *node) isDirty() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// setDirty provides a mock function with given fields: dirty
func (_m *node) setDirty(dirty bool) {
	_m.Called(dirty)
}

// setEncodingAndHash provides a mock function with given fields: _a0, _a1
func (_m *node) setEncodingAndHash(_a0 []byte, _a1 []byte) {
	_m.Called(_a0, _a1)
}

// setKey provides a mock function with given fields: key
func (_m *node) setKey(key []byte) {
	_m.Called(key)
}
