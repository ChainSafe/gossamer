// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/lib/grandpa (interfaces: BlockState,GrandpaState,Network)

// Package grandpa is a generated GoMock package.
package grandpa

import (
	reflect "reflect"

	network "github.com/ChainSafe/gossamer/dot/network"
	types "github.com/ChainSafe/gossamer/dot/types"
	common "github.com/ChainSafe/gossamer/lib/common"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	gomock "github.com/golang/mock/gomock"
	peer "github.com/libp2p/go-libp2p/core/peer"
	protocol "github.com/libp2p/go-libp2p/core/protocol"
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

// FreeFinalisedNotifierChannel mocks base method.
func (m *MockBlockState) FreeFinalisedNotifierChannel(arg0 chan *types.FinalisationInfo) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "FreeFinalisedNotifierChannel", arg0)
}

// FreeFinalisedNotifierChannel indicates an expected call of FreeFinalisedNotifierChannel.
func (mr *MockBlockStateMockRecorder) FreeFinalisedNotifierChannel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeFinalisedNotifierChannel", reflect.TypeOf((*MockBlockState)(nil).FreeFinalisedNotifierChannel), arg0)
}

// FreeImportedBlockNotifierChannel mocks base method.
func (m *MockBlockState) FreeImportedBlockNotifierChannel(arg0 chan *types.Block) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "FreeImportedBlockNotifierChannel", arg0)
}

// FreeImportedBlockNotifierChannel indicates an expected call of FreeImportedBlockNotifierChannel.
func (mr *MockBlockStateMockRecorder) FreeImportedBlockNotifierChannel(arg0 interface{}) *gomock.Call {
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

// GetFinalisedHash mocks base method.
func (m *MockBlockState) GetFinalisedHash(arg0, arg1 uint64) (common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFinalisedHash", arg0, arg1)
	ret0, _ := ret[0].(common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFinalisedHash indicates an expected call of GetFinalisedHash.
func (mr *MockBlockStateMockRecorder) GetFinalisedHash(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFinalisedHash", reflect.TypeOf((*MockBlockState)(nil).GetFinalisedHash), arg0, arg1)
}

// GetFinalisedHeader mocks base method.
func (m *MockBlockState) GetFinalisedHeader(arg0, arg1 uint64) (*types.Header, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFinalisedHeader", arg0, arg1)
	ret0, _ := ret[0].(*types.Header)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFinalisedHeader indicates an expected call of GetFinalisedHeader.
func (mr *MockBlockStateMockRecorder) GetFinalisedHeader(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFinalisedHeader", reflect.TypeOf((*MockBlockState)(nil).GetFinalisedHeader), arg0, arg1)
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

// GetHighestRoundAndSetID mocks base method.
func (m *MockBlockState) GetHighestRoundAndSetID() (uint64, uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHighestRoundAndSetID")
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(uint64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetHighestRoundAndSetID indicates an expected call of GetHighestRoundAndSetID.
func (mr *MockBlockStateMockRecorder) GetHighestRoundAndSetID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHighestRoundAndSetID", reflect.TypeOf((*MockBlockState)(nil).GetHighestRoundAndSetID))
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
func (mr *MockBlockStateMockRecorder) GetRuntime(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRuntime", reflect.TypeOf((*MockBlockState)(nil).GetRuntime), arg0)
}

// HasFinalisedBlock mocks base method.
func (m *MockBlockState) HasFinalisedBlock(arg0, arg1 uint64) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasFinalisedBlock", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// HasFinalisedBlock indicates an expected call of HasFinalisedBlock.
func (mr *MockBlockStateMockRecorder) HasFinalisedBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasFinalisedBlock", reflect.TypeOf((*MockBlockState)(nil).HasFinalisedBlock), arg0, arg1)
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

// SetFinalisedHash mocks base method.
func (m *MockBlockState) SetFinalisedHash(arg0 common.Hash, arg1, arg2 uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetFinalisedHash", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetFinalisedHash indicates an expected call of SetFinalisedHash.
func (mr *MockBlockStateMockRecorder) SetFinalisedHash(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetFinalisedHash", reflect.TypeOf((*MockBlockState)(nil).SetFinalisedHash), arg0, arg1, arg2)
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

// MockGrandpaState is a mock of GrandpaState interface.
type MockGrandpaState struct {
	ctrl     *gomock.Controller
	recorder *MockGrandpaStateMockRecorder
}

// MockGrandpaStateMockRecorder is the mock recorder for MockGrandpaState.
type MockGrandpaStateMockRecorder struct {
	mock *MockGrandpaState
}

// NewMockGrandpaState creates a new mock instance.
func NewMockGrandpaState(ctrl *gomock.Controller) *MockGrandpaState {
	mock := &MockGrandpaState{ctrl: ctrl}
	mock.recorder = &MockGrandpaStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGrandpaState) EXPECT() *MockGrandpaStateMockRecorder {
	return m.recorder
}

// GetAuthorities mocks base method.
func (m *MockGrandpaState) GetAuthorities(arg0 uint64) ([]types.GrandpaVoter, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAuthorities", arg0)
	ret0, _ := ret[0].([]types.GrandpaVoter)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAuthorities indicates an expected call of GetAuthorities.
func (mr *MockGrandpaStateMockRecorder) GetAuthorities(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAuthorities", reflect.TypeOf((*MockGrandpaState)(nil).GetAuthorities), arg0)
}

// GetCurrentSetID mocks base method.
func (m *MockGrandpaState) GetCurrentSetID() (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentSetID")
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCurrentSetID indicates an expected call of GetCurrentSetID.
func (mr *MockGrandpaStateMockRecorder) GetCurrentSetID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentSetID", reflect.TypeOf((*MockGrandpaState)(nil).GetCurrentSetID))
}

// GetLatestRound mocks base method.
func (m *MockGrandpaState) GetLatestRound() (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestRound")
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestRound indicates an expected call of GetLatestRound.
func (mr *MockGrandpaStateMockRecorder) GetLatestRound() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestRound", reflect.TypeOf((*MockGrandpaState)(nil).GetLatestRound))
}

// GetPrecommits mocks base method.
func (m *MockGrandpaState) GetPrecommits(arg0, arg1 uint64) ([]types.GrandpaSignedVote, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPrecommits", arg0, arg1)
	ret0, _ := ret[0].([]types.GrandpaSignedVote)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPrecommits indicates an expected call of GetPrecommits.
func (mr *MockGrandpaStateMockRecorder) GetPrecommits(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPrecommits", reflect.TypeOf((*MockGrandpaState)(nil).GetPrecommits), arg0, arg1)
}

// GetPrevotes mocks base method.
func (m *MockGrandpaState) GetPrevotes(arg0, arg1 uint64) ([]types.GrandpaSignedVote, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPrevotes", arg0, arg1)
	ret0, _ := ret[0].([]types.GrandpaSignedVote)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPrevotes indicates an expected call of GetPrevotes.
func (mr *MockGrandpaStateMockRecorder) GetPrevotes(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPrevotes", reflect.TypeOf((*MockGrandpaState)(nil).GetPrevotes), arg0, arg1)
}

// GetSetIDByBlockNumber mocks base method.
func (m *MockGrandpaState) GetSetIDByBlockNumber(arg0 uint) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSetIDByBlockNumber", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSetIDByBlockNumber indicates an expected call of GetSetIDByBlockNumber.
func (mr *MockGrandpaStateMockRecorder) GetSetIDByBlockNumber(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSetIDByBlockNumber", reflect.TypeOf((*MockGrandpaState)(nil).GetSetIDByBlockNumber), arg0)
}

// NextGrandpaAuthorityChange mocks base method.
func (m *MockGrandpaState) NextGrandpaAuthorityChange(arg0 common.Hash, arg1 uint) (uint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextGrandpaAuthorityChange", arg0, arg1)
	ret0, _ := ret[0].(uint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NextGrandpaAuthorityChange indicates an expected call of NextGrandpaAuthorityChange.
func (mr *MockGrandpaStateMockRecorder) NextGrandpaAuthorityChange(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextGrandpaAuthorityChange", reflect.TypeOf((*MockGrandpaState)(nil).NextGrandpaAuthorityChange), arg0, arg1)
}

// SetLatestRound mocks base method.
func (m *MockGrandpaState) SetLatestRound(arg0 uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetLatestRound", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetLatestRound indicates an expected call of SetLatestRound.
func (mr *MockGrandpaStateMockRecorder) SetLatestRound(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLatestRound", reflect.TypeOf((*MockGrandpaState)(nil).SetLatestRound), arg0)
}

// SetPrecommits mocks base method.
func (m *MockGrandpaState) SetPrecommits(arg0, arg1 uint64, arg2 []types.GrandpaSignedVote) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetPrecommits", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetPrecommits indicates an expected call of SetPrecommits.
func (mr *MockGrandpaStateMockRecorder) SetPrecommits(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPrecommits", reflect.TypeOf((*MockGrandpaState)(nil).SetPrecommits), arg0, arg1, arg2)
}

// SetPrevotes mocks base method.
func (m *MockGrandpaState) SetPrevotes(arg0, arg1 uint64, arg2 []types.GrandpaSignedVote) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetPrevotes", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetPrevotes indicates an expected call of SetPrevotes.
func (mr *MockGrandpaStateMockRecorder) SetPrevotes(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPrevotes", reflect.TypeOf((*MockGrandpaState)(nil).SetPrevotes), arg0, arg1, arg2)
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

// RegisterNotificationsProtocol mocks base method.
func (m *MockNetwork) RegisterNotificationsProtocol(arg0 protocol.ID, arg1 byte, arg2 func() (network.Handshake, error), arg3 func([]byte) (network.Handshake, error), arg4 func(peer.ID, network.Handshake) error, arg5 func([]byte) (network.NotificationsMessage, error), arg6 func(peer.ID, network.NotificationsMessage) (bool, error), arg7 func(peer.ID, network.NotificationsMessage), arg8 uint64) error {
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
