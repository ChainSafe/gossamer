package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// BlockState is the interface required by GRANDPA into the block state
type BlockState interface {
	HasHeader(hash common.Hash) (bool, error)
	GetHeader(hash common.Hash) (*types.Header, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)
	HighestCommonPredecessor(a, b common.Hash) (common.Hash, error)
	GetFinalizedHead() (*types.Header, error)
	Leaves() []common.Hash
}
