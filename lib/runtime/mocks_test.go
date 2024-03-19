// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/lib/runtime (interfaces: Memory)
//
// Generated by this command:
//
//	mockgen -destination=mocks_test.go -package runtime . Memory
//
// Package runtime is a generated GoMock package.
package runtime

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockMemory is a mock of Memory interface.
type MockMemory struct {
	ctrl     *gomock.Controller
	recorder *MockMemoryMockRecorder
}

// MockMemoryMockRecorder is the mock recorder for MockMemory.
type MockMemoryMockRecorder struct {
	mock *MockMemory
}

// NewMockMemory creates a new mock instance.
func NewMockMemory(ctrl *gomock.Controller) *MockMemory {
	mock := &MockMemory{ctrl: ctrl}
	mock.recorder = &MockMemoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMemory) EXPECT() *MockMemoryMockRecorder {
	return m.recorder
}

// Grow mocks base method.
func (m *MockMemory) Grow(arg0 uint32) (uint32, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Grow", arg0)
	ret0, _ := ret[0].(uint32)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// Grow indicates an expected call of Grow.
func (mr *MockMemoryMockRecorder) Grow(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Grow", reflect.TypeOf((*MockMemory)(nil).Grow), arg0)
}

// Read mocks base method.
func (m *MockMemory) Read(arg0 uint32, arg1 uint64) ([]byte, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockMemoryMockRecorder) Read(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockMemory)(nil).Read), arg0, arg1)
}

// ReadByte mocks base method.
func (m *MockMemory) ReadByte(arg0 uint32) (byte, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadByte", arg0)
	ret0, _ := ret[0].(byte)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// ReadByte indicates an expected call of ReadByte.
func (mr *MockMemoryMockRecorder) ReadByte(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadByte", reflect.TypeOf((*MockMemory)(nil).ReadByte), arg0)
}

// ReadUint64Le mocks base method.
func (m *MockMemory) ReadUint64Le(arg0 uint32) (uint64, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadUint64Le", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// ReadUint64Le indicates an expected call of ReadUint64Le.
func (mr *MockMemoryMockRecorder) ReadUint64Le(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadUint64Le", reflect.TypeOf((*MockMemory)(nil).ReadUint64Le), arg0)
}

// Size mocks base method.
func (m *MockMemory) Size() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Size")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// Size indicates an expected call of Size.
func (mr *MockMemoryMockRecorder) Size() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Size", reflect.TypeOf((*MockMemory)(nil).Size))
}

// Write mocks base method.
func (m *MockMemory) Write(arg0 uint32, arg1 []byte) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Write indicates an expected call of Write.
func (mr *MockMemoryMockRecorder) Write(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockMemory)(nil).Write), arg0, arg1)
}

// WriteByte mocks base method.
func (m *MockMemory) WriteByte(arg0 uint32, arg1 byte) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteByte", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// WriteByte indicates an expected call of WriteByte.
func (mr *MockMemoryMockRecorder) WriteByte(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteByte", reflect.TypeOf((*MockMemory)(nil).WriteByte), arg0, arg1)
}

// WriteUint64Le mocks base method.
func (m *MockMemory) WriteUint64Le(arg0 uint32, arg1 uint64) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteUint64Le", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// WriteUint64Le indicates an expected call of WriteUint64Le.
func (mr *MockMemoryMockRecorder) WriteUint64Le(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteUint64Le", reflect.TypeOf((*MockMemory)(nil).WriteUint64Le), arg0, arg1)
}
