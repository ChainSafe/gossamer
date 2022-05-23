// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHeader() (*types.Header, error)
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
}

//go:generate mockgen -destination=mock_epoch_state_test.go -package $GOPACKAGE . EpochState

// EpochState is the interface for state.EpochState
type EpochState interface {
	GetEpochForBlock(header *types.Header) (uint64, error)
	SetEpochData(epoch uint64, info *types.EpochData) error
	SetConfigData(epoch uint64, info *types.ConfigData) error

	StoreBABENextEpochData(epoch uint64, hash common.Hash, nextEpochData types.NextEpochData)
	StoreBABENextConfigData(epoch uint64, hash common.Hash, nextEpochData types.NextConfigData)
	FinalizeBABENextEpochData(finalizedHeader *types.Header) error
	FinalizeBABENextConfigData(finalizedHeader *types.Header) error
}

// GrandpaState is the interface for the state.GrandpaState
type GrandpaState interface {
	SetNextChange(authorities []grandpa.Voter, number uint) error
	IncrementSetID() (newSetID uint64, err error)
	SetNextPause(number uint) error
	SetNextResume(number uint) error
	GetCurrentSetID() (uint64, error)

	HandleGRANDPADigest(header *types.Header, digest scale.VaryingDataType) error
	ApplyScheduledChanges(finalizedHeader *types.Header) error
	ApplyForcedChanges(importedHeader *types.Header) error
}
