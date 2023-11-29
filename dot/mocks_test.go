// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot (interfaces: ServiceRegisterer)

// Package dot is a generated GoMock package.
package dot

import (
	reflect "reflect"

	services "github.com/ChainSafe/gossamer/lib/services"
	gomock "go.uber.org/mock/gomock"
)

// MockServiceRegisterer is a mock of ServiceRegisterer interface.
type MockServiceRegisterer struct {
	ctrl     *gomock.Controller
	recorder *MockServiceRegistererMockRecorder
}

// MockServiceRegistererMockRecorder is the mock recorder for MockServiceRegisterer.
type MockServiceRegistererMockRecorder struct {
	mock *MockServiceRegisterer
}

// NewMockServiceRegisterer creates a new mock instance.
func NewMockServiceRegisterer(ctrl *gomock.Controller) *MockServiceRegisterer {
	mock := &MockServiceRegisterer{ctrl: ctrl}
	mock.recorder = &MockServiceRegistererMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServiceRegisterer) EXPECT() *MockServiceRegistererMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockServiceRegisterer) Get(arg0 interface{}) services.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(services.Service)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockServiceRegistererMockRecorder) Get(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockServiceRegisterer)(nil).Get), arg0)
}

// RegisterService mocks base method.
func (m *MockServiceRegisterer) RegisterService(arg0 services.Service) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterService", arg0)
}

// RegisterService indicates an expected call of RegisterService.
func (mr *MockServiceRegistererMockRecorder) RegisterService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterService", reflect.TypeOf((*MockServiceRegisterer)(nil).RegisterService), arg0)
}

// StartAll mocks base method.
func (m *MockServiceRegisterer) StartAll() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "StartAll")
}

// StartAll indicates an expected call of StartAll.
func (mr *MockServiceRegistererMockRecorder) StartAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartAll", reflect.TypeOf((*MockServiceRegisterer)(nil).StartAll))
}

// StopAll mocks base method.
func (m *MockServiceRegisterer) StopAll() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "StopAll")
}

// StopAll indicates an expected call of StopAll.
func (mr *MockServiceRegistererMockRecorder) StopAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopAll", reflect.TypeOf((*MockServiceRegisterer)(nil).StopAll))
}
