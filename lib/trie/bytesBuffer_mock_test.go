// Code generated by MockGen. DO NOT EDIT.
// Source: hash.go

// Package trie is a generated GoMock package.
package trie

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockbytesBuffer is a mock of bytesBuffer interface.
type MockbytesBuffer struct {
	ctrl     *gomock.Controller
	recorder *MockbytesBufferMockRecorder
}

// MockbytesBufferMockRecorder is the mock recorder for MockbytesBuffer.
type MockbytesBufferMockRecorder struct {
	mock *MockbytesBuffer
}

// NewMockbytesBuffer creates a new mock instance.
func NewMockbytesBuffer(ctrl *gomock.Controller) *MockbytesBuffer {
	mock := &MockbytesBuffer{ctrl: ctrl}
	mock.recorder = &MockbytesBufferMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockbytesBuffer) EXPECT() *MockbytesBufferMockRecorder {
	return m.recorder
}

// Bytes mocks base method.
func (m *MockbytesBuffer) Bytes() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Bytes")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// Bytes indicates an expected call of Bytes.
func (mr *MockbytesBufferMockRecorder) Bytes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Bytes", reflect.TypeOf((*MockbytesBuffer)(nil).Bytes))
}

// Len mocks base method.
func (m *MockbytesBuffer) Len() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Len")
	ret0, _ := ret[0].(int)
	return ret0
}

// Len indicates an expected call of Len.
func (mr *MockbytesBufferMockRecorder) Len() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Len", reflect.TypeOf((*MockbytesBuffer)(nil).Len))
}

// Write mocks base method.
func (m *MockbytesBuffer) Write(p []byte) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", p)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Write indicates an expected call of Write.
func (mr *MockbytesBufferMockRecorder) Write(p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockbytesBuffer)(nil).Write), p)
}
