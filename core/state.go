package core

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
)

// BlockState holds block state methods
type BlockState interface {
	LatestHeader() *types.Header
	GetBlockData(hash common.Hash) (*types.BlockData, error)
	SetBlockData(blockData *types.BlockData) error
	AddBlock(*types.Block) error
	SetBlock(*types.Block) error
	GetHeader(common.Hash) (*types.Header, error)
	SetHeader(*types.Header) error
}

// StorageState holds storage state methods
type StorageState interface {
	StorageRoot() (common.Hash, error)
}
