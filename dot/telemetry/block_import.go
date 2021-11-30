// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
)

// blockImportTM struct to hold block import telemetry messages
type blockImportTM struct {
	BestHash *common.Hash `json:"best"`
	Height   *big.Int     `json:"height"`
	Origin   string       `json:"origin"`
}

// NewBlockImportTM function to create new Block Import Telemetry Message
func NewBlockImportTM(bestHash *common.Hash, height *big.Int, origin string) Message {
	return &blockImportTM{
		BestHash: bestHash,
		Height:   height,
		Origin:   origin,
	}
}

func (blockImportTM) messageType() string {
	return blockImportMsg
}
