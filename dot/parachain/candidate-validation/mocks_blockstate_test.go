// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/parachain/candidate-validation (interfaces: BlockState)
//
// Generated by this command:
//
//	mockgen -destination=mocks_blockstate_test.go -package=candidatevalidation . BlockState
//

// Package candidatevalidation is a generated GoMock package.
package candidatevalidation

import (
	reflect "reflect"

	common "github.com/ChainSafe/gossamer/lib/common"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
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
