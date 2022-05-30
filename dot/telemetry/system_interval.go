// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
)

type systemIntervalTM SystemInterval

var _ Message = (*SystemInterval)(nil)

// SystemInterval struct to hold system interval telemetry messages
type SystemInterval struct {
	BandwidthDownload  float64      `json:"bandwidth_download,omitempty"`
	BandwidthUpload    float64      `json:"bandwidth_upload,omitempty"`
	Peers              int          `json:"peers,omitempty"`
	BestHash           *common.Hash `json:"best,omitempty"`
	BestHeight         uint         `json:"height,omitempty"`
	FinalisedHash      *common.Hash `json:"finalized_hash,omitempty"`
	FinalisedHeight    uint         `json:"finalized_height,omitempty"`
	TxCount            *big.Int     `json:"txcount,omitempty"`
	UsedStateCacheSize *big.Int     `json:"used_state_cache_size,omitempty"`
}

// NewBandwidth function to create new Bandwidth Telemetry Message
func NewBandwidth(bandwidthDownload, bandwidthUpload float64, peers int) *SystemInterval {
	return &SystemInterval{
		BandwidthDownload: bandwidthDownload,
		BandwidthUpload:   bandwidthUpload,
		Peers:             peers,
	}
}

// NewBlockInterval function to create new Block Interval Telemetry Message
func NewBlockInterval(beshHash *common.Hash, bestHeight uint, finalisedHash *common.Hash,
	finalisedHeight uint, txCount, usedStateCacheSize *big.Int) *SystemInterval {
	return &SystemInterval{
		BestHash:           beshHash,
		BestHeight:         bestHeight,
		FinalisedHash:      finalisedHash,
		FinalisedHeight:    finalisedHeight,
		TxCount:            txCount,
		UsedStateCacheSize: usedStateCacheSize,
	}
}

func (si SystemInterval) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		systemIntervalTM
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		Timestamp:        time.Now(),
		MessageType:      systemIntervalMsg,
		systemIntervalTM: systemIntervalTM(si),
	}

	return json.Marshal(telemetryData)
}
