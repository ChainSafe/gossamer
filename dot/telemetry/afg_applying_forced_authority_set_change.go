// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"
)

type afgApplyingForcedAuthoritySetChange AfgApplyingForcedAuthoritySetChange

var _ json.Marshaler = (*AfgApplyingForcedAuthoritySetChange)(nil)

// AfgApplyingForcedAuthoritySetChange is a telemetry message of type `afg.applying_forced_authority_set_change`
// which is meant to be sent when a forced change is applied
type AfgApplyingForcedAuthoritySetChange struct {
	Block string `json:"block"`
}

// NewAfgApplyingForcedAuthoritySetChange creates a new AfgAuthoritySetTM struct.
func NewAfgApplyingForcedAuthoritySetChange(block string) *AfgApplyingForcedAuthoritySetChange {
	return &AfgApplyingForcedAuthoritySetChange{
		Block: block,
	}
}

func (afg AfgApplyingForcedAuthoritySetChange) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgApplyingForcedAuthoritySetChange
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		afgApplyingForcedAuthoritySetChange: afgApplyingForcedAuthoritySetChange(afg),
		MessageType:                         afgApplyingForcedAuthoritySetChangeMsg,
		Timestamp:                           time.Now(),
	}

	return json.Marshal(telemetryData)
}
