// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/parachain/overseer (interfaces: Subsystem)

// Package collatorprotocol is a generated GoMock package.
package collatorprotocol

import (
	context "context"
	reflect "reflect"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	gomock "go.uber.org/mock/gomock"
)

// MockSubsystem is a mock of Subsystem interface.
type MockSubsystem struct {
	ctrl     *gomock.Controller
	recorder *MockSubsystemMockRecorder
}

// MockSubsystemMockRecorder is the mock recorder for MockSubsystem.
type MockSubsystemMockRecorder struct {
	mock *MockSubsystem
}

// NewMockSubsystem creates a new mock instance.
func NewMockSubsystem(ctrl *gomock.Controller) *MockSubsystem {
	mock := &MockSubsystem{ctrl: ctrl}
	mock.recorder = &MockSubsystemMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSubsystem) EXPECT() *MockSubsystemMockRecorder {
	return m.recorder
}

// Name mocks base method.
func (m *MockSubsystem) Name() parachaintypes.SubSystemName {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(parachaintypes.SubSystemName)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockSubsystemMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockSubsystem)(nil).Name))
}

// ProcessActiveLeavesUpdateSignal mocks base method.
func (m *MockSubsystem) ProcessActiveLeavesUpdateSignal(arg0 parachaintypes.ActiveLeavesUpdateSignal) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProcessActiveLeavesUpdateSignal", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ProcessActiveLeavesUpdateSignal indicates an expected call of ProcessActiveLeavesUpdateSignal.
func (mr *MockSubsystemMockRecorder) ProcessActiveLeavesUpdateSignal(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessActiveLeavesUpdateSignal", reflect.TypeOf((*MockSubsystem)(nil).ProcessActiveLeavesUpdateSignal), arg0)
}

// ProcessBlockFinalizedSignal mocks base method.
func (m *MockSubsystem) ProcessBlockFinalizedSignal(arg0 parachaintypes.BlockFinalizedSignal) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ProcessBlockFinalizedSignal", arg0)
}

// ProcessBlockFinalizedSignal indicates an expected call of ProcessBlockFinalizedSignal.
func (mr *MockSubsystemMockRecorder) ProcessBlockFinalizedSignal(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessBlockFinalizedSignal", reflect.TypeOf((*MockSubsystem)(nil).ProcessBlockFinalizedSignal), arg0)
}

// Run mocks base method.
func (m *MockSubsystem) Run(arg0 context.Context, arg1, arg2 chan interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Run", arg0, arg1, arg2)
}

// Run indicates an expected call of Run.
func (mr *MockSubsystemMockRecorder) Run(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockSubsystem)(nil).Run), arg0, arg1, arg2)
}

// Stop mocks base method.
func (m *MockSubsystem) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop.
func (mr *MockSubsystemMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockSubsystem)(nil).Stop))
}
