// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
)

type notifyFinalizedTM NotifyFinalized

var _ Message = (*NotifyFinalized)(nil)

// NotifyFinalized holds `notify.finalized` telemetry message, which is
// supposed to be send when a new block gets finalised.
type NotifyFinalized struct {
	Best *common.Hash `json:"best"`
	// Height is same as block.Header.Number
	Height string `json:"height"`
}

// NewNotifyFinalized gets a new NotifyFinalizedTM struct.
func NewNotifyFinalized(best *common.Hash, height string) *NotifyFinalized {
	return &NotifyFinalized{
		Best:   best,
		Height: height,
	}
}

func (nf NotifyFinalized) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		notifyFinalizedTM
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		Timestamp:         time.Now(),
		MessageType:       notifyFinalizedMsg,
		notifyFinalizedTM: notifyFinalizedTM(nf),
	}

	return json.Marshal(telemetryData)
}
