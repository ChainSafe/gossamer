// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/sync (interfaces: BlockState,StorageState,TransactionState,BabeVerifier,FinalityGadget,BlockImportHandler,Network)

// Package sync is a generated GoMock package.
package sync

import (
	reflect "reflect"

	peerset "github.com/ChainSafe/gossamer/dot/peerset"
	types "github.com/ChainSafe/gossamer/dot/types"
	common "github.com/ChainSafe/gossamer/lib/common"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	storage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	gomock "github.com/golang/mock/gomock"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

// MockBlockState is a mock of BlockState interface.
type MockBlockState struct {
	ctrl     *gomock.Controller
	recorder *MockBlockStateMockRecorder
}

// MockBlockStateMockRecorder is the mock recorder for MockBlockState.
type MockBlockStateMockRecorder struct {
	mock *MockBlockState
}

// NewMockBlockState creates a new mock instance.
func NewMockBlockState(ctrl *gomock.Controller) *MockBlockState {
	mock := &MockBlockState{ctrl: ctrl}
	mock.recorder = &MockBlockStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBlockState) EXPECT() *MockBlockStateMockRecorder {
	return m.recorder
}

// BestBlockHeader mocks base method.
func (m *MockBlockState) BestBlockHeader() (*types.Header, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BestBlockHeader")
	ret0, _ := ret[0].(*types.Header)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BestBlockHeader indicates an expected call of BestBlockHeader.
func (mr *MockBlockStateMockRecorder) BestBlockHeader() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BestBlockHeader", reflect.TypeOf((*MockBlockState)(nil).BestBlockHeader))
}

// BestBlockNumber mocks base method.
func (m *MockBlockState) BestBlockNumber() (uint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BestBlockNumber")
	ret0, _ := ret[0].(uint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BestBlockNumber indicates an expected call of BestBlockNumber.
func (mr *MockBlockStateMockRecorder) BestBlockNumber() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BestBlockNumber", reflect.TypeOf((*MockBlockState)(nil).BestBlockNumber))
}

// CompareAndSetBlockData mocks base method.
func (m *MockBlockState) CompareAndSetBlockData(arg0 *types.BlockData) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CompareAndSetBlockData", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CompareAndSetBlockData indicates an expected call of CompareAndSetBlockData.
func (mr *MockBlockStateMockRecorder) CompareAndSetBlockData(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CompareAndSetBlockData", reflect.TypeOf((*MockBlockState)(nil).CompareAndSetBlockData), arg0)
}

// GetAllBlocksAtNumber mocks base method.
func (m *MockBlockState) GetAllBlocksAtNumber(arg0 uint) ([]common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllBlocksAtNumber", arg0)
	ret0, _ := ret[0].([]common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllBlocksAtNumber indicates an expected call of GetAllBlocksAtNumber.
func (mr *MockBlockStateMockRecorder) GetAllBlocksAtNumber(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllBlocksAtNumber", reflect.TypeOf((*MockBlockState)(nil).GetAllBlocksAtNumber), arg0)
}

// GetBlockBody mocks base method.
func (m *MockBlockState) GetBlockBody(arg0 common.Hash) (*types.Body, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockBody", arg0)
	ret0, _ := ret[0].(*types.Body)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockBody indicates an expected call of GetBlockBody.
func (mr *MockBlockStateMockRecorder) GetBlockBody(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockBody", reflect.TypeOf((*MockBlockState)(nil).GetBlockBody), arg0)
}

// GetBlockByHash mocks base method.
func (m *MockBlockState) GetBlockByHash(arg0 common.Hash) (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockByHash", arg0)
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockByHash indicates an expected call of GetBlockByHash.
func (mr *MockBlockStateMockRecorder) GetBlockByHash(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByHash", reflect.TypeOf((*MockBlockState)(nil).GetBlockByHash), arg0)
}

// GetFinalisedNotifierChannel mocks base method.
func (m *MockBlockState) GetFinalisedNotifierChannel() chan *types.FinalisationInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFinalisedNotifierChannel")
	ret0, _ := ret[0].(chan *types.FinalisationInfo)
	return ret0
}

// GetFinalisedNotifierChannel indicates an expected call of GetFinalisedNotifierChannel.
func (mr *MockBlockStateMockRecorder) GetFinalisedNotifierChannel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFinalisedNotifierChannel", reflect.TypeOf((*MockBlockState)(nil).GetFinalisedNotifierChannel))
}

// GetHashByNumber mocks base method.
func (m *MockBlockState) GetHashByNumber(arg0 uint) (common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHashByNumber", arg0)
	ret0, _ := ret[0].(common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHashByNumber indicates an expected call of GetHashByNumber.
func (mr *MockBlockStateMockRecorder) GetHashByNumber(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHashByNumber", reflect.TypeOf((*MockBlockState)(nil).GetHashByNumber), arg0)
}

// GetHeader mocks base method.
func (m *MockBlockState) GetHeader(arg0 common.Hash) (*types.Header, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHeader", arg0)
	ret0, _ := ret[0].(*types.Header)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHeader indicates an expected call of GetHeader.
func (mr *MockBlockStateMockRecorder) GetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHeader", reflect.TypeOf((*MockBlockState)(nil).GetHeader), arg0)
}

// GetHeaderByNumber mocks base method.
func (m *MockBlockState) GetHeaderByNumber(arg0 uint) (*types.Header, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHeaderByNumber", arg0)
	ret0, _ := ret[0].(*types.Header)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHeaderByNumber indicates an expected call of GetHeaderByNumber.
func (mr *MockBlockStateMockRecorder) GetHeaderByNumber(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHeaderByNumber", reflect.TypeOf((*MockBlockState)(nil).GetHeaderByNumber), arg0)
}

// GetHighestFinalisedHeader mocks base method.
func (m *MockBlockState) GetHighestFinalisedHeader() (*types.Header, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHighestFinalisedHeader")
	ret0, _ := ret[0].(*types.Header)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHighestFinalisedHeader indicates an expected call of GetHighestFinalisedHeader.
func (mr *MockBlockStateMockRecorder) GetHighestFinalisedHeader() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHighestFinalisedHeader", reflect.TypeOf((*MockBlockState)(nil).GetHighestFinalisedHeader))
}

// GetJustification mocks base method.
func (m *MockBlockState) GetJustification(arg0 common.Hash) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetJustification", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetJustification indicates an expected call of GetJustification.
func (mr *MockBlockStateMockRecorder) GetJustification(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetJustification", reflect.TypeOf((*MockBlockState)(nil).GetJustification), arg0)
}

// GetMessageQueue mocks base method.
func (m *MockBlockState) GetMessageQueue(arg0 common.Hash) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMessageQueue", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMessageQueue indicates an expected call of GetMessageQueue.
func (mr *MockBlockStateMockRecorder) GetMessageQueue(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMessageQueue", reflect.TypeOf((*MockBlockState)(nil).GetMessageQueue), arg0)
}

// GetReceipt mocks base method.
func (m *MockBlockState) GetReceipt(arg0 common.Hash) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetReceipt", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetReceipt indicates an expected call of GetReceipt.
func (mr *MockBlockStateMockRecorder) GetReceipt(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetReceipt", reflect.TypeOf((*MockBlockState)(nil).GetReceipt), arg0)
}

// GetRuntime mocks base method.
func (m *MockBlockState) GetRuntime(arg0 common.Hash) (runtime.Instance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRuntime", arg0)
	ret0, _ := ret[0].(runtime.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRuntime indicates an expected call of GetRuntime.
func (mr *MockBlockStateMockRecorder) GetRuntime(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRuntime", reflect.TypeOf((*MockBlockState)(nil).GetRuntime), arg0)
}

// HasHeader mocks base method.
func (m *MockBlockState) HasHeader(arg0 common.Hash) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasHeader", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HasHeader indicates an expected call of HasHeader.
func (mr *MockBlockStateMockRecorder) HasHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasHeader", reflect.TypeOf((*MockBlockState)(nil).HasHeader), arg0)
}

// IsDescendantOf mocks base method.
func (m *MockBlockState) IsDescendantOf(arg0, arg1 common.Hash) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsDescendantOf", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsDescendantOf indicates an expected call of IsDescendantOf.
func (mr *MockBlockStateMockRecorder) IsDescendantOf(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsDescendantOf", reflect.TypeOf((*MockBlockState)(nil).IsDescendantOf), arg0, arg1)
}

// Range mocks base method.
func (m *MockBlockState) Range(arg0, arg1 common.Hash) ([]common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Range", arg0, arg1)
	ret0, _ := ret[0].([]common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Range indicates an expected call of Range.
func (mr *MockBlockStateMockRecorder) Range(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Range", reflect.TypeOf((*MockBlockState)(nil).Range), arg0, arg1)
}

// RangeInMemory mocks base method.
func (m *MockBlockState) RangeInMemory(arg0, arg1 common.Hash) ([]common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RangeInMemory", arg0, arg1)
	ret0, _ := ret[0].([]common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RangeInMemory indicates an expected call of RangeInMemory.
func (mr *MockBlockStateMockRecorder) RangeInMemory(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RangeInMemory", reflect.TypeOf((*MockBlockState)(nil).RangeInMemory), arg0, arg1)
}

// SetJustification mocks base method.
func (m *MockBlockState) SetJustification(arg0 common.Hash, arg1 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetJustification", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetJustification indicates an expected call of SetJustification.
func (mr *MockBlockStateMockRecorder) SetJustification(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetJustification", reflect.TypeOf((*MockBlockState)(nil).SetJustification), arg0, arg1)
}

// StoreRuntime mocks base method.
func (m *MockBlockState) StoreRuntime(arg0 common.Hash, arg1 runtime.Instance) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "StoreRuntime", arg0, arg1)
}

// StoreRuntime indicates an expected call of StoreRuntime.
func (mr *MockBlockStateMockRecorder) StoreRuntime(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreRuntime", reflect.TypeOf((*MockBlockState)(nil).StoreRuntime), arg0, arg1)
}

// MockStorageState is a mock of StorageState interface.
type MockStorageState struct {
	ctrl     *gomock.Controller
	recorder *MockStorageStateMockRecorder
}

// MockStorageStateMockRecorder is the mock recorder for MockStorageState.
type MockStorageStateMockRecorder struct {
	mock *MockStorageState
}

// NewMockStorageState creates a new mock instance.
func NewMockStorageState(ctrl *gomock.Controller) *MockStorageState {
	mock := &MockStorageState{ctrl: ctrl}
	mock.recorder = &MockStorageStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageState) EXPECT() *MockStorageStateMockRecorder {
	return m.recorder
}

// Lock mocks base method.
func (m *MockStorageState) Lock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Lock")
}

// Lock indicates an expected call of Lock.
func (mr *MockStorageStateMockRecorder) Lock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Lock", reflect.TypeOf((*MockStorageState)(nil).Lock))
}

// TrieState mocks base method.
func (m *MockStorageState) TrieState(arg0 *common.Hash) (*storage.TrieState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TrieState", arg0)
	ret0, _ := ret[0].(*storage.TrieState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TrieState indicates an expected call of TrieState.
func (mr *MockStorageStateMockRecorder) TrieState(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TrieState", reflect.TypeOf((*MockStorageState)(nil).TrieState), arg0)
}

// Unlock mocks base method.
func (m *MockStorageState) Unlock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Unlock")
}

// Unlock indicates an expected call of Unlock.
func (mr *MockStorageStateMockRecorder) Unlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unlock", reflect.TypeOf((*MockStorageState)(nil).Unlock))
}

// MockTransactionState is a mock of TransactionState interface.
type MockTransactionState struct {
	ctrl     *gomock.Controller
	recorder *MockTransactionStateMockRecorder
}

// MockTransactionStateMockRecorder is the mock recorder for MockTransactionState.
type MockTransactionStateMockRecorder struct {
	mock *MockTransactionState
}

// NewMockTransactionState creates a new mock instance.
func NewMockTransactionState(ctrl *gomock.Controller) *MockTransactionState {
	mock := &MockTransactionState{ctrl: ctrl}
	mock.recorder = &MockTransactionStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTransactionState) EXPECT() *MockTransactionStateMockRecorder {
	return m.recorder
}

// RemoveExtrinsic mocks base method.
func (m *MockTransactionState) RemoveExtrinsic(arg0 types.Extrinsic) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RemoveExtrinsic", arg0)
}

// RemoveExtrinsic indicates an expected call of RemoveExtrinsic.
func (mr *MockTransactionStateMockRecorder) RemoveExtrinsic(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveExtrinsic", reflect.TypeOf((*MockTransactionState)(nil).RemoveExtrinsic), arg0)
}

// MockBabeVerifier is a mock of BabeVerifier interface.
type MockBabeVerifier struct {
	ctrl     *gomock.Controller
	recorder *MockBabeVerifierMockRecorder
}

// MockBabeVerifierMockRecorder is the mock recorder for MockBabeVerifier.
type MockBabeVerifierMockRecorder struct {
	mock *MockBabeVerifier
}

// NewMockBabeVerifier creates a new mock instance.
func NewMockBabeVerifier(ctrl *gomock.Controller) *MockBabeVerifier {
	mock := &MockBabeVerifier{ctrl: ctrl}
	mock.recorder = &MockBabeVerifierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBabeVerifier) EXPECT() *MockBabeVerifierMockRecorder {
	return m.recorder
}

// VerifyBlock mocks base method.
func (m *MockBabeVerifier) VerifyBlock(arg0 *types.Header) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyBlock", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// VerifyBlock indicates an expected call of VerifyBlock.
func (mr *MockBabeVerifierMockRecorder) VerifyBlock(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyBlock", reflect.TypeOf((*MockBabeVerifier)(nil).VerifyBlock), arg0)
}

// MockFinalityGadget is a mock of FinalityGadget interface.
type MockFinalityGadget struct {
	ctrl     *gomock.Controller
	recorder *MockFinalityGadgetMockRecorder
}

// MockFinalityGadgetMockRecorder is the mock recorder for MockFinalityGadget.
type MockFinalityGadgetMockRecorder struct {
	mock *MockFinalityGadget
}

// NewMockFinalityGadget creates a new mock instance.
func NewMockFinalityGadget(ctrl *gomock.Controller) *MockFinalityGadget {
	mock := &MockFinalityGadget{ctrl: ctrl}
	mock.recorder = &MockFinalityGadgetMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFinalityGadget) EXPECT() *MockFinalityGadgetMockRecorder {
	return m.recorder
}

// VerifyBlockJustification mocks base method.
func (m *MockFinalityGadget) VerifyBlockJustification(arg0 common.Hash, arg1 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyBlockJustification", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// VerifyBlockJustification indicates an expected call of VerifyBlockJustification.
func (mr *MockFinalityGadgetMockRecorder) VerifyBlockJustification(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyBlockJustification", reflect.TypeOf((*MockFinalityGadget)(nil).VerifyBlockJustification), arg0, arg1)
}

// MockBlockImportHandler is a mock of BlockImportHandler interface.
type MockBlockImportHandler struct {
	ctrl     *gomock.Controller
	recorder *MockBlockImportHandlerMockRecorder
}

// MockBlockImportHandlerMockRecorder is the mock recorder for MockBlockImportHandler.
type MockBlockImportHandlerMockRecorder struct {
	mock *MockBlockImportHandler
}

// NewMockBlockImportHandler creates a new mock instance.
func NewMockBlockImportHandler(ctrl *gomock.Controller) *MockBlockImportHandler {
	mock := &MockBlockImportHandler{ctrl: ctrl}
	mock.recorder = &MockBlockImportHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBlockImportHandler) EXPECT() *MockBlockImportHandlerMockRecorder {
	return m.recorder
}

// HandleBlockImport mocks base method.
func (m *MockBlockImportHandler) HandleBlockImport(arg0 *types.Block, arg1 *storage.TrieState, arg2 bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleBlockImport", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleBlockImport indicates an expected call of HandleBlockImport.
func (mr *MockBlockImportHandlerMockRecorder) HandleBlockImport(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleBlockImport", reflect.TypeOf((*MockBlockImportHandler)(nil).HandleBlockImport), arg0, arg1, arg2)
}

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

// AllConnectedPeersIDs mocks base method.
func (m *MockNetwork) AllConnectedPeersIDs() []peer.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AllConnectedPeersIDs")
	ret0, _ := ret[0].([]peer.ID)
	return ret0
}

// AllConnectedPeersIDs indicates an expected call of AllConnectedPeersIDs.
func (mr *MockNetworkMockRecorder) AllConnectedPeersIDs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllConnectedPeersIDs", reflect.TypeOf((*MockNetwork)(nil).AllConnectedPeersIDs))
}

// Peers mocks base method.
func (m *MockNetwork) Peers() []common.PeerInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Peers")
	ret0, _ := ret[0].([]common.PeerInfo)
	return ret0
}

// Peers indicates an expected call of Peers.
func (mr *MockNetworkMockRecorder) Peers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Peers", reflect.TypeOf((*MockNetwork)(nil).Peers))
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
