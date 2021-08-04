// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	core "github.com/ChainSafe/gossamer/dot/core"
	common "github.com/ChainSafe/gossamer/lib/common"

	crypto "github.com/ChainSafe/gossamer/lib/crypto"

	mock "github.com/stretchr/testify/mock"

	runtime "github.com/ChainSafe/gossamer/lib/runtime"

	types "github.com/ChainSafe/gossamer/dot/types"
)

// MockCoreAPI is an autogenerated mock type for the CoreAPI type
type MockCoreAPI struct {
	mock.Mock
}

// GetMetadata provides a mock function with given fields: bhash
func (_m *MockCoreAPI) GetMetadata(bhash *common.Hash) ([]byte, error) {
	ret := _m.Called(bhash)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(*common.Hash) []byte); ok {
		r0 = rf(bhash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
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

// GetRuntimeVersion provides a mock function with given fields: bhash
func (_m *MockCoreAPI) GetRuntimeVersion(bhash *common.Hash) (runtime.Version, error) {
	ret := _m.Called(bhash)

	var r0 runtime.Version
	if rf, ok := ret.Get(0).(func(*common.Hash) runtime.Version); ok {
		r0 = rf(bhash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(runtime.Version)
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

// HandleSubmittedExtrinsic provides a mock function with given fields: _a0
func (_m *MockCoreAPI) HandleSubmittedExtrinsic(_a0 types.Extrinsic) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(types.Extrinsic) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// HasKey provides a mock function with given fields: pubKeyStr, keyType
func (_m *MockCoreAPI) HasKey(pubKeyStr string, keyType string) (bool, error) {
	ret := _m.Called(pubKeyStr, keyType)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, string) bool); ok {
		r0 = rf(pubKeyStr, keyType)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(pubKeyStr, keyType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InsertKey provides a mock function with given fields: kp
func (_m *MockCoreAPI) InsertKey(kp crypto.Keypair) {
	_m.Called(kp)
}

// QueryStorage provides a mock function with given fields: from, to, keys
func (_m *MockCoreAPI) QueryStorage(from common.Hash, to *common.Hash, keys ...string) (map[common.Hash]core.QueryKeyValueChanges, error) {
	_va := make([]interface{}, len(keys))
	for _i := range keys {
		_va[_i] = keys[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, from, to)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 map[common.Hash]core.QueryKeyValueChanges
	if rf, ok := ret.Get(0).(func(common.Hash, *common.Hash, ...string) map[common.Hash]core.QueryKeyValueChanges); ok {
		r0 = rf(from, to, keys...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[common.Hash]core.QueryKeyValueChanges)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash, *common.Hash, ...string) error); ok {
		r1 = rf(from, to, keys...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
