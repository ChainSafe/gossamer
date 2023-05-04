// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/core (interfaces: BlockState,StorageState,TransactionState,Network,CodeSubstitutedState,Telemetry)

// Package core is a generated GoMock package.
package core

import (
	json "encoding/json"
	reflect "reflect"

	network "github.com/ChainSafe/gossamer/dot/network"
	peerset "github.com/ChainSafe/gossamer/dot/peerset"
	runtimeinterface "github.com/ChainSafe/gossamer/dot/runtimeinterface"
	types "github.com/ChainSafe/gossamer/dot/types"
	common "github.com/ChainSafe/gossamer/lib/common"
	storage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	transaction "github.com/ChainSafe/gossamer/lib/transaction"
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

// AddBlock mocks base method.
func (m *MockBlockState) AddBlock(arg0 *types.Block) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddBlock", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddBlock indicates an expected call of AddBlock.
func (mr *MockBlockStateMockRecorder) AddBlock(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddBlock", reflect.TypeOf((*MockBlockState)(nil).AddBlock), arg0)
}

// BestBlockHash mocks base method.
func (m *MockBlockState) BestBlockHash() common.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BestBlockHash")
	ret0, _ := ret[0].(common.Hash)
	return ret0
}

// BestBlockHash indicates an expected call of BestBlockHash.
func (mr *MockBlockStateMockRecorder) BestBlockHash() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BestBlockHash", reflect.TypeOf((*MockBlockState)(nil).BestBlockHash))
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

// GetBlockStateRoot mocks base method.
func (m *MockBlockState) GetBlockStateRoot(arg0 common.Hash) (common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockStateRoot", arg0)
	ret0, _ := ret[0].(common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockStateRoot indicates an expected call of GetBlockStateRoot.
func (mr *MockBlockStateMockRecorder) GetBlockStateRoot(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockStateRoot", reflect.TypeOf((*MockBlockState)(nil).GetBlockStateRoot), arg0)
}

// GetRuntime mocks base method.
func (m *MockBlockState) GetRuntime(arg0 common.Hash) (runtimeinterface.Instance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRuntime", arg0)
	ret0, _ := ret[0].(runtimeinterface.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRuntime indicates an expected call of GetRuntime.
func (mr *MockBlockStateMockRecorder) GetRuntime(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRuntime", reflect.TypeOf((*MockBlockState)(nil).GetRuntime), arg0)
}

// HandleRuntimeChanges mocks base method.
func (m *MockBlockState) HandleRuntimeChanges(arg0 *storage.TrieState, arg1 runtimeinterface.Instance, arg2 common.Hash) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleRuntimeChanges", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleRuntimeChanges indicates an expected call of HandleRuntimeChanges.
func (mr *MockBlockStateMockRecorder) HandleRuntimeChanges(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleRuntimeChanges", reflect.TypeOf((*MockBlockState)(nil).HandleRuntimeChanges), arg0, arg1, arg2)
}

// LowestCommonAncestor mocks base method.
func (m *MockBlockState) LowestCommonAncestor(arg0, arg1 common.Hash) (common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LowestCommonAncestor", arg0, arg1)
	ret0, _ := ret[0].(common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LowestCommonAncestor indicates an expected call of LowestCommonAncestor.
func (mr *MockBlockStateMockRecorder) LowestCommonAncestor(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LowestCommonAncestor", reflect.TypeOf((*MockBlockState)(nil).LowestCommonAncestor), arg0, arg1)
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

// StoreRuntime mocks base method.
func (m *MockBlockState) StoreRuntime(arg0 common.Hash, arg1 runtimeinterface.Instance) {
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

// GenerateTrieProof mocks base method.
func (m *MockStorageState) GenerateTrieProof(arg0 common.Hash, arg1 [][]byte) ([][]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateTrieProof", arg0, arg1)
	ret0, _ := ret[0].([][]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateTrieProof indicates an expected call of GenerateTrieProof.
func (mr *MockStorageStateMockRecorder) GenerateTrieProof(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateTrieProof", reflect.TypeOf((*MockStorageState)(nil).GenerateTrieProof), arg0, arg1)
}

// GetStateRootFromBlock mocks base method.
func (m *MockStorageState) GetStateRootFromBlock(arg0 *common.Hash) (*common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStateRootFromBlock", arg0)
	ret0, _ := ret[0].(*common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStateRootFromBlock indicates an expected call of GetStateRootFromBlock.
func (mr *MockStorageStateMockRecorder) GetStateRootFromBlock(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStateRootFromBlock", reflect.TypeOf((*MockStorageState)(nil).GetStateRootFromBlock), arg0)
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

// StoreTrie mocks base method.
func (m *MockStorageState) StoreTrie(arg0 *storage.TrieState, arg1 *types.Header) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreTrie", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreTrie indicates an expected call of StoreTrie.
func (mr *MockStorageStateMockRecorder) StoreTrie(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreTrie", reflect.TypeOf((*MockStorageState)(nil).StoreTrie), arg0, arg1)
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

// AddToPool mocks base method.
func (m *MockTransactionState) AddToPool(arg0 *transaction.ValidTransaction) common.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddToPool", arg0)
	ret0, _ := ret[0].(common.Hash)
	return ret0
}

// AddToPool indicates an expected call of AddToPool.
func (mr *MockTransactionStateMockRecorder) AddToPool(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddToPool", reflect.TypeOf((*MockTransactionState)(nil).AddToPool), arg0)
}

// Exists mocks base method.
func (m *MockTransactionState) Exists(arg0 types.Extrinsic) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Exists indicates an expected call of Exists.
func (mr *MockTransactionStateMockRecorder) Exists(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockTransactionState)(nil).Exists), arg0)
}

// PendingInPool mocks base method.
func (m *MockTransactionState) PendingInPool() []*transaction.ValidTransaction {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PendingInPool")
	ret0, _ := ret[0].([]*transaction.ValidTransaction)
	return ret0
}

// PendingInPool indicates an expected call of PendingInPool.
func (mr *MockTransactionStateMockRecorder) PendingInPool() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PendingInPool", reflect.TypeOf((*MockTransactionState)(nil).PendingInPool))
}

// Push mocks base method.
func (m *MockTransactionState) Push(arg0 *transaction.ValidTransaction) (common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Push", arg0)
	ret0, _ := ret[0].(common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Push indicates an expected call of Push.
func (mr *MockTransactionStateMockRecorder) Push(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Push", reflect.TypeOf((*MockTransactionState)(nil).Push), arg0)
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

// RemoveExtrinsicFromPool mocks base method.
func (m *MockTransactionState) RemoveExtrinsicFromPool(arg0 types.Extrinsic) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RemoveExtrinsicFromPool", arg0)
}

// RemoveExtrinsicFromPool indicates an expected call of RemoveExtrinsicFromPool.
func (mr *MockTransactionStateMockRecorder) RemoveExtrinsicFromPool(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveExtrinsicFromPool", reflect.TypeOf((*MockTransactionState)(nil).RemoveExtrinsicFromPool), arg0)
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

// IsSynced mocks base method.
func (m *MockNetwork) IsSynced() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSynced")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsSynced indicates an expected call of IsSynced.
func (mr *MockNetworkMockRecorder) IsSynced() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSynced", reflect.TypeOf((*MockNetwork)(nil).IsSynced))
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

// MockCodeSubstitutedState is a mock of CodeSubstitutedState interface.
type MockCodeSubstitutedState struct {
	ctrl     *gomock.Controller
	recorder *MockCodeSubstitutedStateMockRecorder
}

// MockCodeSubstitutedStateMockRecorder is the mock recorder for MockCodeSubstitutedState.
type MockCodeSubstitutedStateMockRecorder struct {
	mock *MockCodeSubstitutedState
}

// NewMockCodeSubstitutedState creates a new mock instance.
func NewMockCodeSubstitutedState(ctrl *gomock.Controller) *MockCodeSubstitutedState {
	mock := &MockCodeSubstitutedState{ctrl: ctrl}
	mock.recorder = &MockCodeSubstitutedStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCodeSubstitutedState) EXPECT() *MockCodeSubstitutedStateMockRecorder {
	return m.recorder
}

// StoreCodeSubstitutedBlockHash mocks base method.
func (m *MockCodeSubstitutedState) StoreCodeSubstitutedBlockHash(arg0 common.Hash) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreCodeSubstitutedBlockHash", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreCodeSubstitutedBlockHash indicates an expected call of StoreCodeSubstitutedBlockHash.
func (mr *MockCodeSubstitutedStateMockRecorder) StoreCodeSubstitutedBlockHash(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreCodeSubstitutedBlockHash", reflect.TypeOf((*MockCodeSubstitutedState)(nil).StoreCodeSubstitutedBlockHash), arg0)
}

// MockTelemetry is a mock of Telemetry interface.
type MockTelemetry struct {
	ctrl     *gomock.Controller
	recorder *MockTelemetryMockRecorder
}

// MockTelemetryMockRecorder is the mock recorder for MockTelemetry.
type MockTelemetryMockRecorder struct {
	mock *MockTelemetry
}

// NewMockTelemetry creates a new mock instance.
func NewMockTelemetry(ctrl *gomock.Controller) *MockTelemetry {
	mock := &MockTelemetry{ctrl: ctrl}
	mock.recorder = &MockTelemetryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTelemetry) EXPECT() *MockTelemetryMockRecorder {
	return m.recorder
}

// SendMessage mocks base method.
func (m *MockTelemetry) SendMessage(arg0 json.Marshaler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendMessage", arg0)
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockTelemetryMockRecorder) SendMessage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockTelemetry)(nil).SendMessage), arg0)
}
