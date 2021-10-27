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
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// Randomness is an alias for a byte array with length types.RandomnessLength
type Randomness = [types.RandomnessLength]byte

// VrfOutputAndProof represents the fields for VRF output and proof
type VrfOutputAndProof struct {
	output [sr25519.VRFOutputLength]byte
	proof  [sr25519.VRFProofLength]byte
}

// Slot represents a BABE slot
type Slot struct {
	start    time.Time
	duration time.Duration
	number   uint64
}

// NewSlot returns a new Slot
func NewSlot(start time.Time, duration time.Duration, number uint64) *Slot {
	return &Slot{
		start:    start,
		duration: duration,
		number:   number,
	}
}

// Authorities is an alias for []*types.Authority
type Authorities []types.Authority

// String returns the Authorities as a formatted string
func (d Authorities) String() string {
	str := ""
	for _, di := range []types.Authority(d) {
		str = str + fmt.Sprintf("[key=0x%x weight=%d] ", di.Key.Encode(), di.Weight)
	}
	return str
}

// epochData contains the current epoch information
type epochData struct {
	randomness     Randomness
	authorityIndex uint32
	authorities    []types.Authority
	threshold      *common.Uint128
}

func (ed *epochData) String() string {
	return fmt.Sprintf("randomness=%x authorityIndex=%d authorities=%v threshold=%s",
		ed.randomness,
		ed.authorityIndex,
		ed.authorities,
		ed.threshold,
	)
}
