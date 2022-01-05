// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

var _ Message = (*PreparedBlockForProposingTM)(nil)

// PreparedBlockForProposingTM holds a 'prepared_block_for_proposing' telemetry
// message, which is supposed to be sent when a new block is built.
type PreparedBlockForProposingTM struct {
	Hash common.Hash `json:"hash"`
	// Height of the chain, Block.Header.Number
	Number string `json:"number"`
	Msg    string `json:"msg"`
}

// NewPreparedBlockForProposingTM gets a new PreparedBlockForProposingTM struct.
func NewPreparedBlockForProposingTM(hash common.Hash, number string) *PreparedBlockForProposingTM {
	return &PreparedBlockForProposingTM{
		Hash:   hash,
		Number: number,
	}
}

func (PreparedBlockForProposingTM) messageType() string {
	return preparedBlockForProposingMsg
}
