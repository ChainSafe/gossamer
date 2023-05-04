// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/runtimeinterface"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	AddBlock(*types.Block) error
	GetAllBlocksAtDepth(hash common.Hash) []common.Hash
	GetHeader(common.Hash) (*types.Header, error)
	GetBlockByNumber(blockNumber uint) (*types.Block, error)
	GetBlockHashesBySlot(slot uint64) (blockHashes []common.Hash, err error)
	GenesisHash() common.Hash
	GetSlotForBlock(common.Hash) (uint64, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)
	NumberIsFinalised(blockNumber uint) (bool, error)
	GetRuntime(blockHash common.Hash) (runtime runtimeinterface.Instance, err error)
	StoreRuntime(common.Hash, runtimeinterface.Instance)
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
	PopWithTimer(timerCh <-chan time.Time) (tx *transaction.ValidTransaction)
}

// EpochState is the interface for epoch methods
type EpochState interface {
	GetEpochLength() (uint64, error)
	GetSlotDuration() (time.Duration, error)
	SetCurrentEpoch(epoch uint64) error
	GetCurrentEpoch() (uint64, error)
	SetEpochData(epoch uint64, info *types.EpochData) error

	GetEpochData(epoch uint64, header *types.Header) (*types.EpochData, error)
	GetConfigData(epoch uint64, header *types.Header) (*types.ConfigData, error)

	GetLatestConfigData() (*types.ConfigData, error)
	GetStartSlotForEpoch(epoch uint64) (uint64, error)
	GetEpochForBlock(header *types.Header) (uint64, error)
	SetFirstSlot(slot uint64) error
	GetLatestEpochData() (*types.EpochData, error)
	SkipVerify(*types.Header) (bool, error)
}

// BlockImportHandler is the interface for the handler of new blocks
type BlockImportHandler interface {
	HandleBlockProduced(block *types.Block, state *rtstorage.TrieState) error
}
