// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/network (interfaces: RequestResponseProtocol)

// Package sync is a generated GoMock package.
package sync

import (
	reflect "reflect"

	network "github.com/ChainSafe/gossamer/dot/network"
	gomock "github.com/golang/mock/gomock"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

// MockRequestResponseProtocol is a mock of RequestResponseProtocol interface.
type MockRequestResponseProtocol struct {
	ctrl     *gomock.Controller
	recorder *MockRequestResponseProtocolMockRecorder
}

// MockRequestResponseProtocolMockRecorder is the mock recorder for MockRequestResponseProtocol.
type MockRequestResponseProtocolMockRecorder struct {
	mock *MockRequestResponseProtocol
}

// NewMockRequestResponseProtocol creates a new mock instance.
func NewMockRequestResponseProtocol(ctrl *gomock.Controller) *MockRequestResponseProtocol {
	mock := &MockRequestResponseProtocol{ctrl: ctrl}
	mock.recorder = &MockRequestResponseProtocolMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRequestResponseProtocol) EXPECT() *MockRequestResponseProtocolMockRecorder {
	return m.recorder
}

// DoRequest mocks base method.
func (m *MockRequestResponseProtocol) DoRequest(arg0 peer.ID, arg1 network.Message, arg2 network.ResponseMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DoRequest", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DoRequest indicates an expected call of DoRequest.
func (mr *MockRequestResponseProtocolMockRecorder) DoRequest(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DoRequest", reflect.TypeOf((*MockRequestResponseProtocol)(nil).DoRequest), arg0, arg1, arg2)
}
