// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// preparedBlockForProposingTM holds a 'prepared_block_for_proposing' telemetry
// message, which is supposed to be sent when a new block is built.
type preparedBlockForProposingTM struct {
	Hash common.Hash `json:"hash"`
	// Height of the chain, Block.Header.Number
	Number string `json:"number"`
	Msg    string `json:"msg"`
}

// NewPreparedBlockForProposingTM gets a new PreparedBlockForProposingTM struct.
func NewPreparedBlockForProposingTM(hash common.Hash, number string) Message {
	return &preparedBlockForProposingTM{
		Hash:   hash,
		Number: number,
		Msg:    "prepared_block_for_proposing",
	}
}

func (tm *preparedBlockForProposingTM) messageType() string {
	return tm.Msg
}
