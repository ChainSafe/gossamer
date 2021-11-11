// Copyright 2021 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package telemetry

// afgAuthoritySetTM is a telemetry message of type `afg.authority_set` which is
// meant to be sent when authority set changes (generally when a round is initiated)
type afgAuthoritySetTM struct {
	// finalized hash and finalized number are sent by substrate, but not ready by
	// substrate telemetry
	// FinalizedHash   *common.Hash `json:"hash"`
	// FinalizedNumber string       `json:"number"`
	AuthorityID    string `json:"authority_id"`
	AuthoritySetID string `json:"authority_set_id"`
	// Substrate creates an array of string of authority IDs. It JSON-serializes
	// that array and send that as a string.
	Authorities string `json:"authorities"`
}

// NewAfgAuthoritySetTM creates a new afgAuthoritySetTM struct.
func NewAfgAuthoritySetTM(authorityID, authoritySetID string, authorities string) Message {
	return &afgAuthoritySetTM{
		AuthorityID:    authorityID,
		AuthoritySetID: authoritySetID,
		Authorities:    authorities,
	}
}

func (afgAuthoritySetTM) messageType() string {
	return afgAuthoritySetMsg
}
