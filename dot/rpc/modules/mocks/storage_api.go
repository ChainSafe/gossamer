// Code generated by mockery v2.14.1. DO NOT EDIT.

package mocks

import (
	common "github.com/ChainSafe/gossamer/lib/common"
	mock "github.com/stretchr/testify/mock"

	state "github.com/ChainSafe/gossamer/dot/state"

	trie "github.com/ChainSafe/gossamer/lib/trie"
)

// StorageAPI is an autogenerated mock type for the StorageAPI type
type StorageAPI struct {
	mock.Mock
}

// Entries provides a mock function with given fields: root
func (_m *StorageAPI) Entries(root *common.Hash) (map[string][]byte, error) {
	ret := _m.Called(root)

	var r0 map[string][]byte
	if rf, ok := ret.Get(0).(func(*common.Hash) map[string][]byte); ok {
		r0 = rf(root)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*common.Hash) error); ok {
		r1 = rf(root)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetKeysWithPrefix provides a mock function with given fields: root, prefix
func (_m *StorageAPI) GetKeysWithPrefix(root *common.Hash, prefix []byte) ([][]byte, error) {
	ret := _m.Called(root, prefix)

	var r0 [][]byte
	if rf, ok := ret.Get(0).(func(*common.Hash, []byte) [][]byte); ok {
		r0 = rf(root, prefix)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([][]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*common.Hash, []byte) error); ok {
		r1 = rf(root, prefix)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStateRootFromBlock provides a mock function with given fields: bhash
func (_m *StorageAPI) GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error) {
	ret := _m.Called(bhash)

	var r0 *common.Hash
	if rf, ok := ret.Get(0).(func(*common.Hash) *common.Hash); ok {
		r0 = rf(bhash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*common.Hash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*common.Hash) error); ok {
		r1 = rf(bhash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStorage provides a mock function with given fields: root, key
func (_m *StorageAPI) GetStorage(root *common.Hash, key []byte) ([]byte, error) {
	ret := _m.Called(root, key)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(*common.Hash, []byte) []byte); ok {
		r0 = rf(root, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*common.Hash, []byte) error); ok {
		r1 = rf(root, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStorageByBlockHash provides a mock function with given fields: bhash, key
func (_m *StorageAPI) GetStorageByBlockHash(bhash *common.Hash, key []byte) ([]byte, error) {
	ret := _m.Called(bhash, key)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(*common.Hash, []byte) []byte); ok {
		r0 = rf(bhash, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*common.Hash, []byte) error); ok {
		r1 = rf(bhash, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStorageChild provides a mock function with given fields: root, keyToChild
func (_m *StorageAPI) GetStorageChild(root *common.Hash, keyToChild []byte) (*trie.Trie, error) {
	ret := _m.Called(root, keyToChild)

	var r0 *trie.Trie
	if rf, ok := ret.Get(0).(func(*common.Hash, []byte) *trie.Trie); ok {
		r0 = rf(root, keyToChild)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*trie.Trie)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*common.Hash, []byte) error); ok {
		r1 = rf(root, keyToChild)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStorageFromChild provides a mock function with given fields: root, keyToChild, key
func (_m *StorageAPI) GetStorageFromChild(root *common.Hash, keyToChild []byte, key []byte) ([]byte, error) {
	ret := _m.Called(root, keyToChild, key)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(*common.Hash, []byte, []byte) []byte); ok {
		r0 = rf(root, keyToChild, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*common.Hash, []byte, []byte) error); ok {
		r1 = rf(root, keyToChild, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RegisterStorageObserver provides a mock function with given fields: observer
func (_m *StorageAPI) RegisterStorageObserver(observer state.Observer) {
	_m.Called(observer)
}

// UnregisterStorageObserver provides a mock function with given fields: observer
func (_m *StorageAPI) UnregisterStorageObserver(observer state.Observer) {
	_m.Called(observer)
}

type mockConstructorTestingTNewStorageAPI interface {
	mock.TestingT
	Cleanup(func())
}

// NewStorageAPI creates a new instance of StorageAPI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewStorageAPI(t mockConstructorTestingTNewStorageAPI) *StorageAPI {
	mock := &StorageAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
