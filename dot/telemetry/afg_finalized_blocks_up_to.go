// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// AfgFinalizedBlocksUpToTM holds telemetry message of type `afg.finalized_blocks_up_to`,
// which is supposed to be sent when GRANDPA client finalises new blocks.
type AfgFinalizedBlocksUpToTM struct {
	Hash   common.Hash `json:"hash"`
	Number string      `json:"number"`
}

// NewAfgFinalizedBlocksUpToTM creates a new AfgFinalizedBlocksUpToTM struct.
func NewAfgFinalizedBlocksUpToTM(hash common.Hash, number string) AfgFinalizedBlocksUpToTM {
	return AfgFinalizedBlocksUpToTM{
		Hash:   hash,
		Number: number,
	}
}

func (AfgFinalizedBlocksUpToTM) messageType() string {
	return afgFinalizedBlocksUpToMsg
}
