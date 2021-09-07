// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	types "github.com/ChainSafe/gossamer/dot/types"
	storage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	mock "github.com/stretchr/testify/mock"
)

// BlockImportHandler is an autogenerated mock type for the BlockImportHandler type
type BlockImportHandler struct {
	mock.Mock
}

func (_m *BlockImportHandler) HandleBlockProducedVdt(block *types.Block, state *storage.TrieState) error {
	ret := _m.Called(block, state)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Block, *storage.TrieState) error); ok {
		r0 = rf(block, state)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
