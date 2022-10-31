// Code generated by mockery v2.14.1. DO NOT EDIT.

package mocks

import (
	common "github.com/ChainSafe/gossamer/lib/common"
	mock "github.com/stretchr/testify/mock"
)

// NetworkAPI is an autogenerated mock type for the NetworkAPI type
type NetworkAPI struct {
	mock.Mock
}

// AddReservedPeers provides a mock function with given fields: addrs
func (_m *NetworkAPI) AddReservedPeers(addrs ...string) error {
	_va := make([]interface{}, len(addrs))
	for _i := range addrs {
		_va[_i] = addrs[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(...string) error); ok {
		r0 = rf(addrs...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Health provides a mock function with given fields:
func (_m *NetworkAPI) Health() common.Health {
	ret := _m.Called()

	var r0 common.Health
	if rf, ok := ret.Get(0).(func() common.Health); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(common.Health)
	}

	return r0
}

// IsStopped provides a mock function with given fields:
func (_m *NetworkAPI) IsStopped() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// NetworkState provides a mock function with given fields:
func (_m *NetworkAPI) NetworkState() common.NetworkState {
	ret := _m.Called()

	var r0 common.NetworkState
	if rf, ok := ret.Get(0).(func() common.NetworkState); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(common.NetworkState)
	}

	return r0
}

// NodeRoles provides a mock function with given fields:
func (_m *NetworkAPI) NodeRoles() common.Roles {
	ret := _m.Called()

	var r0 common.Roles
	if rf, ok := ret.Get(0).(func() common.Roles); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(common.Roles)
	}

	return r0
}

// Peers provides a mock function with given fields:
func (_m *NetworkAPI) Peers() []common.PeerInfo {
	ret := _m.Called()

	var r0 []common.PeerInfo
	if rf, ok := ret.Get(0).(func() []common.PeerInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.PeerInfo)
		}
	}

	return r0
}

// RemoveReservedPeers provides a mock function with given fields: addrs
func (_m *NetworkAPI) RemoveReservedPeers(addrs ...string) error {
	_va := make([]interface{}, len(addrs))
	for _i := range addrs {
		_va[_i] = addrs[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(...string) error); ok {
		r0 = rf(addrs...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields:
func (_m *NetworkAPI) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StartingBlock provides a mock function with given fields:
func (_m *NetworkAPI) StartingBlock() int64 {
	ret := _m.Called()

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	return r0
}

// Stop provides a mock function with given fields:
func (_m *NetworkAPI) Stop() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewNetworkAPI interface {
	mock.TestingT
	Cleanup(func())
}

// NewNetworkAPI creates a new instance of NetworkAPI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewNetworkAPI(t mockConstructorTestingTNewNetworkAPI) *NetworkAPI {
	mock := &NetworkAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
