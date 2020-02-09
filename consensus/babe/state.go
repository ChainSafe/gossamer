package babe

import (
	"github.com/ChainSafe/gossamer/core/types"
)

// BlockState is the interface that holds methods to interact with the Block state
type BlockState interface {
	LatestHeader() *types.Header
	AddBlock(*types.Block) error
}
