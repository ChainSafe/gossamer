// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	types "github.com/ChainSafe/gossamer/dot/types"
	mock "github.com/stretchr/testify/mock"
)

// EpochState is an autogenerated mock type for the EpochState type
type EpochState struct {
	mock.Mock
}

// GetCurrentEpoch provides a mock function with given fields:
func (_m *EpochState) GetCurrentEpoch() (uint64, error) {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEpochForBlock provides a mock function with given fields: header
func (_m *EpochState) GetEpochForBlock(header *types.Header) (uint64, error) {
	ret := _m.Called(header)

	var r0 uint64
	if rf, ok := ret.Get(0).(func(*types.Header) uint64); ok {
		r0 = rf(header)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.Header) error); ok {
		r1 = rf(header)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetConfigData provides a mock function with given fields: epoch, info
func (_m *EpochState) SetConfigData(epoch uint64, info *types.ConfigData) error {
	ret := _m.Called(epoch, info)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64, *types.ConfigData) error); ok {
		r0 = rf(epoch, info)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetCurrentEpoch provides a mock function with given fields: epoch
func (_m *EpochState) SetCurrentEpoch(epoch uint64) error {
	ret := _m.Called(epoch)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64) error); ok {
		r0 = rf(epoch)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetEpochData provides a mock function with given fields: epoch, info
func (_m *EpochState) SetEpochData(epoch uint64, info *types.EpochData) error {
	ret := _m.Called(epoch, info)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64, *types.EpochData) error); ok {
		r0 = rf(epoch, info)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
