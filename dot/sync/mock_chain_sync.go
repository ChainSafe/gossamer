// Code generated by MockGen. DO NOT EDIT.
// Source: chain_sync.go

// Package sync is a generated GoMock package.
package sync

import (
	big "math/big"
	reflect "reflect"

	types "github.com/ChainSafe/gossamer/dot/types"
	common "github.com/ChainSafe/gossamer/lib/common"
	gomock "github.com/golang/mock/gomock"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// MockworkHandler is a mock of workHandler interface.
type MockworkHandler struct {
	ctrl     *gomock.Controller
	recorder *MockworkHandlerMockRecorder
}

// MockworkHandlerMockRecorder is the mock recorder for MockworkHandler.
type MockworkHandlerMockRecorder struct {
	mock *MockworkHandler
}

// NewMockworkHandler creates a new mock instance.
func NewMockworkHandler(ctrl *gomock.Controller) *MockworkHandler {
	mock := &MockworkHandler{ctrl: ctrl}
	mock.recorder = &MockworkHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockworkHandler) EXPECT() *MockworkHandlerMockRecorder {
	return m.recorder
}

// handleNewPeerState mocks base method.
func (m *MockworkHandler) handleNewPeerState(arg0 *peerState) (*worker, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "handleNewPeerState", arg0)
	ret0, _ := ret[0].(*worker)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// handleNewPeerState indicates an expected call of handleNewPeerState.
func (mr *MockworkHandlerMockRecorder) handleNewPeerState(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "handleNewPeerState", reflect.TypeOf((*MockworkHandler)(nil).handleNewPeerState), arg0)
}

// handleTick mocks base method.
func (m *MockworkHandler) handleTick() ([]*worker, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "handleTick")
	ret0, _ := ret[0].([]*worker)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// handleTick indicates an expected call of handleTick.
func (mr *MockworkHandlerMockRecorder) handleTick() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "handleTick", reflect.TypeOf((*MockworkHandler)(nil).handleTick))
}

// handleWorkerResult mocks base method.
func (m *MockworkHandler) handleWorkerResult(w *worker) (*worker, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "handleWorkerResult", w)
	ret0, _ := ret[0].(*worker)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// handleWorkerResult indicates an expected call of handleWorkerResult.
func (mr *MockworkHandlerMockRecorder) handleWorkerResult(w interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "handleWorkerResult", reflect.TypeOf((*MockworkHandler)(nil).handleWorkerResult), w)
}

// hasCurrentWorker mocks base method.
func (m *MockworkHandler) hasCurrentWorker(arg0 *worker, arg1 map[uint64]*worker) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "hasCurrentWorker", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// hasCurrentWorker indicates an expected call of hasCurrentWorker.
func (mr *MockworkHandlerMockRecorder) hasCurrentWorker(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "hasCurrentWorker", reflect.TypeOf((*MockworkHandler)(nil).hasCurrentWorker), arg0, arg1)
}

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
func (m *MockChainSync) getHighestBlock() (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getHighestBlock")
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// getHighestBlock indicates an expected call of getHighestBlock.
func (mr *MockChainSyncMockRecorder) getHighestBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getHighestBlock", reflect.TypeOf((*MockChainSync)(nil).getHighestBlock))
}

// setBlockAnnounce mocks base method.
func (m *MockChainSync) setBlockAnnounce(from peer.ID, header *types.Header) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "setBlockAnnounce", from, header)
	ret0, _ := ret[0].(error)
	return ret0
}

// setBlockAnnounce indicates an expected call of setBlockAnnounce.
func (mr *MockChainSyncMockRecorder) setBlockAnnounce(from, header interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "setBlockAnnounce", reflect.TypeOf((*MockChainSync)(nil).setBlockAnnounce), from, header)
}

// setPeerHead mocks base method.
func (m *MockChainSync) setPeerHead(p peer.ID, hash common.Hash, number *big.Int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "setPeerHead", p, hash, number)
	ret0, _ := ret[0].(error)
	return ret0
}

// setPeerHead indicates an expected call of setPeerHead.
func (mr *MockChainSyncMockRecorder) setPeerHead(p, hash, number interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "setPeerHead", reflect.TypeOf((*MockChainSync)(nil).setPeerHead), p, hash, number)
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
func (m *MockChainSync) stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "stop")
}

// stop indicates an expected call of stop.
func (mr *MockChainSyncMockRecorder) stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "stop", reflect.TypeOf((*MockChainSync)(nil).stop))
}

// syncState mocks base method.
func (m *MockChainSync) syncState() chainSyncState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncState")
	ret0, _ := ret[0].(chainSyncState)
	return ret0
}

// syncState indicates an expected call of syncState.
func (mr *MockChainSyncMockRecorder) syncState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncState", reflect.TypeOf((*MockChainSync)(nil).syncState))
}
