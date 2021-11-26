// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// afgFinalizedBlocksUpToTM holds telemetry message of type `afg.finalized_blocks_up_to`,
// which is supposed to be sent when GRANDPA client finalises new blocks.
type afgFinalizedBlocksUpToTM struct {
	Hash   common.Hash `json:"hash"`
	Number string      `json:"number"`
}

// NewAfgFinalizedBlocksUpToTM creates a new afgFinalizedBlocksUpToTM struct.
func NewAfgFinalizedBlocksUpToTM(hash common.Hash, number string) Message {
	return &afgFinalizedBlocksUpToTM{
		Hash:   hash,
		Number: number,
	}
}

func (afgFinalizedBlocksUpToTM) messageType() string {
	return afgFinalizedBlocksUpToMsg
}
