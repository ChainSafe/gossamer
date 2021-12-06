// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// BlockProducerAPI is an autogenerated mock type for the BlockProducerAPI type
type BlockProducerAPI struct {
	mock.Mock
}

// EpochLength provides a mock function with given fields:
func (_m *BlockProducerAPI) EpochLength() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// Pause provides a mock function with given fields:
func (_m *BlockProducerAPI) Pause() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Resume provides a mock function with given fields:
func (_m *BlockProducerAPI) Resume() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SlotDuration provides a mock function with given fields:
func (_m *BlockProducerAPI) SlotDuration() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}
