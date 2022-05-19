// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ChainSafe/gossamer/dot/rpc/modules (interfaces: StorageAPI)

// Package modules is a generated GoMock package.
package modules

import (
	reflect "reflect"

	state "github.com/ChainSafe/gossamer/dot/state"
	common "github.com/ChainSafe/gossamer/lib/common"
	trie "github.com/ChainSafe/gossamer/lib/trie"
	gomock "github.com/golang/mock/gomock"
)

// MockStorageAPI is a mock of StorageAPI interface.
type MockStorageAPI struct {
	ctrl     *gomock.Controller
	recorder *MockStorageAPIMockRecorder
}

// MockStorageAPIMockRecorder is the mock recorder for MockStorageAPI.
type MockStorageAPIMockRecorder struct {
	mock *MockStorageAPI
}

// NewMockStorageAPI creates a new mock instance.
func NewMockStorageAPI(ctrl *gomock.Controller) *MockStorageAPI {
	mock := &MockStorageAPI{ctrl: ctrl}
	mock.recorder = &MockStorageAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageAPI) EXPECT() *MockStorageAPIMockRecorder {
	return m.recorder
}

// Entries mocks base method.
func (m *MockStorageAPI) Entries(arg0 *common.Hash) (map[string][]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Entries", arg0)
	ret0, _ := ret[0].(map[string][]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Entries indicates an expected call of Entries.
func (mr *MockStorageAPIMockRecorder) Entries(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Entries", reflect.TypeOf((*MockStorageAPI)(nil).Entries), arg0)
}

// GetKeysWithPrefix mocks base method.
func (m *MockStorageAPI) GetKeysWithPrefix(arg0 *common.Hash, arg1 []byte) ([][]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKeysWithPrefix", arg0, arg1)
	ret0, _ := ret[0].([][]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetKeysWithPrefix indicates an expected call of GetKeysWithPrefix.
func (mr *MockStorageAPIMockRecorder) GetKeysWithPrefix(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKeysWithPrefix", reflect.TypeOf((*MockStorageAPI)(nil).GetKeysWithPrefix), arg0, arg1)
}

// GetStateRootFromBlock mocks base method.
func (m *MockStorageAPI) GetStateRootFromBlock(arg0 *common.Hash) (*common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStateRootFromBlock", arg0)
	ret0, _ := ret[0].(*common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStateRootFromBlock indicates an expected call of GetStateRootFromBlock.
func (mr *MockStorageAPIMockRecorder) GetStateRootFromBlock(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStateRootFromBlock", reflect.TypeOf((*MockStorageAPI)(nil).GetStateRootFromBlock), arg0)
}

// GetStorage mocks base method.
func (m *MockStorageAPI) GetStorage(arg0 *common.Hash, arg1 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorage", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorage indicates an expected call of GetStorage.
func (mr *MockStorageAPIMockRecorder) GetStorage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorage", reflect.TypeOf((*MockStorageAPI)(nil).GetStorage), arg0, arg1)
}

// GetStorageByBlockHash mocks base method.
func (m *MockStorageAPI) GetStorageByBlockHash(arg0 *common.Hash, arg1 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorageByBlockHash", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorageByBlockHash indicates an expected call of GetStorageByBlockHash.
func (mr *MockStorageAPIMockRecorder) GetStorageByBlockHash(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageByBlockHash", reflect.TypeOf((*MockStorageAPI)(nil).GetStorageByBlockHash), arg0, arg1)
}

// GetStorageChild mocks base method.
func (m *MockStorageAPI) GetStorageChild(arg0 *common.Hash, arg1 []byte) (*trie.Trie, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorageChild", arg0, arg1)
	ret0, _ := ret[0].(*trie.Trie)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorageChild indicates an expected call of GetStorageChild.
func (mr *MockStorageAPIMockRecorder) GetStorageChild(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageChild", reflect.TypeOf((*MockStorageAPI)(nil).GetStorageChild), arg0, arg1)
}

// GetStorageFromChild mocks base method.
func (m *MockStorageAPI) GetStorageFromChild(arg0 *common.Hash, arg1, arg2 []byte) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorageFromChild", arg0, arg1, arg2)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorageFromChild indicates an expected call of GetStorageFromChild.
func (mr *MockStorageAPIMockRecorder) GetStorageFromChild(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageFromChild", reflect.TypeOf((*MockStorageAPI)(nil).GetStorageFromChild), arg0, arg1, arg2)
}

// RegisterStorageObserver mocks base method.
func (m *MockStorageAPI) RegisterStorageObserver(arg0 state.Observer) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterStorageObserver", arg0)
}

// RegisterStorageObserver indicates an expected call of RegisterStorageObserver.
func (mr *MockStorageAPIMockRecorder) RegisterStorageObserver(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterStorageObserver", reflect.TypeOf((*MockStorageAPI)(nil).RegisterStorageObserver), arg0)
}

// UnregisterStorageObserver mocks base method.
func (m *MockStorageAPI) UnregisterStorageObserver(arg0 state.Observer) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UnregisterStorageObserver", arg0)
}

// UnregisterStorageObserver indicates an expected call of UnregisterStorageObserver.
func (mr *MockStorageAPIMockRecorder) UnregisterStorageObserver(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnregisterStorageObserver", reflect.TypeOf((*MockStorageAPI)(nil).UnregisterStorageObserver), arg0)
}
