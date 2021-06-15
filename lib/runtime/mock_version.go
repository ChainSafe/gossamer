// Code generated by mockery v2.8.0. DO NOT EDIT.

package runtime

import mock "github.com/stretchr/testify/mock"

// MockVersion is an autogenerated mock type for the Version type
type MockVersion struct {
	mock.Mock
}

// APIItems provides a mock function with given fields:
func (_m *MockVersion) APIItems() []*APIItem {
	ret := _m.Called()

	var r0 []*APIItem
	if rf, ok := ret.Get(0).(func() []*APIItem); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*APIItem)
		}
	}

	return r0
}

// AuthoringVersion provides a mock function with given fields:
func (_m *MockVersion) AuthoringVersion() uint32 {
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
func (_m *MockVersion) Encode() ([]byte, error) {
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
func (_m *MockVersion) ImplName() []byte {
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
func (_m *MockVersion) ImplVersion() uint32 {
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
func (_m *MockVersion) SpecName() []byte {
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
func (_m *MockVersion) SpecVersion() uint32 {
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
func (_m *MockVersion) TransactionVersion() uint32 {
	ret := _m.Called()

	var r0 uint32
	if rf, ok := ret.Get(0).(func() uint32); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint32)
	}

	return r0
}
