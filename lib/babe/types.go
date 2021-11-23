// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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

func (s Slot) String() string {
	return fmt.Sprintf("slot number %d started at %s for a duration of %s",
		s.number, s.start, s.duration)
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
