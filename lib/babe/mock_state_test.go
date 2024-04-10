// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/lib/babe (interfaces: BlockState,ImportedBlockNotifierManager,StorageState,TransactionState,EpochState,BlockImportHandler,SlotState)
//
// Generated by this command:
//
//	mockgen -destination=mock_state_test.go -package babe . BlockState,ImportedBlockNotifierManager,StorageState,TransactionState,EpochState,BlockImportHandler,SlotState
//

// Package babe is a generated GoMock package.
package babe

import (
	reflect "reflect"
	time "time"

	types "github.com/ChainSafe/gossamer/dot/types"
	common "github.com/ChainSafe/gossamer/lib/common"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	storage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	transaction "github.com/ChainSafe/gossamer/lib/transaction"
	gomock "go.uber.org/mock/gomock"
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
func (mr *MockBlockStateMockRecorder) AddBlock(arg0 any) *gomock.Call {
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

// FreeImportedBlockNotifierChannel mocks base method.
func (m *MockBlockState) FreeImportedBlockNotifierChannel(arg0 chan *types.Block) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "FreeImportedBlockNotifierChannel", arg0)
}

// FreeImportedBlockNotifierChannel indicates an expected call of FreeImportedBlockNotifierChannel.
func (mr *MockBlockStateMockRecorder) FreeImportedBlockNotifierChannel(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeImportedBlockNotifierChannel", reflect.TypeOf((*MockBlockState)(nil).FreeImportedBlockNotifierChannel), arg0)
}

// GenesisHash mocks base method.
func (m *MockBlockState) GenesisHash() common.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenesisHash")
	ret0, _ := ret[0].(common.Hash)
	return ret0
}

// GenesisHash indicates an expected call of GenesisHash.
func (mr *MockBlockStateMockRecorder) GenesisHash() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenesisHash", reflect.TypeOf((*MockBlockState)(nil).GenesisHash))
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
func (mr *MockBlockStateMockRecorder) GetBlockByHash(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByHash", reflect.TypeOf((*MockBlockState)(nil).GetBlockByHash), arg0)
}

// GetBlockByNumber mocks base method.
func (m *MockBlockState) GetBlockByNumber(arg0 uint) (*types.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockByNumber", arg0)
	ret0, _ := ret[0].(*types.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockByNumber indicates an expected call of GetBlockByNumber.
func (mr *MockBlockStateMockRecorder) GetBlockByNumber(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockByNumber", reflect.TypeOf((*MockBlockState)(nil).GetBlockByNumber), arg0)
}

// GetBlockHashesBySlot mocks base method.
func (m *MockBlockState) GetBlockHashesBySlot(arg0 uint64) ([]common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockHashesBySlot", arg0)
	ret0, _ := ret[0].([]common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockHashesBySlot indicates an expected call of GetBlockHashesBySlot.
func (mr *MockBlockStateMockRecorder) GetBlockHashesBySlot(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockHashesBySlot", reflect.TypeOf((*MockBlockState)(nil).GetBlockHashesBySlot), arg0)
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
func (mr *MockBlockStateMockRecorder) GetHeader(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHeader", reflect.TypeOf((*MockBlockState)(nil).GetHeader), arg0)
}

// GetImportedBlockNotifierChannel mocks base method.
func (m *MockBlockState) GetImportedBlockNotifierChannel() chan *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImportedBlockNotifierChannel")
	ret0, _ := ret[0].(chan *types.Block)
	return ret0
}

// GetImportedBlockNotifierChannel indicates an expected call of GetImportedBlockNotifierChannel.
func (mr *MockBlockStateMockRecorder) GetImportedBlockNotifierChannel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImportedBlockNotifierChannel", reflect.TypeOf((*MockBlockState)(nil).GetImportedBlockNotifierChannel))
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
func (mr *MockBlockStateMockRecorder) GetRuntime(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRuntime", reflect.TypeOf((*MockBlockState)(nil).GetRuntime), arg0)
}

// GetSlotForBlock mocks base method.
func (m *MockBlockState) GetSlotForBlock(arg0 common.Hash) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSlotForBlock", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSlotForBlock indicates an expected call of GetSlotForBlock.
func (mr *MockBlockStateMockRecorder) GetSlotForBlock(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSlotForBlock", reflect.TypeOf((*MockBlockState)(nil).GetSlotForBlock), arg0)
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
func (mr *MockBlockStateMockRecorder) IsDescendantOf(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsDescendantOf", reflect.TypeOf((*MockBlockState)(nil).IsDescendantOf), arg0, arg1)
}

// NumberIsFinalised mocks base method.
func (m *MockBlockState) NumberIsFinalised(arg0 uint) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NumberIsFinalised", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NumberIsFinalised indicates an expected call of NumberIsFinalised.
func (mr *MockBlockStateMockRecorder) NumberIsFinalised(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NumberIsFinalised", reflect.TypeOf((*MockBlockState)(nil).NumberIsFinalised), arg0)
}

// StoreRuntime mocks base method.
func (m *MockBlockState) StoreRuntime(arg0 common.Hash, arg1 runtime.Instance) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "StoreRuntime", arg0, arg1)
}

// StoreRuntime indicates an expected call of StoreRuntime.
func (mr *MockBlockStateMockRecorder) StoreRuntime(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreRuntime", reflect.TypeOf((*MockBlockState)(nil).StoreRuntime), arg0, arg1)
}

// MockImportedBlockNotifierManager is a mock of ImportedBlockNotifierManager interface.
type MockImportedBlockNotifierManager struct {
	ctrl     *gomock.Controller
	recorder *MockImportedBlockNotifierManagerMockRecorder
}

// MockImportedBlockNotifierManagerMockRecorder is the mock recorder for MockImportedBlockNotifierManager.
type MockImportedBlockNotifierManagerMockRecorder struct {
	mock *MockImportedBlockNotifierManager
}

// NewMockImportedBlockNotifierManager creates a new mock instance.
func NewMockImportedBlockNotifierManager(ctrl *gomock.Controller) *MockImportedBlockNotifierManager {
	mock := &MockImportedBlockNotifierManager{ctrl: ctrl}
	mock.recorder = &MockImportedBlockNotifierManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockImportedBlockNotifierManager) EXPECT() *MockImportedBlockNotifierManagerMockRecorder {
	return m.recorder
}

// FreeImportedBlockNotifierChannel mocks base method.
func (m *MockImportedBlockNotifierManager) FreeImportedBlockNotifierChannel(arg0 chan *types.Block) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "FreeImportedBlockNotifierChannel", arg0)
}

// FreeImportedBlockNotifierChannel indicates an expected call of FreeImportedBlockNotifierChannel.
func (mr *MockImportedBlockNotifierManagerMockRecorder) FreeImportedBlockNotifierChannel(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeImportedBlockNotifierChannel", reflect.TypeOf((*MockImportedBlockNotifierManager)(nil).FreeImportedBlockNotifierChannel), arg0)
}

// GetImportedBlockNotifierChannel mocks base method.
func (m *MockImportedBlockNotifierManager) GetImportedBlockNotifierChannel() chan *types.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImportedBlockNotifierChannel")
	ret0, _ := ret[0].(chan *types.Block)
	return ret0
}

// GetImportedBlockNotifierChannel indicates an expected call of GetImportedBlockNotifierChannel.
func (mr *MockImportedBlockNotifierManagerMockRecorder) GetImportedBlockNotifierChannel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImportedBlockNotifierChannel", reflect.TypeOf((*MockImportedBlockNotifierManager)(nil).GetImportedBlockNotifierChannel))
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
func (mr *MockStorageStateMockRecorder) TrieState(arg0 any) *gomock.Call {
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

// PopWithTimer mocks base method.
func (m *MockTransactionState) PopWithTimer(arg0 <-chan time.Time) *transaction.ValidTransaction {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PopWithTimer", arg0)
	ret0, _ := ret[0].(*transaction.ValidTransaction)
	return ret0
}

// PopWithTimer indicates an expected call of PopWithTimer.
func (mr *MockTransactionStateMockRecorder) PopWithTimer(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PopWithTimer", reflect.TypeOf((*MockTransactionState)(nil).PopWithTimer), arg0)
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
func (mr *MockTransactionStateMockRecorder) Push(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Push", reflect.TypeOf((*MockTransactionState)(nil).Push), arg0)
}

// MockEpochState is a mock of EpochState interface.
type MockEpochState struct {
	ctrl     *gomock.Controller
	recorder *MockEpochStateMockRecorder
}

// MockEpochStateMockRecorder is the mock recorder for MockEpochState.
type MockEpochStateMockRecorder struct {
	mock *MockEpochState
}

// NewMockEpochState creates a new mock instance.
func NewMockEpochState(ctrl *gomock.Controller) *MockEpochState {
	mock := &MockEpochState{ctrl: ctrl}
	mock.recorder = &MockEpochStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEpochState) EXPECT() *MockEpochStateMockRecorder {
	return m.recorder
}

// FindSkippedEpochDataRaw mocks base method.
func (m *MockEpochState) FindSkippedEpochDataRaw(arg0, arg1 uint64, arg2 *types.Header) (*types.EpochDataRaw, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindSkippedEpochDataRaw", arg0, arg1, arg2)
	ret0, _ := ret[0].(*types.EpochDataRaw)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindSkippedEpochDataRaw indicates an expected call of FindSkippedEpochDataRaw.
func (mr *MockEpochStateMockRecorder) FindSkippedEpochDataRaw(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindSkippedEpochDataRaw", reflect.TypeOf((*MockEpochState)(nil).FindSkippedEpochDataRaw), arg0, arg1, arg2)
}

// GetConfigData mocks base method.
func (m *MockEpochState) GetConfigData(arg0 uint64, arg1 *types.Header) (*types.ConfigData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConfigData", arg0, arg1)
	ret0, _ := ret[0].(*types.ConfigData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConfigData indicates an expected call of GetConfigData.
func (mr *MockEpochStateMockRecorder) GetConfigData(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfigData", reflect.TypeOf((*MockEpochState)(nil).GetConfigData), arg0, arg1)
}

// GetCurrentEpoch mocks base method.
func (m *MockEpochState) GetCurrentEpoch() (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentEpoch")
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCurrentEpoch indicates an expected call of GetCurrentEpoch.
func (mr *MockEpochStateMockRecorder) GetCurrentEpoch() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentEpoch", reflect.TypeOf((*MockEpochState)(nil).GetCurrentEpoch))
}

// GetEpochDataRaw mocks base method.
func (m *MockEpochState) GetEpochDataRaw(arg0 uint64, arg1 *types.Header) (*types.EpochDataRaw, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpochDataRaw", arg0, arg1)
	ret0, _ := ret[0].(*types.EpochDataRaw)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEpochDataRaw indicates an expected call of GetEpochDataRaw.
func (mr *MockEpochStateMockRecorder) GetEpochDataRaw(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpochDataRaw", reflect.TypeOf((*MockEpochState)(nil).GetEpochDataRaw), arg0, arg1)
}

// GetEpochForBlock mocks base method.
func (m *MockEpochState) GetEpochForBlock(arg0 *types.Header) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpochForBlock", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEpochForBlock indicates an expected call of GetEpochForBlock.
func (mr *MockEpochStateMockRecorder) GetEpochForBlock(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpochForBlock", reflect.TypeOf((*MockEpochState)(nil).GetEpochForBlock), arg0)
}

// GetEpochFromTime mocks base method.
func (m *MockEpochState) GetEpochFromTime(arg0 time.Time, arg1 common.Hash) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpochFromTime", arg0, arg1)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEpochFromTime indicates an expected call of GetEpochFromTime.
func (mr *MockEpochStateMockRecorder) GetEpochFromTime(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpochFromTime", reflect.TypeOf((*MockEpochState)(nil).GetEpochFromTime), arg0, arg1)
}

// GetEpochLength mocks base method.
func (m *MockEpochState) GetEpochLength() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpochLength")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetEpochLength indicates an expected call of GetEpochLength.
func (mr *MockEpochStateMockRecorder) GetEpochLength() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpochLength", reflect.TypeOf((*MockEpochState)(nil).GetEpochLength))
}

// GetLatestConfigData mocks base method.
func (m *MockEpochState) GetLatestConfigData() (*types.ConfigData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestConfigData")
	ret0, _ := ret[0].(*types.ConfigData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestConfigData indicates an expected call of GetLatestConfigData.
func (mr *MockEpochStateMockRecorder) GetLatestConfigData() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestConfigData", reflect.TypeOf((*MockEpochState)(nil).GetLatestConfigData))
}

// GetLatestEpochDataRaw mocks base method.
func (m *MockEpochState) GetLatestEpochDataRaw() (*types.EpochDataRaw, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestEpochDataRaw")
	ret0, _ := ret[0].(*types.EpochDataRaw)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestEpochDataRaw indicates an expected call of GetLatestEpochDataRaw.
func (mr *MockEpochStateMockRecorder) GetLatestEpochDataRaw() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestEpochDataRaw", reflect.TypeOf((*MockEpochState)(nil).GetLatestEpochDataRaw))
}

// GetSlotDuration mocks base method.
func (m *MockEpochState) GetSlotDuration() (time.Duration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSlotDuration")
	ret0, _ := ret[0].(time.Duration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSlotDuration indicates an expected call of GetSlotDuration.
func (mr *MockEpochStateMockRecorder) GetSlotDuration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSlotDuration", reflect.TypeOf((*MockEpochState)(nil).GetSlotDuration))
}

// GetStartSlotForEpoch mocks base method.
func (m *MockEpochState) GetStartSlotForEpoch(arg0 uint64, arg1 common.Hash) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStartSlotForEpoch", arg0, arg1)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStartSlotForEpoch indicates an expected call of GetStartSlotForEpoch.
func (mr *MockEpochStateMockRecorder) GetStartSlotForEpoch(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStartSlotForEpoch", reflect.TypeOf((*MockEpochState)(nil).GetStartSlotForEpoch), arg0, arg1)
}

// SkipVerify mocks base method.
func (m *MockEpochState) SkipVerify(arg0 *types.Header) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SkipVerify", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SkipVerify indicates an expected call of SkipVerify.
func (mr *MockEpochStateMockRecorder) SkipVerify(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SkipVerify", reflect.TypeOf((*MockEpochState)(nil).SkipVerify), arg0)
}

// StoreCurrentEpoch mocks base method.
func (m *MockEpochState) StoreCurrentEpoch(arg0 uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreCurrentEpoch", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreCurrentEpoch indicates an expected call of StoreCurrentEpoch.
func (mr *MockEpochStateMockRecorder) StoreCurrentEpoch(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreCurrentEpoch", reflect.TypeOf((*MockEpochState)(nil).StoreCurrentEpoch), arg0)
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

// HandleBlockProduced mocks base method.
func (m *MockBlockImportHandler) HandleBlockProduced(arg0 *types.Block, arg1 *storage.TrieState) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleBlockProduced", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleBlockProduced indicates an expected call of HandleBlockProduced.
func (mr *MockBlockImportHandlerMockRecorder) HandleBlockProduced(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleBlockProduced", reflect.TypeOf((*MockBlockImportHandler)(nil).HandleBlockProduced), arg0, arg1)
}

// MockSlotState is a mock of SlotState interface.
type MockSlotState struct {
	ctrl     *gomock.Controller
	recorder *MockSlotStateMockRecorder
}

// MockSlotStateMockRecorder is the mock recorder for MockSlotState.
type MockSlotStateMockRecorder struct {
	mock *MockSlotState
}

// NewMockSlotState creates a new mock instance.
func NewMockSlotState(ctrl *gomock.Controller) *MockSlotState {
	mock := &MockSlotState{ctrl: ctrl}
	mock.recorder = &MockSlotStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSlotState) EXPECT() *MockSlotStateMockRecorder {
	return m.recorder
}

// CheckEquivocation mocks base method.
func (m *MockSlotState) CheckEquivocation(arg0, arg1 uint64, arg2 *types.Header, arg3 types.AuthorityID) (*types.BabeEquivocationProof, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckEquivocation", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*types.BabeEquivocationProof)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckEquivocation indicates an expected call of CheckEquivocation.
func (mr *MockSlotStateMockRecorder) CheckEquivocation(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckEquivocation", reflect.TypeOf((*MockSlotState)(nil).CheckEquivocation), arg0, arg1, arg2, arg3)
}
