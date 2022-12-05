// Code generated by mockery v2.14.1. DO NOT EDIT.

package mocks

import (
	ed25519 "github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	mock "github.com/stretchr/testify/mock"

	types "github.com/ChainSafe/gossamer/dot/types"
)

// BlockFinalityAPI is an autogenerated mock type for the BlockFinalityAPI type
type BlockFinalityAPI struct {
	mock.Mock
}

// GetRound provides a mock function with given fields:
func (_m *BlockFinalityAPI) GetRound() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// GetSetID provides a mock function with given fields:
func (_m *BlockFinalityAPI) GetSetID() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// GetVoters provides a mock function with given fields:
func (_m *BlockFinalityAPI) GetVoters() types.GrandpaVoters {
	ret := _m.Called()

	var r0 types.GrandpaVoters
	if rf, ok := ret.Get(0).(func() types.GrandpaVoters); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.GrandpaVoters)
		}
	}

	return r0
}

// PreCommits provides a mock function with given fields:
func (_m *BlockFinalityAPI) PreCommits() []ed25519.PublicKeyBytes {
	ret := _m.Called()

	var r0 []ed25519.PublicKeyBytes
	if rf, ok := ret.Get(0).(func() []ed25519.PublicKeyBytes); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ed25519.PublicKeyBytes)
		}
	}

	return r0
}

// PreVotes provides a mock function with given fields:
func (_m *BlockFinalityAPI) PreVotes() []ed25519.PublicKeyBytes {
	ret := _m.Called()

	var r0 []ed25519.PublicKeyBytes
	if rf, ok := ret.Get(0).(func() []ed25519.PublicKeyBytes); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ed25519.PublicKeyBytes)
		}
	}

	return r0
}

type mockConstructorTestingTNewBlockFinalityAPI interface {
	mock.TestingT
	Cleanup(func())
}

// NewBlockFinalityAPI creates a new instance of BlockFinalityAPI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBlockFinalityAPI(t mockConstructorTestingTNewBlockFinalityAPI) *BlockFinalityAPI {
	mock := &BlockFinalityAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
