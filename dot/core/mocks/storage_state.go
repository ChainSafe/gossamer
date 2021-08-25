// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	common "github.com/ChainSafe/gossamer/lib/common"

	mock "github.com/stretchr/testify/mock"

	storage "github.com/ChainSafe/gossamer/lib/runtime/storage"

	types "github.com/ChainSafe/gossamer/dot/types"
)

// MockStorageState is an autogenerated mock type for the StorageState type
type MockStorageState struct {
	mock.Mock
}

// GetStateRootFromBlock provides a mock function with given fields: bhash
func (_m *MockStorageState) GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error) {
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
func (_m *MockStorageState) GetStorage(root *common.Hash, key []byte) ([]byte, error) {
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

// LoadCode provides a mock function with given fields: root
func (_m *MockStorageState) LoadCode(root *common.Hash) ([]byte, error) {
	ret := _m.Called(root)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(*common.Hash) []byte); ok {
		r0 = rf(root)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
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

// LoadCodeHash provides a mock function with given fields: root
func (_m *MockStorageState) LoadCodeHash(root *common.Hash) (common.Hash, error) {
	ret := _m.Called(root)

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func(*common.Hash) common.Hash); ok {
		r0 = rf(root)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
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

// StoreTrie provides a mock function with given fields: _a0, _a1
func (_m *MockStorageState) StoreTrie(_a0 *storage.TrieState, _a1 *types.Header) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*storage.TrieState, *types.Header) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// TrieState provides a mock function with given fields: root
func (_m *MockStorageState) TrieState(root *common.Hash) (*storage.TrieState, error) {
	ret := _m.Called(root)

	var r0 *storage.TrieState
	if rf, ok := ret.Get(0).(func(*common.Hash) *storage.TrieState); ok {
		r0 = rf(root)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*storage.TrieState)
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
