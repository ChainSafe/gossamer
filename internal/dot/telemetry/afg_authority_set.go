// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

// afgAuthoritySetTM is a telemetry message of type `afg.authority_set` which is
// meant to be sent when authority set changes (generally when a round is initiated)
type afgAuthoritySetTM struct {
	AuthorityID    string `json:"authority_id"`
	AuthoritySetID string `json:"authority_set_id"`
	// Substrate creates an array of string of authority IDs. It JSON-serialises
	// that array and send that as a string.
	Authorities string `json:"authorities"`
}

// NewAfgAuthoritySetTM creates a new afgAuthoritySetTM struct.
func NewAfgAuthoritySetTM(authorityID, authoritySetID, authorities string) Message {
	return &afgAuthoritySetTM{
		AuthorityID:    authorityID,
		AuthoritySetID: authoritySetID,
		Authorities:    authorities,
	}
}

func (afgAuthoritySetTM) messageType() string {
	return afgAuthoritySetMsg
}
