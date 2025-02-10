// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

type SlotState interface {
	CheckEquivocation(slotNow, slot uint64, header *types.Header,
		signer types.AuthorityID) (*types.BabeEquivocationProof, error)
}

// ImportedBlockNotifierManager is the interface for block notification channels
type ImportedBlockNotifierManager interface {
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
}

// TransactionState is the interface for transaction queue methods
type TransactionState interface {
	Push(vt *transaction.ValidTransaction) (common.Hash, error)
	PopWithTimer(timerCh <-chan time.Time) (tx *transaction.ValidTransaction)
}

// EpochState is the interface for epoch methods
type EpochState interface {
	GetEpochLength() uint64
	GetSlotDuration() (time.Duration, error)
	StoreCurrentEpoch(epoch uint64) error
	GetCurrentEpoch() (uint64, error)

	GetSkippedEpochDataRaw(skippedEpoch, currentEpoch uint64, header *types.Header) (
		*types.EpochDataRaw, error)
	GetSkippedConfigData(skippedEpoch, currentEpoch uint64, header *types.Header) (
		*types.ConfigData, error)

	GetEpochDataRaw(epoch uint64, header *types.Header) (*types.EpochDataRaw, error)
	GetConfigData(epoch uint64, header *types.Header) (*types.ConfigData, error)

	GetStartSlotForEpoch(epoch uint64, bestBlockHash common.Hash) (uint64, error)
	GetEpochForBlock(header *types.Header) (uint64, error)
	SkipVerify(*types.Header) (bool, error)
}

// BlockImportHandler is the interface for the handler of new blocks
type BlockImportHandler interface {
	HandleBlockProduced(block *types.Block, state *rtstorage.TrieState) error
}
