// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
)

type afgFinalizedBlocksUpToTM AfgFinalizedBlocksUpTo

var _ Message = (*AfgFinalizedBlocksUpTo)(nil)

// AfgFinalizedBlocksUpTo holds telemetry message of type `afg.finalized_blocks_up_to`,
// which is supposed to be sent when GRANDPA client finalises new blocks.
type AfgFinalizedBlocksUpTo struct {
	Hash   common.Hash `json:"hash"`
	Number string      `json:"number"`
}

// NewAfgFinalizedBlocksUpTo creates a new AfgFinalizedBlocksUpToTM struct.
func NewAfgFinalizedBlocksUpTo(hash common.Hash, number string) *AfgFinalizedBlocksUpTo {
	return &AfgFinalizedBlocksUpTo{
		Hash:   hash,
		Number: number,
	}
}

func (AfgFinalizedBlocksUpTo) messageType() string {
	return afgFinalizedBlocksUpToMsg
}

func (afg AfgFinalizedBlocksUpTo) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgFinalizedBlocksUpToTM
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		Timestamp:                time.Now(),
		MessageType:              afg.messageType(),
		afgFinalizedBlocksUpToTM: afgFinalizedBlocksUpToTM(afg),
	}

	return json.Marshal(telemetryData)
}
