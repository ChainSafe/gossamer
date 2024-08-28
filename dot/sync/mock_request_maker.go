// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/network (interfaces: RequestMaker)
//
// Generated by this command:
//
//	mockgen -destination=mock_request_maker.go -package sync github.com/ChainSafe/gossamer/dot/network RequestMaker
//

// Package sync is a generated GoMock package.
package sync

import (
	reflect "reflect"

	messages "github.com/ChainSafe/gossamer/dot/network/messages"
	peer "github.com/libp2p/go-libp2p/core/peer"
	gomock "go.uber.org/mock/gomock"
)

// MockRequestMaker is a mock of RequestMaker interface.
type MockRequestMaker struct {
	ctrl     *gomock.Controller
	recorder *MockRequestMakerMockRecorder
}

// MockRequestMakerMockRecorder is the mock recorder for MockRequestMaker.
type MockRequestMakerMockRecorder struct {
	mock *MockRequestMaker
}

// NewMockRequestMaker creates a new mock instance.
func NewMockRequestMaker(ctrl *gomock.Controller) *MockRequestMaker {
	mock := &MockRequestMaker{ctrl: ctrl}
	mock.recorder = &MockRequestMakerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRequestMaker) EXPECT() *MockRequestMakerMockRecorder {
	return m.recorder
}

// Do mocks base method.
func (m *MockRequestMaker) Do(arg0 peer.ID, arg1, arg2 messages.P2PMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Do", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Do indicates an expected call of Do.
func (mr *MockRequestMakerMockRecorder) Do(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockRequestMaker)(nil).Do), arg0, arg1, arg2)
}
