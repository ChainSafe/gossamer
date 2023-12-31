// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/digest (interfaces: Telemetry)
//
// Generated by this command:
//
//	mockgen -destination=mock_telemetry_test.go -package digest . Telemetry
//

// Package digest is a generated GoMock package.
package digest

import (
	json "encoding/json"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockTelemetry is a mock of Telemetry interface.
type MockTelemetry struct {
	ctrl     *gomock.Controller
	recorder *MockTelemetryMockRecorder
}

// MockTelemetryMockRecorder is the mock recorder for MockTelemetry.
type MockTelemetryMockRecorder struct {
	mock *MockTelemetry
}

// NewMockTelemetry creates a new mock instance.
func NewMockTelemetry(ctrl *gomock.Controller) *MockTelemetry {
	mock := &MockTelemetry{ctrl: ctrl}
	mock.recorder = &MockTelemetryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTelemetry) EXPECT() *MockTelemetryMockRecorder {
	return m.recorder
}

// SendMessage mocks base method.
func (m *MockTelemetry) SendMessage(arg0 json.Marshaler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendMessage", arg0)
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockTelemetryMockRecorder) SendMessage(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockTelemetry)(nil).SendMessage), arg0)
}
