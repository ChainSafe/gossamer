// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/network (interfaces: BlockState)

// Package network is a generated GoMock package.
package network

import (
	reflect "reflect"

	types "github.com/ChainSafe/gossamer/dot/types"
	common "github.com/ChainSafe/gossamer/lib/common"
	gomock "github.com/golang/mock/gomock"
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
