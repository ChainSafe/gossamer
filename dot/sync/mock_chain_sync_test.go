// Code generated by MockGen. DO NOT EDIT.
// Source: chain_sync.go
//
// Generated by this command:
//
//	mockgen -destination=mock_chain_sync_test.go -package sync -source chain_sync.go . ChainSync
//

// Package sync is a generated GoMock package.
package sync

import (
	reflect "reflect"

	common "github.com/ChainSafe/gossamer/lib/common"
	peer "github.com/libp2p/go-libp2p/core/peer"
	gomock "go.uber.org/mock/gomock"
)

// MockChainSync is a mock of ChainSync interface.
type MockChainSync struct {
	ctrl     *gomock.Controller
	recorder *MockChainSyncMockRecorder
}

// MockChainSyncMockRecorder is the mock recorder for MockChainSync.
type MockChainSyncMockRecorder struct {
	mock *MockChainSync
}

// NewMockChainSync creates a new mock instance.
func NewMockChainSync(ctrl *gomock.Controller) *MockChainSync {
	mock := &MockChainSync{ctrl: ctrl}
	mock.recorder = &MockChainSyncMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChainSync) EXPECT() *MockChainSyncMockRecorder {
	return m.recorder
}

// getHighestBlock mocks base method.
func (m *MockChainSync) getHighestBlock() (uint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getHighestBlock")
	ret0, _ := ret[0].(uint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// getHighestBlock indicates an expected call of getHighestBlock.
func (mr *MockChainSyncMockRecorder) getHighestBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getHighestBlock", reflect.TypeOf((*MockChainSync)(nil).getHighestBlock))
}

// getSyncMode mocks base method.
func (m *MockChainSync) getSyncMode() ChainSyncState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getSyncMode")
	ret0, _ := ret[0].(ChainSyncState)
	return ret0
}

// getSyncMode indicates an expected call of getSyncMode.
func (mr *MockChainSyncMockRecorder) getSyncMode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getSyncMode", reflect.TypeOf((*MockChainSync)(nil).getSyncMode))
}

// onBlockAnnounce mocks base method.
func (m *MockChainSync) onBlockAnnounce(arg0 announcedBlock) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "onBlockAnnounce", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// onBlockAnnounce indicates an expected call of onBlockAnnounce.
func (mr *MockChainSyncMockRecorder) onBlockAnnounce(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "onBlockAnnounce", reflect.TypeOf((*MockChainSync)(nil).onBlockAnnounce), arg0)
}

// onBlockAnnounceHandshake mocks base method.
func (m *MockChainSync) onBlockAnnounceHandshake(p peer.ID, hash common.Hash, number uint) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "onBlockAnnounceHandshake", p, hash, number)
	ret0, _ := ret[0].(error)
	return ret0
}

// onBlockAnnounceHandshake indicates an expected call of onBlockAnnounceHandshake.
func (mr *MockChainSyncMockRecorder) onBlockAnnounceHandshake(p, hash, number any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "onBlockAnnounceHandshake", reflect.TypeOf((*MockChainSync)(nil).onBlockAnnounceHandshake), p, hash, number)
}

// start mocks base method.
func (m *MockChainSync) start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "start")
}

// start indicates an expected call of start.
func (mr *MockChainSyncMockRecorder) start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "start", reflect.TypeOf((*MockChainSync)(nil).start))
}

// stop mocks base method.
func (m *MockChainSync) stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// stop indicates an expected call of stop.
func (mr *MockChainSyncMockRecorder) stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "stop", reflect.TypeOf((*MockChainSync)(nil).stop))
}
