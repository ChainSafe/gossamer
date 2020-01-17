package core

import (
	"github.com/ChainSafe/gossamer/core/types"
)

type BlockState interface {
	GetLatestBlockHeader() *types.BlockHeader
	AddBlock(types.BlockHeader) error
}
