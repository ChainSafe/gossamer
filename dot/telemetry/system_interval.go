// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
)

// systemIntervalTM struct to hold system interval telemetry messages
type systemIntervalTM struct {
	BandwidthDownload  float64      `json:"bandwidth_download,omitempty"`
	BandwidthUpload    float64      `json:"bandwidth_upload,omitempty"`
	Peers              int          `json:"peers,omitempty"`
	BestHash           *common.Hash `json:"best,omitempty"`
	BestHeight         *big.Int     `json:"height,omitempty"`
	FinalisedHash      *common.Hash `json:"finalized_hash,omitempty"`   // nolint
	FinalisedHeight    *big.Int     `json:"finalized_height,omitempty"` // nolint
	TxCount            *big.Int     `json:"txcount,omitempty"`
	UsedStateCacheSize *big.Int     `json:"used_state_cache_size,omitempty"`
}

// NewBandwidthTM function to create new Bandwidth Telemetry Message
func NewBandwidthTM(bandwidthDownload, bandwidthUpload float64, peers int) Message {
	return &systemIntervalTM{
		BandwidthDownload: bandwidthDownload,
		BandwidthUpload:   bandwidthUpload,
		Peers:             peers,
	}
}

// NewBlockIntervalTM function to create new Block Interval Telemetry Message
func NewBlockIntervalTM(beshHash *common.Hash, bestHeight *big.Int, finalisedHash *common.Hash,
	finalisedHeight, txCount, usedStateCacheSize *big.Int) Message {
	return &systemIntervalTM{
		BestHash:           beshHash,
		BestHeight:         bestHeight,
		FinalisedHash:      finalisedHash,
		FinalisedHeight:    finalisedHeight,
		TxCount:            txCount,
		UsedStateCacheSize: usedStateCacheSize,
	}
}

func (systemIntervalTM) messageType() string {
	return systemIntervalMsg
}
