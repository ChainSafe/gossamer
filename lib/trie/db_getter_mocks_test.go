// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/lib/trie/db (interfaces: DBGetter)
//
// Generated by this command:
//
//	mockgen -destination=db_getter_mocks_test.go -package=trie github.com/ChainSafe/gossamer/lib/trie/db DBGetter
//
// Package trie is a generated GoMock package.
package trie

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockDBGetter is a mock of DBGetter interface.
type MockDBGetter struct {
	ctrl     *gomock.Controller
	recorder *MockDBGetterMockRecorder
}

// MockDBGetterMockRecorder is the mock recorder for MockDBGetter.
type MockDBGetterMockRecorder struct {
	mock *MockDBGetter
}

// NewMockDBGetter creates a new mock instance.
func NewMockDBGetter(ctrl *gomock.Controller) *MockDBGetter {
	mock := &MockDBGetter{ctrl: ctrl}
	mock.recorder = &MockDBGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDBGetter) EXPECT() *MockDBGetterMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockDBGetter) Get(arg0 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockDBGetterMockRecorder) Get(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockDBGetter)(nil).Get), arg0)
}
