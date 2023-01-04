// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/network (interfaces: Syncer)

// Package modules is a generated GoMock package.
package modules

import (
	reflect "reflect"

	network "github.com/ChainSafe/gossamer/dot/network"
	gomock "github.com/golang/mock/gomock"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

// MockSyncer is a mock of Syncer interface.
type MockSyncer struct {
	ctrl     *gomock.Controller
	recorder *MockSyncerMockRecorder
}

// MockSyncerMockRecorder is the mock recorder for MockSyncer.
type MockSyncerMockRecorder struct {
	mock *MockSyncer
}

// NewMockSyncer creates a new mock instance.
func NewMockSyncer(ctrl *gomock.Controller) *MockSyncer {
	mock := &MockSyncer{ctrl: ctrl}
	mock.recorder = &MockSyncerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSyncer) EXPECT() *MockSyncerMockRecorder {
	return m.recorder
}

// CreateBlockResponse mocks base method.
func (m *MockSyncer) CreateBlockResponse(arg0 *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBlockResponse", arg0)
	ret0, _ := ret[0].(*network.BlockResponseMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBlockResponse indicates an expected call of CreateBlockResponse.
func (mr *MockSyncerMockRecorder) CreateBlockResponse(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBlockResponse", reflect.TypeOf((*MockSyncer)(nil).CreateBlockResponse), arg0)
}

// HandleBlockAnnounce mocks base method.
func (m *MockSyncer) HandleBlockAnnounce(arg0 peer.ID, arg1 *network.BlockAnnounceMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleBlockAnnounce", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleBlockAnnounce indicates an expected call of HandleBlockAnnounce.
func (mr *MockSyncerMockRecorder) HandleBlockAnnounce(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleBlockAnnounce", reflect.TypeOf((*MockSyncer)(nil).HandleBlockAnnounce), arg0, arg1)
}

// HandleBlockAnnounceHandshake mocks base method.
func (m *MockSyncer) HandleBlockAnnounceHandshake(arg0 peer.ID, arg1 *network.BlockAnnounceHandshake) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleBlockAnnounceHandshake", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleBlockAnnounceHandshake indicates an expected call of HandleBlockAnnounceHandshake.
func (mr *MockSyncerMockRecorder) HandleBlockAnnounceHandshake(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleBlockAnnounceHandshake", reflect.TypeOf((*MockSyncer)(nil).HandleBlockAnnounceHandshake), arg0, arg1)
}

// IsSynced mocks base method.
func (m *MockSyncer) IsSynced() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSynced")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsSynced indicates an expected call of IsSynced.
func (mr *MockSyncerMockRecorder) IsSynced() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSynced", reflect.TypeOf((*MockSyncer)(nil).IsSynced))
}
