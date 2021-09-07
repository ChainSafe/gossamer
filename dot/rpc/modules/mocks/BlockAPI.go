// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	big "math/big"

	common "github.com/ChainSafe/gossamer/lib/common"
	mock "github.com/stretchr/testify/mock"

	runtime "github.com/ChainSafe/gossamer/lib/runtime"

	types "github.com/ChainSafe/gossamer/dot/types"
)

// BlockAPI is an autogenerated mock type for the BlockAPI type
type BlockAPI struct {
	mock.Mock
}

// BestBlockHash provides a mock function with given fields:
func (_m *BlockAPI) BestBlockHash() common.Hash {
	ret := _m.Called()

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func() common.Hash); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	return r0
}

// GetBlockByHash provides a mock function with given fields: hash
func (_m *BlockAPI) GetBlockByHashVdt(hash common.Hash) (*types.BlockVdt, error) {
	ret := _m.Called(hash)

	var r0 *types.BlockVdt
	if rf, ok := ret.Get(0).(func(common.Hash) *types.BlockVdt); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockVdt)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

//// GetBlockByHash provides a mock function with given fields: hash
//func (_m *BlockAPI) GetBlockByHash(hash common.Hash) (*types.Block, error) {
//	ret := _m.Called(hash)
//
//	var r0 *types.Block
//	if rf, ok := ret.Get(0).(func(common.Hash) *types.Block); ok {
//		r0 = rf(hash)
//	} else {
//		if ret.Get(0) != nil {
//			r0 = ret.Get(0).(*types.Block)
//		}
//	}
//
//	var r1 error
//	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
//		r1 = rf(hash)
//	} else {
//		r1 = ret.Error(1)
//	}
//
//	return r0, r1
//}

// GetBlockHash provides a mock function with given fields: blockNumber
func (_m *BlockAPI) GetBlockHash(blockNumber *big.Int) (common.Hash, error) {
	ret := _m.Called(blockNumber)

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func(*big.Int) common.Hash); ok {
		r0 = rf(blockNumber)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*big.Int) error); ok {
		r1 = rf(blockNumber)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFinalisedHash provides a mock function with given fields: _a0, _a1
func (_m *BlockAPI) GetFinalisedHash(_a0 uint64, _a1 uint64) (common.Hash, error) {
	ret := _m.Called(_a0, _a1)

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func(uint64, uint64) common.Hash); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint64, uint64) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHeader provides a mock function with given fields: hash
func (_m *BlockAPI) GetHeaderVdt(hash common.Hash) (*types.HeaderVdt, error) {
	ret := _m.Called(hash)

	var r0 *types.HeaderVdt
	if rf, ok := ret.Get(0).(func(common.Hash) *types.HeaderVdt); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.HeaderVdt)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHeader provides a mock function with given fields: hash
func (_m *BlockAPI) GetHeader(hash common.Hash) (*types.Header, error) {
	ret := _m.Called(hash)

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func(common.Hash) *types.Header); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHighestFinalisedHash provides a mock function with given fields:
func (_m *BlockAPI) GetHighestFinalisedHash() (common.Hash, error) {
	ret := _m.Called()

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func() common.Hash); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetJustification provides a mock function with given fields: hash
func (_m *BlockAPI) GetJustification(hash common.Hash) ([]byte, error) {
	ret := _m.Called(hash)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(common.Hash) []byte); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HasJustification provides a mock function with given fields: hash
func (_m *BlockAPI) HasJustification(hash common.Hash) (bool, error) {
	ret := _m.Called(hash)

	var r0 bool
	if rf, ok := ret.Get(0).(func(common.Hash) bool); ok {
		r0 = rf(hash)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RegisterFinalizedChannel provides a mock function with given fields: ch
func (_m *BlockAPI) RegisterFinalizedChannel(ch chan<- *types.FinalisationInfoVdt) (byte, error) {
	ret := _m.Called(ch)

	var r0 byte
	if rf, ok := ret.Get(0).(func(chan<- *types.FinalisationInfoVdt) byte); ok {
		r0 = rf(ch)
	} else {
		r0 = ret.Get(0).(byte)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(chan<- *types.FinalisationInfoVdt) error); ok {
		r1 = rf(ch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RegisterImportedChannel provides a mock function with given fields: ch
func (_m *BlockAPI) RegisterImportedChannel(ch chan<- *types.BlockVdt) (byte, error) {
	ret := _m.Called(ch)

	var r0 byte
	if rf, ok := ret.Get(0).(func(chan<- *types.BlockVdt) byte); ok {
		r0 = rf(ch)
	} else {
		r0 = ret.Get(0).(byte)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(chan<- *types.BlockVdt) error); ok {
		r1 = rf(ch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RegisterRuntimeUpdatedChannel provides a mock function with given fields: ch
func (_m *BlockAPI) RegisterRuntimeUpdatedChannel(ch chan<- runtime.Version) (uint32, error) {
	ret := _m.Called(ch)

	var r0 uint32
	if rf, ok := ret.Get(0).(func(chan<- runtime.Version) uint32); ok {
		r0 = rf(ch)
	} else {
		r0 = ret.Get(0).(uint32)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(chan<- runtime.Version) error); ok {
		r1 = rf(ch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SubChain provides a mock function with given fields: start, end
func (_m *BlockAPI) SubChain(start common.Hash, end common.Hash) ([]common.Hash, error) {
	ret := _m.Called(start, end)

	var r0 []common.Hash
	if rf, ok := ret.Get(0).(func(common.Hash, common.Hash) []common.Hash); ok {
		r0 = rf(start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.Hash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash, common.Hash) error); ok {
		r1 = rf(start, end)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UnregisterFinalisedChannel provides a mock function with given fields: id
func (_m *BlockAPI) UnregisterFinalisedChannel(id byte) {
	_m.Called(id)
}

// UnregisterImportedChannel provides a mock function with given fields: id
func (_m *BlockAPI) UnregisterImportedChannel(id byte) {
	_m.Called(id)
}

// UnregisterRuntimeUpdatedChannel provides a mock function with given fields: id
func (_m *BlockAPI) UnregisterRuntimeUpdatedChannel(id uint32) bool {
	ret := _m.Called(id)

	var r0 bool
	if rf, ok := ret.Get(0).(func(uint32) bool); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
