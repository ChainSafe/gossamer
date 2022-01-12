// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
)

type PreparedBlockForProposingTM PreparedBlockForProposing

var _ Message = (*PreparedBlockForProposing)(nil)

// PreparedBlockForProposing holds a 'prepared_block_for_proposing' telemetry
// message, which is supposed to be sent when a new block is built.
type PreparedBlockForProposing struct {
	Hash common.Hash `json:"hash"`
	// Height of the chain, Block.Header.Number
	Number string `json:"number"`
}

// NewPreparedBlockForProposingTM gets a new PreparedBlockForProposingTM struct.
func NewPreparedBlockForProposing(hash common.Hash, number string) *PreparedBlockForProposing {
	return &PreparedBlockForProposing{
		Hash:   hash,
		Number: number,
	}
}

func (PreparedBlockForProposing) messageType() string {
	return preparedBlockForProposingMsg
}

func (pb PreparedBlockForProposing) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		PreparedBlockForProposingTM
		Timestamp   time.Time `json:"ts"`
		MessageType string    `json:"msg"`
	}{
		Timestamp:                   time.Now(),
		MessageType:                 pb.messageType(),
		PreparedBlockForProposingTM: PreparedBlockForProposingTM(pb),
	}

	return json.Marshal(telemetryData)
}
