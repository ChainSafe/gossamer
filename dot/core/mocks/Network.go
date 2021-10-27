// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	network "github.com/ChainSafe/gossamer/dot/network"
	mock "github.com/stretchr/testify/mock"

	peer "github.com/libp2p/go-libp2p-core/peer"

	peerset "github.com/ChainSafe/gossamer/dot/peerset"
)

// Network is an autogenerated mock type for the Network type
type Network struct {
	mock.Mock
}

// GossipMessage provides a mock function with given fields: _a0
func (_m *Network) GossipMessage(_a0 network.NotificationsMessage) {
	_m.Called(_a0)
}

// IsSynced provides a mock function with given fields:
func (_m *Network) IsSynced() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// ReportPeer provides a mock function with given fields: change, p
func (_m *MockNetwork) ReportPeer(change peerset.ReputationChange, p peer.ID) {
	_m.Called(change, p)
}
