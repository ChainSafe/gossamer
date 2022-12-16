// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/dot/types"
)

// Runtime is the runtime interface for the babe package.
type Runtime interface {
	BlockHandler
	ExtrinsicHandler
}

// BlockHandler handles block initialisation and finalisation.
type BlockHandler interface {
	InitializeBlock(header *types.Header) error
	FinalizeBlock() (*types.Header, error)
}

// ExtrinsicHandler deals with extrinsics.
type ExtrinsicHandler interface {
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}
