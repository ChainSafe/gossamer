package core

import (
	"github.com/ChainSafe/gossamer/core/types"
)

type BlockState interface {
	GetLatestBlockHeader() *types.Header
	AddBlock(types.Header) error
}
