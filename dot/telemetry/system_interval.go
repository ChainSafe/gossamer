// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
)

var _ Message = (*SystemIntervalTM)(nil)

// SystemIntervalTM struct to hold system interval telemetry messages
type SystemIntervalTM struct {
	BandwidthDownload  float64      `json:"bandwidth_download,omitempty"`
	BandwidthUpload    float64      `json:"bandwidth_upload,omitempty"`
	Peers              int          `json:"peers,omitempty"`
	BestHash           *common.Hash `json:"best,omitempty"`
	BestHeight         *big.Int     `json:"height,omitempty"`
	FinalisedHash      *common.Hash `json:"finalized_hash,omitempty"`
	FinalisedHeight    *big.Int     `json:"finalized_height,omitempty"`
	TxCount            *big.Int     `json:"txcount,omitempty"`
	UsedStateCacheSize *big.Int     `json:"used_state_cache_size,omitempty"`
}

// NewBandwidthTM function to create new Bandwidth Telemetry Message
func NewBandwidthTM(bandwidthDownload, bandwidthUpload float64, peers int) *SystemIntervalTM {
	return &SystemIntervalTM{
		BandwidthDownload: bandwidthDownload,
		BandwidthUpload:   bandwidthUpload,
		Peers:             peers,
	}
}

// NewBlockIntervalTM function to create new Block Interval Telemetry Message
func NewBlockIntervalTM(beshHash *common.Hash, bestHeight *big.Int, finalisedHash *common.Hash,
	finalisedHeight, txCount, usedStateCacheSize *big.Int) *SystemIntervalTM {
	return &SystemIntervalTM{
		BestHash:           beshHash,
		BestHeight:         bestHeight,
		FinalisedHash:      finalisedHash,
		FinalisedHeight:    finalisedHeight,
		TxCount:            txCount,
		UsedStateCacheSize: usedStateCacheSize,
	}
}

func (SystemIntervalTM) messageType() string {
	return systemIntervalMsg
}
