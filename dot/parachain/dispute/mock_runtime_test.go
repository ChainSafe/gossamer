// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/parachain/runtime (interfaces: RuntimeInstance)

// Package dispute is a generated GoMock package.
package dispute

import (
	reflect "reflect"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	common "github.com/ChainSafe/gossamer/lib/common"
<<<<<<<< HEAD:dot/parachain/dispute/mock_runtime_test.go
	scale "github.com/ChainSafe/gossamer/pkg/scale"
========
>>>>>>>> e496f25e2 (add tests):dot/parachain/runtime_mock.go
	gomock "github.com/golang/mock/gomock"
)

// MockRuntimeInstance is a mock of RuntimeInstance interface.
type MockRuntimeInstance struct {
	ctrl     *gomock.Controller
	recorder *MockRuntimeInstanceMockRecorder
}

// MockRuntimeInstanceMockRecorder is the mock recorder for MockRuntimeInstance.
type MockRuntimeInstanceMockRecorder struct {
	mock *MockRuntimeInstance
}

// NewMockRuntimeInstance creates a new mock instance.
func NewMockRuntimeInstance(ctrl *gomock.Controller) *MockRuntimeInstance {
	mock := &MockRuntimeInstance{ctrl: ctrl}
	mock.recorder = &MockRuntimeInstanceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRuntimeInstance) EXPECT() *MockRuntimeInstanceMockRecorder {
	return m.recorder
}

// ParachainHostCandidateEvents mocks base method.
func (m *MockRuntimeInstance) ParachainHostCandidateEvents(arg0 common.Hash) (*scale.VaryingDataTypeSlice, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostCandidateEvents", arg0)
	ret0, _ := ret[0].(*scale.VaryingDataTypeSlice)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostCandidateEvents indicates an expected call of ParachainHostCandidateEvents.
func (mr *MockRuntimeInstanceMockRecorder) ParachainHostCandidateEvents(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostCandidateEvents", reflect.TypeOf((*MockRuntimeInstance)(nil).ParachainHostCandidateEvents), arg0)
}

// ParachainHostCheckValidationOutputs mocks base method.
func (m *MockRuntimeInstance) ParachainHostCheckValidationOutputs(arg0 uint32, arg1 parachaintypes.CandidateCommitments) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostCheckValidationOutputs", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostCheckValidationOutputs indicates an expected call of ParachainHostCheckValidationOutputs.
func (mr *MockRuntimeInstanceMockRecorder) ParachainHostCheckValidationOutputs(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostCheckValidationOutputs", reflect.TypeOf((*MockRuntimeInstance)(nil).ParachainHostCheckValidationOutputs), arg0, arg1)
}

// ParachainHostOnChainVotes mocks base method.
func (m *MockRuntimeInstance) ParachainHostOnChainVotes(arg0 common.Hash) (*parachaintypes.ScrapedOnChainVotes, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostOnChainVotes", arg0)
	ret0, _ := ret[0].(*parachaintypes.ScrapedOnChainVotes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostOnChainVotes indicates an expected call of ParachainHostOnChainVotes.
func (mr *MockRuntimeInstanceMockRecorder) ParachainHostOnChainVotes(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostOnChainVotes", reflect.TypeOf((*MockRuntimeInstance)(nil).ParachainHostOnChainVotes), arg0)
}

// ParachainHostPersistedValidationData mocks base method.
func (m *MockRuntimeInstance) ParachainHostPersistedValidationData(arg0 uint32, arg1 parachaintypes.OccupiedCoreAssumption) (*parachaintypes.PersistedValidationData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostPersistedValidationData", arg0, arg1)
	ret0, _ := ret[0].(*parachaintypes.PersistedValidationData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostPersistedValidationData indicates an expected call of ParachainHostPersistedValidationData.
func (mr *MockRuntimeInstanceMockRecorder) ParachainHostPersistedValidationData(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostPersistedValidationData", reflect.TypeOf((*MockRuntimeInstance)(nil).ParachainHostPersistedValidationData), arg0, arg1)
}

// ParachainHostValidationCode mocks base method.
func (m *MockRuntimeInstance) ParachainHostValidationCode(arg0 uint32, arg1 parachaintypes.OccupiedCoreAssumption) (*parachaintypes.ValidationCode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostValidationCode", arg0, arg1)
	ret0, _ := ret[0].(*parachaintypes.ValidationCode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostValidationCode indicates an expected call of ParachainHostValidationCode.
func (mr *MockRuntimeInstanceMockRecorder) ParachainHostValidationCode(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostValidationCode", reflect.TypeOf((*MockRuntimeInstance)(nil).ParachainHostValidationCode), arg0, arg1)
}

// ParachainHostValidationCodeByHash mocks base method.
func (m *MockRuntimeInstance) ParachainHostValidationCodeByHash(arg0 common.Hash, arg1 parachaintypes.ValidationCodeHash) (*parachaintypes.ValidationCode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParachainHostValidationCodeByHash", arg0, arg1)
	ret0, _ := ret[0].(*parachaintypes.ValidationCode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParachainHostValidationCodeByHash indicates an expected call of ParachainHostValidationCodeByHash.
func (mr *MockRuntimeInstanceMockRecorder) ParachainHostValidationCodeByHash(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParachainHostValidationCodeByHash", reflect.TypeOf((*MockRuntimeInstance)(nil).ParachainHostValidationCodeByHash), arg0, arg1)
}
