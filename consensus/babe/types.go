// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package babe

import (
	"github.com/ChainSafe/gossamer/crypto/sr25519"
)

// BabeConfiguration contains the starting data needed for Babe
// see: https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/consensus/babe/primitives/src/lib.rs#L132
type BabeConfiguration struct {
	SlotDuration       uint64 // milliseconds
	EpochLength        uint64 // duration of epoch in slots
	C1                 uint64 // (1-(c1/c2)) is the probability of a slot being empty
	C2                 uint64
	GenesisAuthorities []AuthorityDataRaw
	Randomness         byte
	SecondarySlots     bool
}

type AuthorityDataRaw struct {
	Id     [sr25519.PublicKeyLength]byte
	Weight uint64
}

//nolint:structcheck
type AuthorityData struct {
	id     *sr25519.PublicKey
	weight uint64
}

func NewAuthorityData(pub *sr25519.PublicKey, weight uint64) *AuthorityData {
	return &AuthorityData{
		id:     pub,
		weight: weight,
	}
}

type VrfOutputAndProof struct {
	output [32]byte
	proof  [64]byte
}

// Slot represents a BABE slot
type Slot struct {
	start    uint64
	duration uint64
	number   uint64
}
