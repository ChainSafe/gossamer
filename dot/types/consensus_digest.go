// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// NewBabeConsensusDigest constructs a vdt representing a babe consensus digest
func NewBabeConsensusDigest() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(NextEpochData{}, BABEOnDisabled{}, NewVersionedNextConfigData())
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

func (g GrandpaScheduledChange) String() string {
	return fmt.Sprintf("GrandpaScheduledChange{Auths=%v, Delay=%d", g.Auths, g.Delay)
}

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

func (g GrandpaForcedChange) String() string {
	return fmt.Sprintf("GrandpaForcedChange{BestFinalizedBlock=%d, Auths=%v, Delay=%d",
		g.BestFinalizedBlock, g.Auths, g.Delay)
}

// GrandpaOnDisabled represents a GRANDPA authority being disabled
type GrandpaOnDisabled struct {
	ID uint64
}

// Index returns VDT index
func (GrandpaOnDisabled) Index() uint { return 3 }

func (g GrandpaOnDisabled) String() string {
	return fmt.Sprintf("GrandpaOnDisabled{ID=%d}", g.ID)
}

// GrandpaPause represents an authority set pause
type GrandpaPause struct {
	Delay uint32
}

// Index returns VDT index
func (GrandpaPause) Index() uint { return 4 }

func (g GrandpaPause) String() string {
	return fmt.Sprintf("GrandpaPause{Delay=%d}", g.Delay)
}

// GrandpaResume represents an authority set resume
type GrandpaResume struct {
	Delay uint32
}

// Index returns VDT index
func (GrandpaResume) Index() uint { return 5 }

func (g GrandpaResume) String() string {
	return fmt.Sprintf("GrandpaResume{Delay=%d}", g.Delay)
}

// NextEpochData is the digest that contains the data for the upcoming BABE epoch.
// It is included in the first block of every epoch to describe the next epoch.
type NextEpochData struct {
	Authorities []AuthorityRaw
	Randomness  [RandomnessLength]byte
}

// Index returns VDT index
func (NextEpochData) Index() uint { return 1 } //skipcq: GO-W1029

func (d NextEpochData) String() string { //skipcq: GO-W1029
	return fmt.Sprintf("NextEpochData Authorities=%v Randomness=%v", d.Authorities, d.Randomness)
}

// ToEpochData returns the NextEpochData as EpochData
func (d *NextEpochData) ToEpochDataRaw() *EpochDataRaw {
	return &EpochDataRaw{
		Authorities: d.Authorities,
		Randomness:  d.Randomness,
	}
}

// BABEOnDisabled represents a GRANDPA authority being disabled
type BABEOnDisabled struct {
	ID uint32
}

// Index returns VDT index
func (BABEOnDisabled) Index() uint { return 2 }

func (b BABEOnDisabled) String() string {
	return fmt.Sprintf("BABEOnDisabled{ID=%d}", b.ID)
}

// NextConfigDataV1 is the digest that contains changes to the BABE configuration.
// It is potentially included in the first block of an epoch to describe the next epoch.
type NextConfigDataV1 struct {
	C1             uint64
	C2             uint64
	SecondarySlots byte
}

// Index returns VDT index
func (NextConfigDataV1) Index() uint { return 1 } //skipcq: GO-W1029

func (d NextConfigDataV1) String() string { //skipcq: GO-W1029
	return fmt.Sprintf("NextConfigData{C1=%d, C2=%d, SecondarySlots=%d}",
		d.C1, d.C2, d.SecondarySlots)
}

// ToConfigData returns the NextConfigData as ConfigData
func (d *NextConfigDataV1) ToConfigData() *ConfigData { //skipcq: GO-W1029
	return &ConfigData{
		C1:             d.C1,
		C2:             d.C2,
		SecondarySlots: d.SecondarySlots,
	}
}

// VersionedNextConfigData represents the enum of next config data consensus digest messages
type VersionedNextConfigData scale.VaryingDataType

// Index returns VDT index
func (VersionedNextConfigData) Index() uint { return 3 }

// Value returns the current VDT value
func (vncd *VersionedNextConfigData) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*vncd)
	return vdt.Value()
}

// Set updates the current VDT value to be `val`
func (vncd *VersionedNextConfigData) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*vncd)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting varying data type value: %w", err)
	}
	*vncd = VersionedNextConfigData(vdt)
	return nil
}

// String returns the string representation for the current VDT value
func (vncd VersionedNextConfigData) String() string {
	val, err := vncd.Value()
	if err != nil {
		return "VersionedNextConfigData()"
	}

	return fmt.Sprintf("VersionedNextConfigData(%s)", val)
}

// NewVersionedNextConfigData creates a new VersionedNextConfigData instance
func NewVersionedNextConfigData() VersionedNextConfigData {
	vdt := scale.MustNewVaryingDataType(NextConfigDataV1{})

	return VersionedNextConfigData(vdt)
}
