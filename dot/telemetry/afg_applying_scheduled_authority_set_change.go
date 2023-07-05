// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"
)

type afgApplyingScheduledAuthoritySetChange AfgApplyingScheduledAuthoritySetChange

var _ json.Marshaler = (*AfgApplyingScheduledAuthoritySetChange)(nil)

// AfgApplyingScheduledAuthoritySetChange is a telemetry message of type `afg.applying_scheduled_authority_set_change`
// which is meant to be sent when a scheduled change is applied
type AfgApplyingScheduledAuthoritySetChange struct {
	Block string `json:"block"`
}

// NewAfgApplyingScheduledAuthoritySetChange creates a new AfgAuthoritySetTM struct.
func NewAfgApplyingScheduledAuthoritySetChange(block string) *AfgApplyingScheduledAuthoritySetChange {
	return &AfgApplyingScheduledAuthoritySetChange{
		Block: block,
	}
}

func (afg AfgApplyingScheduledAuthoritySetChange) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgApplyingScheduledAuthoritySetChange
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		afgApplyingScheduledAuthoritySetChange: afgApplyingScheduledAuthoritySetChange(afg),
		MessageType:                            afgApplyingScheduledAuthoritySetChangeMsg,
		Timestamp:                              time.Now(),
	}

	return json.Marshal(telemetryData)
}
