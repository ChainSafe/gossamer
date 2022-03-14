// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
)

type blockImportTM BlockImport

var _ Message = (*BlockImport)(nil)

// BlockImport struct to hold block import telemetry messages
type BlockImport struct {
	BestHash *common.Hash `json:"best"`
	Height   uint         `json:"height"`
	Origin   string       `json:"origin"`
}

// NewBlockImport function to create new Block Import Telemetry Message
func NewBlockImport(bestHash *common.Hash, height uint, origin string) *BlockImport {
	return &BlockImport{
		BestHash: bestHash,
		Height:   height,
		Origin:   origin,
	}
}

func (bi BlockImport) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		blockImportTM
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		Timestamp:     time.Now(),
		MessageType:   blockImportMsg,
		blockImportTM: blockImportTM(bi),
	}

	return json.Marshal(telemetryData)
}
