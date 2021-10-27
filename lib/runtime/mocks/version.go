// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	mock "github.com/stretchr/testify/mock"
)

// Version is an autogenerated mock type for the Version type
type Version struct {
	mock.Mock
}

// APIItems provides a mock function with given fields:
func (_m *Version) APIItems() []runtime.APIItem {
	ret := _m.Called()

	var r0 []runtime.APIItem
	if rf, ok := ret.Get(0).(func() []runtime.APIItem); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]runtime.APIItem)
		}
	}

	return r0
}

// AuthoringVersion provides a mock function with given fields:
func (_m *Version) AuthoringVersion() uint32 {
	ret := _m.Called()

	var r0 uint32
	if rf, ok := ret.Get(0).(func() uint32); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint32)
	}

	return r0
}

// Encode provides a mock function with given fields:
func (_m *Version) Encode() ([]byte, error) {
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

// ImplName provides a mock function with given fields:
func (_m *Version) ImplName() []byte {
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

// ImplVersion provides a mock function with given fields:
func (_m *Version) ImplVersion() uint32 {
	ret := _m.Called()

	var r0 uint32
	if rf, ok := ret.Get(0).(func() uint32); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint32)
	}

	return r0
}

// SpecName provides a mock function with given fields:
func (_m *Version) SpecName() []byte {
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

// SpecVersion provides a mock function with given fields:
func (_m *Version) SpecVersion() uint32 {
	ret := _m.Called()

	var r0 uint32
	if rf, ok := ret.Get(0).(func() uint32); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint32)
	}

	return r0
}

// TransactionVersion provides a mock function with given fields:
func (_m *Version) TransactionVersion() uint32 {
	ret := _m.Called()

	var r0 uint32
	if rf, ok := ret.Get(0).(func() uint32); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint32)
	}

	return r0
}
