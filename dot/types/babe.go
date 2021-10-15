// Copyright 2020 ChainSafe Systems (ON) Corp.
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
package types

import (
	"fmt"
)

// RandomnessLength is the length of the epoch randomness (32 bytes)
const RandomnessLength = 32

// BabeConfiguration contains the genesis data for BABE
// see: https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/consensus/babe/primitives/src/lib.rs#L132
type BabeConfiguration struct {
	SlotDuration       uint64 // milliseconds
	EpochLength        uint64 // duration of epoch in slots
	C1                 uint64 // (1-(c1/c2)) is the probability of a slot being empty
	C2                 uint64
	GenesisAuthorities []AuthorityRaw
	Randomness         [RandomnessLength]byte
	SecondarySlots     byte
}

// BABEAuthorityRawToAuthority turns a slice of BABE AuthorityRaw into a slice of Authority
func BABEAuthorityRawToAuthority(adr []AuthorityRaw) ([]Authority, error) {
	ad := make([]Authority, len(adr))
	for i, r := range adr {
		ad[i] = Authority{}
		err := ad[i].FromRawSr25519(&r) //nolint
		if err != nil {
			return nil, err
		}
	}

	return ad, nil
}

// EpochData is the data provided for a BABE epoch
type EpochData struct {
	Authorities []Authority
	Randomness  [RandomnessLength]byte
}

// ToEpochDataRaw returns the EpochData as an EpochDataRaw, converting the Authority to AuthorityRaw
func (d *EpochData) ToEpochDataRaw() *EpochDataRaw {
	raw := &EpochDataRaw{
		Randomness: d.Randomness,
	}

	rawAuths := make([]AuthorityRaw, len(d.Authorities))
	for i, auth := range d.Authorities {
		rawAuths[i] = *auth.ToRaw()
	}

	raw.Authorities = rawAuths
	return raw
}

// EpochDataRaw is the data provided for an epoch, with Authority as AuthorityRaw
type EpochDataRaw struct {
	Authorities []AuthorityRaw
	Randomness  [RandomnessLength]byte
}

// ToEpochData returns the EpochDataRaw as EpochData
func (d *EpochDataRaw) ToEpochData() (*EpochData, error) {
	epochData := &EpochData{
		Randomness: d.Randomness,
	}

	auths, err := BABEAuthorityRawToAuthority(d.Authorities)
	if err != nil {
		return nil, err
	}

	epochData.Authorities = auths
	return epochData, nil
}

// ConfigData represents a BABE configuration update
type ConfigData struct {
	C1             uint64
	C2             uint64
	SecondarySlots byte // TODO: this is unused, will need to update BABE verifier to use this (#1863)
}

// GetSlotFromHeader returns the BABE slot from the given header
func GetSlotFromHeader(header *Header) (uint64, error) {
	if len(header.Digest.Types) == 0 {
		return 0, fmt.Errorf("chain head missing digest")
	}

	preDigest, ok := header.Digest.Types[0].Value().(PreRuntimeDigest)
	if !ok {
		return 0, fmt.Errorf("first digest item is not pre-digest")
	}

	digest, err := DecodeBabePreDigest(preDigest.Data)
	if err != nil {
		return 0, fmt.Errorf("cannot decode BabePreDigest from pre-digest: %s", err)
	}

	var slotNumber uint64
	switch d := digest.(type) {
	case BabePrimaryPreDigest:
		slotNumber = d.SlotNumber
	case BabeSecondaryVRFPreDigest:
		slotNumber = d.SlotNumber
	case BabeSecondaryPlainPreDigest:
		slotNumber = d.SlotNumber
	}

	return slotNumber, nil
}
