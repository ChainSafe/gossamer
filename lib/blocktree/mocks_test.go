// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/lib/runtime (interfaces: Instance)

// Package blocktree is a generated GoMock package.
package blocktree

import (
	reflect "reflect"

	types "github.com/ChainSafe/gossamer/dot/types"
	common "github.com/ChainSafe/gossamer/lib/common"
	keystore "github.com/ChainSafe/gossamer/lib/keystore"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	transaction "github.com/ChainSafe/gossamer/lib/transaction"
	gomock "github.com/golang/mock/gomock"
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
func (mr *MockInstanceMockRecorder) ApplyExtrinsic(arg0 interface{}) *gomock.Call {
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
func (mr *MockInstanceMockRecorder) BabeGenerateKeyOwnershipProof(arg0, arg1 interface{}) *gomock.Call {
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
func (mr *MockInstanceMockRecorder) BabeSubmitReportEquivocationUnsignedExtrinsic(arg0, arg1 interface{}) *gomock.Call {
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
func (mr *MockInstanceMockRecorder) DecodeSessionKeys(arg0 interface{}) *gomock.Call {
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
func (mr *MockInstanceMockRecorder) Exec(arg0, arg1 interface{}) *gomock.Call {
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
func (mr *MockInstanceMockRecorder) ExecuteBlock(arg0 interface{}) *gomock.Call {
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

// InherentExtrinsics mocks base method.
func (m *MockInstance) InherentExtrinsics(arg0 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InherentExtrinsics", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InherentExtrinsics indicates an expected call of InherentExtrinsics.
func (mr *MockInstanceMockRecorder) InherentExtrinsics(arg0 interface{}) *gomock.Call {
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
func (mr *MockInstanceMockRecorder) InitializeBlock(arg0 interface{}) *gomock.Call {
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

// PaymentQueryInfo mocks base method.
func (m *MockInstance) PaymentQueryInfo(arg0 []byte) (*types.RuntimeDispatchInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentQueryInfo", arg0)
	ret0, _ := ret[0].(*types.RuntimeDispatchInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentQueryInfo indicates an expected call of PaymentQueryInfo.
func (mr *MockInstanceMockRecorder) PaymentQueryInfo(arg0 interface{}) *gomock.Call {
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
func (mr *MockInstanceMockRecorder) SetContextStorage(arg0 interface{}) *gomock.Call {
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

// UpdateRuntimeCode mocks base method.
func (m *MockInstance) UpdateRuntimeCode(arg0 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRuntimeCode", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateRuntimeCode indicates an expected call of UpdateRuntimeCode.
func (mr *MockInstanceMockRecorder) UpdateRuntimeCode(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRuntimeCode", reflect.TypeOf((*MockInstance)(nil).UpdateRuntimeCode), arg0)
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
func (mr *MockInstanceMockRecorder) ValidateTransaction(arg0 interface{}) *gomock.Call {
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
func (m *MockInstance) Version() runtime.Version {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Version")
	ret0, _ := ret[0].(runtime.Version)
	return ret0
}

// Version indicates an expected call of Version.
func (mr *MockInstanceMockRecorder) Version() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Version", reflect.TypeOf((*MockInstance)(nil).Version))
}
