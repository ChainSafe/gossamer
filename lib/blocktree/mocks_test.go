// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/lib/runtime (interfaces: Instance)
//
// Generated by this command:
//
//	mockgen -destination=mocks_test.go -package blocktree github.com/ChainSafe/gossamer/lib/runtime Instance
//

// Package blocktree is a generated GoMock package.
package blocktree

import (
	reflect "reflect"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	types "github.com/ChainSafe/gossamer/dot/types"
	common "github.com/ChainSafe/gossamer/lib/common"
	ed25519 "github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	keystore "github.com/ChainSafe/gossamer/lib/keystore"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	transaction "github.com/ChainSafe/gossamer/lib/transaction"
	scale "github.com/ChainSafe/gossamer/pkg/scale"
	gomock "go.uber.org/mock/gomock"
)

// MockInstance is a mock of Instance interface.
type MockInstance struct {
	ctrl     *gomock.Controller
	recorder *MockInstanceMockRecorder
}

// MockInstanceMockRecorder is the mock recorder for MockInstance.
type MockInstanceMockRecorder struct {
	mock *MockInstance
}

// NewMockInstance creates a new mock instance.
func NewMockInstance(ctrl *gomock.Controller) *MockInstance {
	mock := &MockInstance{ctrl: ctrl}
	mock.recorder = &MockInstanceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInstance) EXPECT() *MockInstanceMockRecorder {
	return m.recorder
}

// ApplyExtrinsic mocks base method.
func (m *MockInstance) ApplyExtrinsic(arg0 types.Extrinsic) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyExtrinsic", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ApplyExtrinsic indicates an expected call of ApplyExtrinsic.
func (mr *MockInstanceMockRecorder) ApplyExtrinsic(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyExtrinsic", reflect.TypeOf((*MockInstance)(nil).ApplyExtrinsic), arg0)
}

// BabeConfiguration mocks base method.
func (m *MockInstance) BabeConfiguration() (*types.BabeConfiguration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BabeConfiguration")
	ret0, _ := ret[0].(*types.BabeConfiguration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BabeConfiguration indicates an expected call of BabeConfiguration.
func (mr *MockInstanceMockRecorder) BabeConfiguration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BabeConfiguration", reflect.TypeOf((*MockInstance)(nil).BabeConfiguration))
}

// BabeGenerateKeyOwnershipProof mocks base method.
func (m *MockInstance) BabeGenerateKeyOwnershipProof(arg0 uint64, arg1 [32]byte) (types.OpaqueKeyOwnershipProof, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BabeGenerateKeyOwnershipProof", arg0, arg1)
	ret0, _ := ret[0].(types.OpaqueKeyOwnershipProof)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BabeGenerateKeyOwnershipProof indicates an expected call of BabeGenerateKeyOwnershipProof.
func (mr *MockInstanceMockRecorder) BabeGenerateKeyOwnershipProof(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BabeGenerateKeyOwnershipProof", reflect.TypeOf((*MockInstance)(nil).BabeGenerateKeyOwnershipProof), arg0, arg1)
}

// BabeSubmitReportEquivocationUnsignedExtrinsic mocks base method.
func (m *MockInstance) BabeSubmitReportEquivocationUnsignedExtrinsic(arg0 types.BabeEquivocationProof, arg1 types.OpaqueKeyOwnershipProof) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BabeSubmitReportEquivocationUnsignedExtrinsic", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// BabeSubmitReportEquivocationUnsignedExtrinsic indicates an expected call of BabeSubmitReportEquivocationUnsignedExtrinsic.
func (mr *MockInstanceMockRecorder) BabeSubmitReportEquivocationUnsignedExtrinsic(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BabeSubmitReportEquivocationUnsignedExtrinsic", reflect.TypeOf((*MockInstance)(nil).BabeSubmitReportEquivocationUnsignedExtrinsic), arg0, arg1)
}

// CheckInherents mocks base method.
func (m *MockInstance) CheckInherents() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "CheckInherents")
}

// CheckInherents indicates an expected call of CheckInherents.
func (mr *MockInstanceMockRecorder) CheckInherents() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckInherents", reflect.TypeOf((*MockInstance)(nil).CheckInherents))
}

// DecodeSessionKeys mocks base method.
func (m *MockInstance) DecodeSessionKeys(arg0 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DecodeSessionKeys", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DecodeSessionKeys indicates an expected call of DecodeSessionKeys.
func (mr *MockInstanceMockRecorder) DecodeSessionKeys(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DecodeSessionKeys", reflect.TypeOf((*MockInstance)(nil).DecodeSessionKeys), arg0)
}

// Exec mocks base method.
func (m *MockInstance) Exec(arg0 string, arg1 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exec", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exec indicates an expected call of Exec.
func (mr *MockInstanceMockRecorder) Exec(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exec", reflect.TypeOf((*MockInstance)(nil).Exec), arg0, arg1)
}

// ExecuteBlock mocks base method.
func (m *MockInstance) ExecuteBlock(arg0 *types.Block) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecuteBlock", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecuteBlock indicates an expected call of ExecuteBlock.
func (mr *MockInstanceMockRecorder) ExecuteBlock(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecuteBlock", reflect.TypeOf((*MockInstance)(nil).ExecuteBlock), arg0)
}

// FinalizeBlock mocks base method.
func (m *MockInstance) FinalizeBlock() (*types.Header, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FinalizeBlock")
	ret0, _ := ret[0].(*types.Header)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FinalizeBlock indicates an expected call of FinalizeBlock.
func (mr *MockInstanceMockRecorder) FinalizeBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FinalizeBlock", reflect.TypeOf((*MockInstance)(nil).FinalizeBlock))
}

// GenerateSessionKeys mocks base method.
func (m *MockInstance) GenerateSessionKeys() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "GenerateSessionKeys")
}

// GenerateSessionKeys indicates an expected call of GenerateSessionKeys.
func (mr *MockInstanceMockRecorder) GenerateSessionKeys() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateSessionKeys", reflect.TypeOf((*MockInstance)(nil).GenerateSessionKeys))
}

// GetCodeHash mocks base method.
func (m *MockInstance) GetCodeHash() common.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCodeHash")
	ret0, _ := ret[0].(common.Hash)
	return ret0
}

// GetCodeHash indicates an expected call of GetCodeHash.
func (mr *MockInstanceMockRecorder) GetCodeHash() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCodeHash", reflect.TypeOf((*MockInstance)(nil).GetCodeHash))
}

// GrandpaAuthorities mocks base method.
func (m *MockInstance) GrandpaAuthorities() ([]types.Authority, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GrandpaAuthorities")
	ret0, _ := ret[0].([]types.Authority)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GrandpaAuthorities indicates an expected call of GrandpaAuthorities.
func (mr *MockInstanceMockRecorder) GrandpaAuthorities() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GrandpaAuthorities", reflect.TypeOf((*MockInstance)(nil).GrandpaAuthorities))
}

// GrandpaGenerateKeyOwnershipProof mocks base method.
func (m *MockInstance) GrandpaGenerateKeyOwnershipProof(arg0 uint64, arg1 ed25519.PublicKeyBytes) (types.GrandpaOpaqueKeyOwnershipProof, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GrandpaGenerateKeyOwnershipProof", arg0, arg1)
	ret0, _ := ret[0].(types.GrandpaOpaqueKeyOwnershipProof)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GrandpaGenerateKeyOwnershipProof indicates an expected call of GrandpaGenerateKeyOwnershipProof.
func (mr *MockInstanceMockRecorder) GrandpaGenerateKeyOwnershipProof(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GrandpaGenerateKeyOwnershipProof", reflect.TypeOf((*MockInstance)(nil).GrandpaGenerateKeyOwnershipProof), arg0, arg1)
}

// GrandpaSubmitReportEquivocationUnsignedExtrinsic mocks base method.
func (m *MockInstance) GrandpaSubmitReportEquivocationUnsignedExtrinsic(arg0 types.GrandpaEquivocationProof, arg1 types.GrandpaOpaqueKeyOwnershipProof) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GrandpaSubmitReportEquivocationUnsignedExtrinsic", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GrandpaSubmitReportEquivocationUnsignedExtrinsic indicates an expected call of GrandpaSubmitReportEquivocationUnsignedExtrinsic.
func (mr *MockInstanceMockRecorder) GrandpaSubmitReportEquivocationUnsignedExtrinsic(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GrandpaSubmitReportEquivocationUnsignedExtrinsic", reflect.TypeOf((*MockInstance)(nil).GrandpaSubmitReportEquivocationUnsignedExtrinsic), arg0, arg1)
}

// InherentExtrinsics mocks base method.
func (m *MockInstance) InherentExtrinsics(arg0 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InherentExtrinsics", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InherentExtrinsics indicates an expected call of InherentExtrinsics.
func (mr *MockInstanceMockRecorder) InherentExtrinsics(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InherentExtrinsics", reflect.TypeOf((*MockInstance)(nil).InherentExtrinsics), arg0)
}

// InitializeBlock mocks base method.
func (m *MockInstance) InitializeBlock(arg0 *types.Header) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InitializeBlock", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// InitializeBlock indicates an expected call of InitializeBlock.
func (mr *MockInstanceMockRecorder) InitializeBlock(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitializeBlock", reflect.TypeOf((*MockInstance)(nil).InitializeBlock), arg0)
}

// Keystore mocks base method.
func (m *MockInstance) Keystore() *keystore.GlobalKeystore {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Keystore")
	ret0, _ := ret[0].(*keystore.GlobalKeystore)
	return ret0
}

// Keystore indicates an expected call of Keystore.
func (mr *MockInstanceMockRecorder) Keystore() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Keystore", reflect.TypeOf((*MockInstance)(nil).Keystore))
}

// Metadata mocks base method.
func (m *MockInstance) Metadata() ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Metadata")
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Metadata indicates an expected call of Metadata.
func (mr *MockInstanceMockRecorder) Metadata() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Metadata", reflect.TypeOf((*MockInstance)(nil).Metadata))
}

// NetworkService mocks base method.
func (m *MockInstance) NetworkService() runtime.BasicNetwork {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NetworkService")
	ret0, _ := ret[0].(runtime.BasicNetwork)
	return ret0
}

// NetworkService indicates an expected call of NetworkService.
func (mr *MockInstanceMockRecorder) NetworkService() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NetworkService", reflect.TypeOf((*MockInstance)(nil).NetworkService))
}

// NodeStorage mocks base method.
func (m *MockInstance) NodeStorage() runtime.NodeStorage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NodeStorage")
	ret0, _ := ret[0].(runtime.NodeStorage)
	return ret0
}

// NodeStorage indicates an expected call of NodeStorage.
func (mr *MockInstanceMockRecorder) NodeStorage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NodeStorage", reflect.TypeOf((*MockInstance)(nil).NodeStorage))
}

// OffchainWorker mocks base method.
func (m *MockInstance) OffchainWorker() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OffchainWorker")
}

// OffchainWorker indicates an expected call of OffchainWorker.
func (mr *MockInstanceMockRecorder) OffchainWorker() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OffchainWorker", reflect.TypeOf((*MockInstance)(nil).OffchainWorker))
}

// ParachainHostAsyncBackingParams mocks base method.
func (m *MockInstance) ParachainHostAsyncBackingParams() (*parachaintypes.AsyncBackingParams, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostAsyncBackingParams")
	ret0, _ := ret[0].(*parachaintypes.AsyncBackingParams)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostAsyncBackingParams indicates an expected call of ParachainHostAsyncBackingParams.
func (mr *MockInstanceMockRecorder) ParachainHostAsyncBackingParams() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostAsyncBackingParams", reflect.TypeOf((*MockInstance)(nil).ParachainHostAsyncBackingParams))
}

// ParachainHostAvailabilityCores mocks base method.
func (m *MockInstance) ParachainHostAvailabilityCores() (*scale.VaryingDataTypeSlice, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostAvailabilityCores")
	ret0, _ := ret[0].(*scale.VaryingDataTypeSlice)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostAvailabilityCores indicates an expected call of ParachainHostAvailabilityCores.
func (mr *MockInstanceMockRecorder) ParachainHostAvailabilityCores() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostAvailabilityCores", reflect.TypeOf((*MockInstance)(nil).ParachainHostAvailabilityCores))
}

// ParachainHostCandidateEvents mocks base method.
func (m *MockInstance) ParachainHostCandidateEvents() (*scale.VaryingDataTypeSlice, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostCandidateEvents")
	ret0, _ := ret[0].(*scale.VaryingDataTypeSlice)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostCandidateEvents indicates an expected call of ParachainHostCandidateEvents.
func (mr *MockInstanceMockRecorder) ParachainHostCandidateEvents() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostCandidateEvents", reflect.TypeOf((*MockInstance)(nil).ParachainHostCandidateEvents))
}

// ParachainHostCandidatePendingAvailability mocks base method.
func (m *MockInstance) ParachainHostCandidatePendingAvailability(arg0 parachaintypes.ParaID) (*parachaintypes.CommittedCandidateReceipt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostCandidatePendingAvailability", arg0)
	ret0, _ := ret[0].(*parachaintypes.CommittedCandidateReceipt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostCandidatePendingAvailability indicates an expected call of ParachainHostCandidatePendingAvailability.
func (mr *MockInstanceMockRecorder) ParachainHostCandidatePendingAvailability(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostCandidatePendingAvailability", reflect.TypeOf((*MockInstance)(nil).ParachainHostCandidatePendingAvailability), arg0)
}

// ParachainHostCheckValidationOutputs mocks base method.
func (m *MockInstance) ParachainHostCheckValidationOutputs(arg0 parachaintypes.ParaID, arg1 parachaintypes.CandidateCommitments) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostCheckValidationOutputs", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostCheckValidationOutputs indicates an expected call of ParachainHostCheckValidationOutputs.
func (mr *MockInstanceMockRecorder) ParachainHostCheckValidationOutputs(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostCheckValidationOutputs", reflect.TypeOf((*MockInstance)(nil).ParachainHostCheckValidationOutputs), arg0, arg1)
}

// ParachainHostMinimumBackingVotes mocks base method.
func (m *MockInstance) ParachainHostMinimumBackingVotes() (uint32, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostMinimumBackingVotes")
	ret0, _ := ret[0].(uint32)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostMinimumBackingVotes indicates an expected call of ParachainHostMinimumBackingVotes.
func (mr *MockInstanceMockRecorder) ParachainHostMinimumBackingVotes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostMinimumBackingVotes", reflect.TypeOf((*MockInstance)(nil).ParachainHostMinimumBackingVotes))
}

// ParachainHostPersistedValidationData mocks base method.
func (m *MockInstance) ParachainHostPersistedValidationData(arg0 uint32, arg1 parachaintypes.OccupiedCoreAssumption) (*parachaintypes.PersistedValidationData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostPersistedValidationData", arg0, arg1)
	ret0, _ := ret[0].(*parachaintypes.PersistedValidationData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostPersistedValidationData indicates an expected call of ParachainHostPersistedValidationData.
func (mr *MockInstanceMockRecorder) ParachainHostPersistedValidationData(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostPersistedValidationData", reflect.TypeOf((*MockInstance)(nil).ParachainHostPersistedValidationData), arg0, arg1)
}

// ParachainHostSessionIndexForChild mocks base method.
func (m *MockInstance) ParachainHostSessionIndexForChild() (parachaintypes.SessionIndex, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostSessionIndexForChild")
	ret0, _ := ret[0].(parachaintypes.SessionIndex)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostSessionIndexForChild indicates an expected call of ParachainHostSessionIndexForChild.
func (mr *MockInstanceMockRecorder) ParachainHostSessionIndexForChild() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostSessionIndexForChild", reflect.TypeOf((*MockInstance)(nil).ParachainHostSessionIndexForChild))
}

// ParachainHostSessionInfo mocks base method.
func (m *MockInstance) ParachainHostSessionInfo(arg0 parachaintypes.SessionIndex) (*parachaintypes.SessionInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostSessionInfo", arg0)
	ret0, _ := ret[0].(*parachaintypes.SessionInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostSessionInfo indicates an expected call of ParachainHostSessionInfo.
func (mr *MockInstanceMockRecorder) ParachainHostSessionInfo(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostSessionInfo", reflect.TypeOf((*MockInstance)(nil).ParachainHostSessionInfo), arg0)
}

// ParachainHostValidationCode mocks base method.
func (m *MockInstance) ParachainHostValidationCode(arg0 uint32, arg1 parachaintypes.OccupiedCoreAssumption) (*parachaintypes.ValidationCode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostValidationCode", arg0, arg1)
	ret0, _ := ret[0].(*parachaintypes.ValidationCode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostValidationCode indicates an expected call of ParachainHostValidationCode.
func (mr *MockInstanceMockRecorder) ParachainHostValidationCode(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostValidationCode", reflect.TypeOf((*MockInstance)(nil).ParachainHostValidationCode), arg0, arg1)
}

// ParachainHostValidationCodeByHash mocks base method.
func (m *MockInstance) ParachainHostValidationCodeByHash(arg0 common.Hash) (*parachaintypes.ValidationCode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostValidationCodeByHash", arg0)
	ret0, _ := ret[0].(*parachaintypes.ValidationCode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostValidationCodeByHash indicates an expected call of ParachainHostValidationCodeByHash.
func (mr *MockInstanceMockRecorder) ParachainHostValidationCodeByHash(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostValidationCodeByHash", reflect.TypeOf((*MockInstance)(nil).ParachainHostValidationCodeByHash), arg0)
}

// ParachainHostValidatorGroups mocks base method.
func (m *MockInstance) ParachainHostValidatorGroups() (*parachaintypes.ValidatorGroups, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostValidatorGroups")
	ret0, _ := ret[0].(*parachaintypes.ValidatorGroups)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostValidatorGroups indicates an expected call of ParachainHostValidatorGroups.
func (mr *MockInstanceMockRecorder) ParachainHostValidatorGroups() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostValidatorGroups", reflect.TypeOf((*MockInstance)(nil).ParachainHostValidatorGroups))
}

// ParachainHostValidators mocks base method.
func (m *MockInstance) ParachainHostValidators() ([]parachaintypes.ValidatorID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostValidators")
	ret0, _ := ret[0].([]parachaintypes.ValidatorID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostValidators indicates an expected call of ParachainHostValidators.
func (mr *MockInstanceMockRecorder) ParachainHostValidators() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostValidators", reflect.TypeOf((*MockInstance)(nil).ParachainHostValidators))
}

// PaymentQueryInfo mocks base method.
func (m *MockInstance) PaymentQueryInfo(arg0 []byte) (*types.RuntimeDispatchInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentQueryInfo", arg0)
	ret0, _ := ret[0].(*types.RuntimeDispatchInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentQueryInfo indicates an expected call of PaymentQueryInfo.
func (mr *MockInstanceMockRecorder) PaymentQueryInfo(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentQueryInfo", reflect.TypeOf((*MockInstance)(nil).PaymentQueryInfo), arg0)
}

// RandomSeed mocks base method.
func (m *MockInstance) RandomSeed() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RandomSeed")
}

// RandomSeed indicates an expected call of RandomSeed.
func (mr *MockInstanceMockRecorder) RandomSeed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RandomSeed", reflect.TypeOf((*MockInstance)(nil).RandomSeed))
}

// SetContextStorage mocks base method.
func (m *MockInstance) SetContextStorage(arg0 runtime.Storage) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetContextStorage", arg0)
}

// SetContextStorage indicates an expected call of SetContextStorage.
func (mr *MockInstanceMockRecorder) SetContextStorage(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetContextStorage", reflect.TypeOf((*MockInstance)(nil).SetContextStorage), arg0)
}

// Stop mocks base method.
func (m *MockInstance) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop.
func (mr *MockInstanceMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockInstance)(nil).Stop))
}

// ValidateTransaction mocks base method.
func (m *MockInstance) ValidateTransaction(arg0 types.Extrinsic) (*transaction.Validity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateTransaction", arg0)
	ret0, _ := ret[0].(*transaction.Validity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateTransaction indicates an expected call of ValidateTransaction.
func (mr *MockInstanceMockRecorder) ValidateTransaction(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateTransaction", reflect.TypeOf((*MockInstance)(nil).ValidateTransaction), arg0)
}

// Validator mocks base method.
func (m *MockInstance) Validator() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Validator")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Validator indicates an expected call of Validator.
func (mr *MockInstanceMockRecorder) Validator() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Validator", reflect.TypeOf((*MockInstance)(nil).Validator))
}

// Version mocks base method.
func (m *MockInstance) Version() (runtime.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Version")
	ret0, _ := ret[0].(runtime.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Version indicates an expected call of Version.
func (mr *MockInstanceMockRecorder) Version() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Version", reflect.TypeOf((*MockInstance)(nil).Version))
}
