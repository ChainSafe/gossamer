// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	big "math/big"

	common "github.com/ChainSafe/gossamer/lib/common"
	mock "github.com/stretchr/testify/mock"

	runtime "github.com/ChainSafe/gossamer/lib/runtime"

	types "github.com/ChainSafe/gossamer/dot/types"
)

// BlockState is an autogenerated mock type for the BlockState type
type BlockState struct {
	mock.Mock
}

// AddBlock provides a mock function with given fields: _a0
func (_m *BlockState) AddBlock(_a0 *types.Block) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Block) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddBlockToBlockTree provides a mock function with given fields: header
func (_m *BlockState) AddBlockToBlockTree(header *types.Header) error {
	ret := _m.Called(header)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Header) error); ok {
		r0 = rf(header)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BestBlockHash provides a mock function with given fields:
func (_m *BlockState) BestBlockHash() common.Hash {
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

// BestBlockHeader provides a mock function with given fields:
func (_m *BlockState) BestBlockHeader() (*types.Header, error) {
	ret := _m.Called()

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func() *types.Header); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
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

// BestBlockNumber provides a mock function with given fields:
func (_m *BlockState) BestBlockNumber() (*big.Int, error) {
	ret := _m.Called()

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func() *big.Int); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
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

// CompareAndSetBlockData provides a mock function with given fields: bd
func (_m *BlockState) CompareAndSetBlockData(bd *types.BlockData) error {
	ret := _m.Called(bd)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.BlockData) error); ok {
		r0 = rf(bd)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAllBlocksAtNumber provides a mock function with given fields: num
func (_m *BlockState) GetAllBlocksAtNumber(num *big.Int) ([]common.Hash, error) {
	ret := _m.Called(num)

	var r0 []common.Hash
	if rf, ok := ret.Get(0).(func(*big.Int) []common.Hash); ok {
		r0 = rf(num)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.Hash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*big.Int) error); ok {
		r1 = rf(num)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockBody provides a mock function with given fields: _a0
func (_m *BlockState) GetBlockBody(_a0 common.Hash) (*types.Body, error) {
	ret := _m.Called(_a0)

	var r0 *types.Body
	if rf, ok := ret.Get(0).(func(common.Hash) *types.Body); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Body)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockByHash provides a mock function with given fields: _a0
func (_m *BlockState) GetBlockByHash(_a0 common.Hash) (*types.Block, error) {
	ret := _m.Called(_a0)

	var r0 *types.Block
	if rf, ok := ret.Get(0).(func(common.Hash) *types.Block); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Block)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockByNumber provides a mock function with given fields: _a0
func (_m *BlockState) GetBlockByNumber(_a0 *big.Int) (*types.Block, error) {
	ret := _m.Called(_a0)

	var r0 *types.Block
	if rf, ok := ret.Get(0).(func(*big.Int) *types.Block); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Block)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*big.Int) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFinalisedNotifierChannel provides a mock function with given fields:
func (_m *BlockState) GetFinalisedNotifierChannel() chan *types.FinalisationInfo {
	ret := _m.Called()

	var r0 chan *types.FinalisationInfo
	if rf, ok := ret.Get(0).(func() chan *types.FinalisationInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan *types.FinalisationInfo)
		}
	}

	return r0
}

// GetHashByNumber provides a mock function with given fields: _a0
func (_m *BlockState) GetHashByNumber(_a0 *big.Int) (common.Hash, error) {
	ret := _m.Called(_a0)

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func(*big.Int) common.Hash); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*big.Int) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHeader provides a mock function with given fields: _a0
func (_m *BlockState) GetHeader(_a0 common.Hash) (*types.Header, error) {
	ret := _m.Called(_a0)

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func(common.Hash) *types.Header); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHeaderByNumber provides a mock function with given fields: num
func (_m *BlockState) GetHeaderByNumber(num *big.Int) (*types.Header, error) {
	ret := _m.Called(num)

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func(*big.Int) *types.Header); ok {
		r0 = rf(num)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*big.Int) error); ok {
		r1 = rf(num)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHighestFinalisedHeader provides a mock function with given fields:
func (_m *BlockState) GetHighestFinalisedHeader() (*types.Header, error) {
	ret := _m.Called()

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func() *types.Header); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
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

// GetJustification provides a mock function with given fields: _a0
func (_m *BlockState) GetJustification(_a0 common.Hash) ([]byte, error) {
	ret := _m.Called(_a0)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(common.Hash) []byte); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMessageQueue provides a mock function with given fields: _a0
func (_m *BlockState) GetMessageQueue(_a0 common.Hash) ([]byte, error) {
	ret := _m.Called(_a0)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(common.Hash) []byte); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetReceipt provides a mock function with given fields: _a0
func (_m *BlockState) GetReceipt(_a0 common.Hash) ([]byte, error) {
	ret := _m.Called(_a0)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(common.Hash) []byte); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRuntime provides a mock function with given fields: _a0
func (_m *BlockState) GetRuntime(_a0 *common.Hash) (runtime.Instance, error) {
	ret := _m.Called(_a0)

	var r0 runtime.Instance
	if rf, ok := ret.Get(0).(func(*common.Hash) runtime.Instance); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(runtime.Instance)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HasBlockBody provides a mock function with given fields: hash
func (_m *BlockState) HasBlockBody(hash common.Hash) (bool, error) {
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

// HasHeader provides a mock function with given fields: hash
func (_m *BlockState) HasHeader(hash common.Hash) (bool, error) {
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

// IsDescendantOf provides a mock function with given fields: parent, child
func (_m *BlockState) IsDescendantOf(parent common.Hash, child common.Hash) (bool, error) {
	ret := _m.Called(parent, child)

	var r0 bool
	if rf, ok := ret.Get(0).(func(common.Hash, common.Hash) bool); ok {
		r0 = rf(parent, child)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash, common.Hash) error); ok {
		r1 = rf(parent, child)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetFinalisedHash provides a mock function with given fields: hash, round, setID
func (_m *BlockState) SetFinalisedHash(hash common.Hash, round uint64, setID uint64) error {
	ret := _m.Called(hash, round, setID)

	var r0 error
	if rf, ok := ret.Get(0).(func(common.Hash, uint64, uint64) error); ok {
		r0 = rf(hash, round, setID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetHeader provides a mock function with given fields: _a0
func (_m *BlockState) SetHeader(_a0 *types.Header) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Header) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetJustification provides a mock function with given fields: hash, data
func (_m *BlockState) SetJustification(hash common.Hash, data []byte) error {
	ret := _m.Called(hash, data)

	var r0 error
	if rf, ok := ret.Get(0).(func(common.Hash, []byte) error); ok {
		r0 = rf(hash, data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StoreRuntime provides a mock function with given fields: _a0, _a1
func (_m *BlockState) StoreRuntime(_a0 common.Hash, _a1 runtime.Instance) {
	_m.Called(_a0, _a1)
}

// SubChain provides a mock function with given fields: start, end
func (_m *BlockState) SubChain(start common.Hash, end common.Hash) ([]common.Hash, error) {
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
