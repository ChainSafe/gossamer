// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

// AfgAuthoritySetTM is a telemetry message of type `afg.authority_set` which is
// meant to be sent when authority set changes (generally when a round is initiated)
type AfgAuthoritySetTM struct {
	AuthorityID    string `json:"authority_id"`
	AuthoritySetID string `json:"authority_set_id"`
	// Substrate creates an array of string of authority IDs. It JSON-serialises
	// that array and send that as a string.
	Authorities string `json:"authorities"`
}

// NewAfgAuthoritySetTM creates a new AfgAuthoritySetTM struct.
func NewAfgAuthoritySetTM(authorityID, authoritySetID, authorities string) AfgAuthoritySetTM {
	return AfgAuthoritySetTM{
		AuthorityID:    authorityID,
		AuthoritySetID: authoritySetID,
		Authorities:    authorities,
	}
}

func (AfgAuthoritySetTM) messageType() string {
	return afgAuthoritySetMsg
}
