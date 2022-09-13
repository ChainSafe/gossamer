// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"context"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

//go:generate mockgen -destination=./mock_state_test.go -package $GOPACKAGE . BlockState,ImportedBlockNotifierManager,StorageState,TransactionState,EpochState,DigestHandler,BlockImportHandler

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (blockNumber uint, err error)
	BestBlock() (*types.Block, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	AddBlock(*types.Block) error
	GetAllBlocksAtDepth(hash common.Hash) []common.Hash
	GetHeader(common.Hash) (*types.Header, error)
	GetBlockByNumber(blockNumber uint) (*types.Block, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetArrivalTime(common.Hash) (time.Time, error)
	GenesisHash() common.Hash
	GetSlotForBlock(common.Hash) (uint64, error)
	GetFinalisedHeader(uint64, uint64) (*types.Header, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)
	NumberIsFinalised(blockNumber uint) (bool, error)
	GetRuntime(*common.Hash) (runtime.Instance, error)
	StoreRuntime(common.Hash, runtime.Instance)
	ImportedBlockNotifierManager
}

// ImportedBlockNotifierManager is the interface for block notification channels
type ImportedBlockNotifierManager interface {
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
}

// StorageState interface for storage state methods
type StorageState interface {
	TrieState(hash *common.Hash) (*rtstorage.TrieState, error)
	sync.Locker
}

// TransactionState is the interface for transaction queue methods
type TransactionState interface {
	Push(vt *transaction.ValidTransaction) (common.Hash, error)
	Pop(ctx context.Context) *transaction.ValidTransaction
	Peek() *transaction.ValidTransaction
}

// EpochState is the interface for epoch methods
type EpochState interface {
	GetEpochLength() (uint64, error)
	GetSlotDuration() (time.Duration, error)
	SetCurrentEpoch(epoch uint64) error
	GetCurrentEpoch() (uint64, error)
	SetEpochData(uint64, *types.EpochData) error

	GetEpochData(epoch uint64, header *types.Header) (*types.EpochData, error)
	GetConfigData(epoch uint64, header *types.Header) (*types.ConfigData, error)

	GetLatestConfigData() (*types.ConfigData, error)
	GetStartSlotForEpoch(epoch uint64) (uint64, error)
	GetEpochForBlock(header *types.Header) (uint64, error)
	SetFirstSlot(slot uint64) error
	GetLatestEpochData() (*types.EpochData, error)
	SkipVerify(*types.Header) (bool, error)
	GetEpochFromTime(time.Time) (uint64, error)
}

// DigestHandler is the interface for the consensus digest handler
type DigestHandler interface {
	HandleDigests(*types.Header)
}

//go:generate mockery --name BlockImportHandler --structname BlockImportHandler --case underscore --keeptree

// BlockImportHandler is the interface for the handler of new blocks
type BlockImportHandler interface {
	HandleBlockProduced(block *types.Block, state *rtstorage.TrieState) error
}
