// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/dot/types"
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
	HandleBABEDigest(header *types.Header, digest types.BabeConsensusDigest) error
	FinalizeBABENextEpochData(finalizedHeader *types.Header) error
	FinalizeBABENextConfigData(finalizedHeader *types.Header) error
}

// GrandpaState is the interface for the state.GrandpaState
type GrandpaState interface {
	HandleGRANDPADigest(header *types.Header, digest types.GrandpaConsensusDigest) error
	ApplyScheduledChanges(finalizedHeader *types.Header) error
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}

// Logger logs messages at the debug or error level.
type Logger interface {
	Error(s string)
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}
