package digest

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (*big.Int, error)
	BestBlockStateRoot() (common.Hash, error)
	BestBlock() (*types.Block, error)
	AddBlock(*types.Block) error
	GetAllBlocksAtDepth(hash common.Hash) []common.Hash
	GetBlockByHash(common.Hash) (*types.Block, error)
	GenesisHash() common.Hash
	GetSlotForBlock(common.Hash) (uint64, error)
	GetFinalizedHeader(uint64, uint64) (*types.Header, error)
	GetFinalizedHash(uint64, uint64) (common.Hash, error)
	SetFinalizedHash(common.Hash, uint64, uint64) error
	RegisterImportedChannel(ch chan<- *types.Block) (byte, error)
	UnregisterImportedChannel(id byte)
	RegisterFinalizedChannel(ch chan<- *types.FinalisationInfo) (byte, error)
	UnregisterFinalizedChannel(id byte)
	HighestCommonAncestor(a, b common.Hash) (common.Hash, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	GetBlockBody(hash common.Hash) (*types.Body, error)
}

// EpochState is the interface for state.EpochState
type EpochState interface {
	GetEpochForBlock(header *types.Header) (uint64, error)
	SetEpochData(epoch uint64, info *types.EpochData) error
	SetConfigData(epoch uint64, info *types.ConfigData) error
	SetCurrentEpoch(epoch uint64) error
	GetCurrentEpoch() (uint64, error)
}

// GrandpaState is the interface for the state.GrandpaState
type GrandpaState interface {
	SetNextChange(authorities []*grandpa.Voter, number *big.Int) error
	IncrementSetID() error
	SetNextPause(number *big.Int) error
	SetNextResume(number *big.Int) error
	GetCurrentSetID() (uint64, error)
}
