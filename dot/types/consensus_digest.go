// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// NewBabeConsensusDigest constructs a vdt representing a babe consensus digest
func NewBabeConsensusDigest() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(NextEpochData{}, BABEOnDisabled{}, NextConfigData{})
}

// NewGrandpaConsensusDigest constructs a vdt representing a grandpa consensus digest
func NewGrandpaConsensusDigest() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(GrandpaScheduledChange{}, GrandpaForcedChange{},
		GrandpaOnDisabled{}, GrandpaPause{}, GrandpaResume{})
}

// GrandpaScheduledChange represents a GRANDPA scheduled authority change
type GrandpaScheduledChange struct {
	Auths []GrandpaAuthoritiesRaw
	Delay uint32
}

// Index returns VDT index
func (GrandpaScheduledChange) Index() uint { return 1 }

// GrandpaForcedChange represents a GRANDPA forced authority change
type GrandpaForcedChange struct {
	// BestFinalizedBlock is specified by the governance mechanism, defines
	// the starting block at which Delay is applied.
	// https://github.com/w3f/polkadot-spec/pull/506#issuecomment-1128849492
	BestFinalizedBlock uint32
	Auths              []GrandpaAuthoritiesRaw
	Delay              uint32
}

// Index returns VDT index
func (GrandpaForcedChange) Index() uint { return 2 }

// GrandpaOnDisabled represents a GRANDPA authority being disabled
type GrandpaOnDisabled struct {
	ID uint64
}

// Index returns VDT index
func (GrandpaOnDisabled) Index() uint { return 3 }

// GrandpaPause represents an authority set pause
type GrandpaPause struct {
	Delay uint32
}

// Index returns VDT index
func (GrandpaPause) Index() uint { return 4 }

// GrandpaResume represents an authority set resume
type GrandpaResume struct {
	Delay uint32
}

// Index returns VDT index
func (GrandpaResume) Index() uint { return 5 }

// NextEpochData is the digest that contains the data for the upcoming BABE epoch.
// It is included in the first block of every epoch to describe the next epoch.
type NextEpochData struct {
	Authorities []AuthorityRaw
	Randomness  [RandomnessLength]byte
}

// Index returns VDT index
func (NextEpochData) Index() uint { return 1 }

func (d NextEpochData) String() string {
	return fmt.Sprintf("NextEpochData Authorities=%v Randomness=%v", d.Authorities, d.Randomness)
}

// ToEpochData returns the NextEpochData as EpochData
func (d *NextEpochData) ToEpochData() (*EpochData, error) {
	auths, err := BABEAuthorityRawToAuthority(d.Authorities)
	if err != nil {
		return nil, err
	}

	return &EpochData{
		Authorities: auths,
		Randomness:  d.Randomness,
	}, nil
}

// BABEOnDisabled represents a GRANDPA authority being disabled
type BABEOnDisabled struct {
	ID uint32
}

// Index returns VDT index
func (BABEOnDisabled) Index() uint { return 2 }

// NextConfigData is the digest that contains changes to the BABE configuration.
// It is potentially included in the first block of an epoch to describe the next epoch.
type NextConfigData struct {
	C1             uint64
	C2             uint64
	SecondarySlots byte
}

// Index returns VDT index
func (NextConfigData) Index() uint { return 3 }

// ToConfigData returns the NextConfigData as ConfigData
func (d *NextConfigData) ToConfigData() *ConfigData {
	return &ConfigData{
		C1:             d.C1,
		C2:             d.C2,
		SecondarySlots: d.SecondarySlots,
	}
}
