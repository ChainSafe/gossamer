// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"
)

type afgAuthoritySetTM AfgAuthoritySet

var _ Message = (*AfgAuthoritySet)(nil)

// AfgAuthoritySet is a telemetry message of type `afg.authority_set` which is
// meant to be sent when authority set changes (generally when a round is initiated)
type AfgAuthoritySet struct {
	AuthorityID    string `json:"authority_id"`
	AuthoritySetID string `json:"authority_set_id"`
	// Substrate creates an array of string of authority IDs. It JSON-serialises
	// that array and send that as a string.
	Authorities string `json:"authorities"`
}

// NewAfgAuthoritySet creates a new AfgAuthoritySetTM struct.
func NewAfgAuthoritySet(authorityID, authoritySetID, authorities string) *AfgAuthoritySet {
	return &AfgAuthoritySet{
		AuthorityID:    authorityID,
		AuthoritySetID: authoritySetID,
		Authorities:    authorities,
	}
}

func (AfgAuthoritySet) Type() string {
	return afgAuthoritySetMsg
}

// MarshalJSON ...
func (afg AfgAuthoritySet) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		afgAuthoritySetTM
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		afgAuthoritySetTM: afgAuthoritySetTM(afg),
		MessageType:       afg.Type(),
		Timestamp:         time.Now(),
	}

	return json.Marshal(telemetryData)
}
