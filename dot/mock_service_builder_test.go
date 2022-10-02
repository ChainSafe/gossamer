// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot (interfaces: ServiceBuilder)

// Package dot is a generated GoMock package.
package dot

import (
	reflect "reflect"

	babe "github.com/ChainSafe/gossamer/lib/babe"
	gomock "github.com/golang/mock/gomock"
)

// MockServiceBuilder is a mock of ServiceBuilder interface.
type MockServiceBuilder struct {
	ctrl     *gomock.Controller
	recorder *MockServiceBuilderMockRecorder
}

// MockServiceBuilderMockRecorder is the mock recorder for MockServiceBuilder.
type MockServiceBuilderMockRecorder struct {
	mock *MockServiceBuilder
}

// NewMockServiceBuilder creates a new mock instance.
func NewMockServiceBuilder(ctrl *gomock.Controller) *MockServiceBuilder {
	mock := &MockServiceBuilder{ctrl: ctrl}
	mock.recorder = &MockServiceBuilderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServiceBuilder) EXPECT() *MockServiceBuilderMockRecorder {
	return m.recorder
}

// NewServiceIFace mocks base method.
func (m *MockServiceBuilder) NewServiceIFace(arg0 *babe.ServiceConfig) (*babe.Service, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewServiceIFace", arg0)
	ret0, _ := ret[0].(*babe.Service)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewServiceIFace indicates an expected call of NewServiceIFace.
func (mr *MockServiceBuilderMockRecorder) NewServiceIFace(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewServiceIFace", reflect.TypeOf((*MockServiceBuilder)(nil).NewServiceIFace), arg0)
}
