// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

const NextRandomnessKey = "0x1cb6f36e027abb2091cfb5110ab5087f7ce678799d3eff024253b90e84927cc6"
const NextAuthoritiesKey = "0x1cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4c"
const EpochIndexKey = "0x1cb6f36e027abb2091cfb5110ab5087f38316cbf8fa0da822a20ac1c55bf1be3"

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
	HandleBlockProduced(block *types.Block, state rtstorage.TrieState) error
}

func GetNextEpochDataRawFromState(state trie.Trie) (*types.EpochDataRaw, error) {
	nextRandomnessBytes := state.Get(common.MustHexToBytes(NextRandomnessKey))
	if nextRandomnessBytes == nil {
		return nil, fmt.Errorf("next babe randomness not found in new state")
	}

	var nextRandomness [types.RandomnessLength]byte
	err := scale.Unmarshal(nextRandomnessBytes, &nextRandomness)
	if err != nil {
		return nil, err
	}

	nextAuthoritiesBytes := state.Get(common.MustHexToBytes(NextAuthoritiesKey))
	if nextAuthoritiesBytes == nil {
		return nil, fmt.Errorf("next babe authorities not found in new state")
	}

	var nextAuthorities []types.AuthorityRaw
	err = scale.Unmarshal(nextAuthoritiesBytes, &nextAuthorities)
	if err != nil {
		return nil, err
	}

	return &types.EpochDataRaw{
		Randomness:  nextRandomness,
		Authorities: nextAuthorities,
	}, nil
}

func GetCurrentEpochIndexFromState(state trie.Trie) (uint64, error) {
	epochIndexBytes := state.Get(common.MustHexToBytes(EpochIndexKey))
	if epochIndexBytes == nil {
		return 0, fmt.Errorf("babe epoch index not found in new state")
	}

	var epochIndex uint64
	err := scale.Unmarshal(epochIndexBytes, &epochIndex)
	if err != nil {
		return 0, err
	}

	return epochIndex, nil
}
