// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/parachain/overseer (interfaces: BlockState)
//
// Generated by this command:
//
//	mockgen -destination=mock_blockstate_test.go -package=collatorprotocol github.com/ChainSafe/gossamer/dot/parachain/overseer BlockState
//

// Package collatorprotocol is a generated GoMock package.
package collatorprotocol

import (
	reflect "reflect"

	types "github.com/ChainSafe/gossamer/dot/types"
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

// FreeFinalisedNotifierChannel mocks base method.
func (m *MockBlockState) FreeFinalisedNotifierChannel(arg0 chan *types.FinalisationInfo) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "FreeFinalisedNotifierChannel", arg0)
}

// FreeFinalisedNotifierChannel indicates an expected call of FreeFinalisedNotifierChannel.
func (mr *MockBlockStateMockRecorder) FreeFinalisedNotifierChannel(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeFinalisedNotifierChannel", reflect.TypeOf((*MockBlockState)(nil).FreeFinalisedNotifierChannel), arg0)
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