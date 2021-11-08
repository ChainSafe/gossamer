// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

//nolint
// notifyFinalizedTM holds `notify.finalized` telemetry message, which is
// supposed to be send when a new block gets finalized.
type notifyFinalizedTM struct {
	Best common.Hash `json:"best"`
	// Height is same as block.Header.Number
	Height string `json:"height"`
}

// NewNotifyFinalizedTM gets a new NotifyFinalizedTM struct.
func NewNotifyFinalizedTM(best common.Hash, height string) Message {
	return &notifyFinalizedTM{
		Best:   best,
		Height: height,
	}
}

func (notifyFinalizedTM) messageType() string {
	return notifyFinalizedMsg
}
