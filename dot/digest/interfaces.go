// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// BlockState interface for block state methods
type BlockState interface {
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
}

// EpochState is the interface for state.EpochState
type EpochState interface {
	GetEpochForBlock(header *types.Header) (uint64, error)
	StoreBABENextEpochData(epoch uint64, hash common.Hash, nextEpochData types.NextEpochData)
	StoreBABENextConfigData(epoch uint64, hash common.Hash, nextEpochData types.NextConfigData)
	FinalizeBABENextEpochData(finalizedHeader *types.Header) error
	FinalizeBABENextConfigData(finalizedHeader *types.Header) error
}

// GrandpaState is the interface for the state.GrandpaState
type GrandpaState interface {
	HandleGRANDPADigest(header *types.Header, digest scale.VaryingDataType) error
	ApplyScheduledChanges(finalizedHeader *types.Header) error
	ApplyForcedChanges(importedHeader *types.Header) error
}

// logger logs messages at the debug or error level.
type logger interface {
	Error(s string)
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}
