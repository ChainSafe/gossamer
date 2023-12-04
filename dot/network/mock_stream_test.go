// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/libp2p/go-libp2p/core/network (interfaces: Stream)
//
// Generated by this command:
//
//	mockgen -destination=mock_stream_test.go -package network github.com/libp2p/go-libp2p/core/network Stream
//
// Package network is a generated GoMock package.
package network

import (
	reflect "reflect"
	time "time"

	network "github.com/libp2p/go-libp2p/core/network"
	protocol "github.com/libp2p/go-libp2p/core/protocol"
	gomock "go.uber.org/mock/gomock"
)

// MockStream is a mock of Stream interface.
type MockStream struct {
	ctrl     *gomock.Controller
	recorder *MockStreamMockRecorder
}

// MockStreamMockRecorder is the mock recorder for MockStream.
type MockStreamMockRecorder struct {
	mock *MockStream
}

// NewMockStream creates a new mock instance.
func NewMockStream(ctrl *gomock.Controller) *MockStream {
	mock := &MockStream{ctrl: ctrl}
	mock.recorder = &MockStreamMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStream) EXPECT() *MockStreamMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockStream) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockStreamMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockStream)(nil).Close))
}

// CloseRead mocks base method.
func (m *MockStream) CloseRead() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseRead")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseRead indicates an expected call of CloseRead.
func (mr *MockStreamMockRecorder) CloseRead() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseRead", reflect.TypeOf((*MockStream)(nil).CloseRead))
}

// CloseWrite mocks base method.
func (m *MockStream) CloseWrite() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseWrite")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseWrite indicates an expected call of CloseWrite.
func (mr *MockStreamMockRecorder) CloseWrite() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseWrite", reflect.TypeOf((*MockStream)(nil).CloseWrite))
}

// Conn mocks base method.
func (m *MockStream) Conn() network.Conn {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Conn")
	ret0, _ := ret[0].(network.Conn)
	return ret0
}

// Conn indicates an expected call of Conn.
func (mr *MockStreamMockRecorder) Conn() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Conn", reflect.TypeOf((*MockStream)(nil).Conn))
}

// ID mocks base method.
func (m *MockStream) ID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockStreamMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockStream)(nil).ID))
}

// Protocol mocks base method.
func (m *MockStream) Protocol() protocol.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Protocol")
	ret0, _ := ret[0].(protocol.ID)
	return ret0
}

// Protocol indicates an expected call of Protocol.
func (mr *MockStreamMockRecorder) Protocol() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Protocol", reflect.TypeOf((*MockStream)(nil).Protocol))
}

// Read mocks base method.
func (m *MockStream) Read(arg0 []byte) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", arg0)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockStreamMockRecorder) Read(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockStream)(nil).Read), arg0)
}

// Reset mocks base method.
func (m *MockStream) Reset() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reset")
	ret0, _ := ret[0].(error)
	return ret0
}

// Reset indicates an expected call of Reset.
func (mr *MockStreamMockRecorder) Reset() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reset", reflect.TypeOf((*MockStream)(nil).Reset))
}

// Scope mocks base method.
func (m *MockStream) Scope() network.StreamScope {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Scope")
	ret0, _ := ret[0].(network.StreamScope)
	return ret0
}

// Scope indicates an expected call of Scope.
func (mr *MockStreamMockRecorder) Scope() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Scope", reflect.TypeOf((*MockStream)(nil).Scope))
}

// SetDeadline mocks base method.
func (m *MockStream) SetDeadline(arg0 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetDeadline", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetDeadline indicates an expected call of SetDeadline.
func (mr *MockStreamMockRecorder) SetDeadline(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDeadline", reflect.TypeOf((*MockStream)(nil).SetDeadline), arg0)
}

// SetProtocol mocks base method.
func (m *MockStream) SetProtocol(arg0 protocol.ID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetProtocol", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetProtocol indicates an expected call of SetProtocol.
func (mr *MockStreamMockRecorder) SetProtocol(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetProtocol", reflect.TypeOf((*MockStream)(nil).SetProtocol), arg0)
}

// SetReadDeadline mocks base method.
func (m *MockStream) SetReadDeadline(arg0 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetReadDeadline", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetReadDeadline indicates an expected call of SetReadDeadline.
func (mr *MockStreamMockRecorder) SetReadDeadline(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetReadDeadline", reflect.TypeOf((*MockStream)(nil).SetReadDeadline), arg0)
}

// SetWriteDeadline mocks base method.
func (m *MockStream) SetWriteDeadline(arg0 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetWriteDeadline", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetWriteDeadline indicates an expected call of SetWriteDeadline.
func (mr *MockStreamMockRecorder) SetWriteDeadline(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetWriteDeadline", reflect.TypeOf((*MockStream)(nil).SetWriteDeadline), arg0)
}

// Stat mocks base method.
func (m *MockStream) Stat() network.Stats {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stat")
	ret0, _ := ret[0].(network.Stats)
	return ret0
}

// Stat indicates an expected call of Stat.
func (mr *MockStreamMockRecorder) Stat() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stat", reflect.TypeOf((*MockStream)(nil).Stat))
}

// Write mocks base method.
func (m *MockStream) Write(arg0 []byte) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", arg0)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Write indicates an expected call of Write.
func (mr *MockStreamMockRecorder) Write(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockStream)(nil).Write), arg0)
}
