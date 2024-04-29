// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/parachain/collator-protocol (interfaces: Network)

// Package collatorprotocol is a generated GoMock package.
package collatorprotocol

import (
	reflect "reflect"
	time "time"

	network "github.com/ChainSafe/gossamer/dot/network"
	peerset "github.com/ChainSafe/gossamer/dot/peerset"
	gomock "go.uber.org/mock/gomock"
	peer "github.com/libp2p/go-libp2p/core/peer"
	protocol "github.com/libp2p/go-libp2p/core/protocol"
)

// MockNetwork is a mock of Network interface.
type MockNetwork struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkMockRecorder
}

// MockNetworkMockRecorder is the mock recorder for MockNetwork.
type MockNetworkMockRecorder struct {
	mock *MockNetwork
}

// NewMockNetwork creates a new mock instance.
func NewMockNetwork(ctrl *gomock.Controller) *MockNetwork {
	mock := &MockNetwork{ctrl: ctrl}
	mock.recorder = &MockNetworkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNetwork) EXPECT() *MockNetworkMockRecorder {
	return m.recorder
}

// FreeNetworkEventsChannel mocks base method.
func (m *MockNetwork) FreeNetworkEventsChannel(arg0 chan *network.NetworkEventInfo) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "FreeNetworkEventsChannel", arg0)
}

// FreeNetworkEventsChannel indicates an expected call of FreeNetworkEventsChannel.
func (mr *MockNetworkMockRecorder) FreeNetworkEventsChannel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeNetworkEventsChannel", reflect.TypeOf((*MockNetwork)(nil).FreeNetworkEventsChannel), arg0)
}

// GetNetworkEventsChannel mocks base method.
func (m *MockNetwork) GetNetworkEventsChannel() chan *network.NetworkEventInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNetworkEventsChannel")
	ret0, _ := ret[0].(chan *network.NetworkEventInfo)
	return ret0
}

// GetNetworkEventsChannel indicates an expected call of GetNetworkEventsChannel.
func (mr *MockNetworkMockRecorder) GetNetworkEventsChannel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNetworkEventsChannel", reflect.TypeOf((*MockNetwork)(nil).GetNetworkEventsChannel))
}

// GetRequestResponseProtocol mocks base method.
func (m *MockNetwork) GetRequestResponseProtocol(arg0 string, arg1 time.Duration, arg2 uint64) *network.RequestResponseProtocol {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRequestResponseProtocol", arg0, arg1, arg2)
	ret0, _ := ret[0].(*network.RequestResponseProtocol)
	return ret0
}

// GetRequestResponseProtocol indicates an expected call of GetRequestResponseProtocol.
func (mr *MockNetworkMockRecorder) GetRequestResponseProtocol(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRequestResponseProtocol", reflect.TypeOf((*MockNetwork)(nil).GetRequestResponseProtocol), arg0, arg1, arg2)
}

// GossipMessage mocks base method.
func (m *MockNetwork) GossipMessage(arg0 network.NotificationsMessage) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "GossipMessage", arg0)
}

// GossipMessage indicates an expected call of GossipMessage.
func (mr *MockNetworkMockRecorder) GossipMessage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GossipMessage", reflect.TypeOf((*MockNetwork)(nil).GossipMessage), arg0)
}

// RegisterNotificationsProtocol mocks base method.
func (m *MockNetwork) RegisterNotificationsProtocol(arg0 protocol.ID, arg1 network.MessageType, arg2 func() (network.Handshake, error), arg3 func([]byte) (network.Handshake, error), arg4 func(peer.ID, network.Handshake) error, arg5 func([]byte) (network.NotificationsMessage, error), arg6 func(peer.ID, network.NotificationsMessage) (bool, error), arg7 func(peer.ID, network.NotificationsMessage), arg8 uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterNotificationsProtocol", arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8)
	ret0, _ := ret[0].(error)
	return ret0
}

// RegisterNotificationsProtocol indicates an expected call of RegisterNotificationsProtocol.
func (mr *MockNetworkMockRecorder) RegisterNotificationsProtocol(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterNotificationsProtocol", reflect.TypeOf((*MockNetwork)(nil).RegisterNotificationsProtocol), arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8)
}

// ReportPeer mocks base method.
func (m *MockNetwork) ReportPeer(arg0 peerset.ReputationChange, arg1 peer.ID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ReportPeer", arg0, arg1)
}

// ReportPeer indicates an expected call of ReportPeer.
func (mr *MockNetworkMockRecorder) ReportPeer(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportPeer", reflect.TypeOf((*MockNetwork)(nil).ReportPeer), arg0, arg1)
}

// SendMessage mocks base method.
func (m *MockNetwork) SendMessage(arg0 peer.ID, arg1 network.NotificationsMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessage", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockNetworkMockRecorder) SendMessage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockNetwork)(nil).SendMessage), arg0, arg1)
}
