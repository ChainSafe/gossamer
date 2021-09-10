package types

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// NewBabeConsensusDigest constructs a vdt representing a babe consensus digest
func NewBabeConsensusDigest() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(NextEpochData{}, BABEOnDisabled{}, NextConfigData{})
}

// NewGrandpaConsensusDigest constructs a vdt representing a grandpa consensus digest
func NewGrandpaConsensusDigest() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(GrandpaScheduledChange{}, GrandpaForcedChange{}, GrandpaOnDisabled{}, GrandpaPause{}, GrandpaResume{})
}

// GrandpaScheduledChange represents a GRANDPA scheduled authority change
type GrandpaScheduledChange struct {
	Auths []GrandpaAuthoritiesRaw
	Delay uint32
}

// Index Returns VDT index
func (sc GrandpaScheduledChange) Index() uint { return 1 }

// GrandpaForcedChange represents a GRANDPA forced authority change
type GrandpaForcedChange struct {
	Auths []GrandpaAuthoritiesRaw
	Delay uint32
}

// Index Returns VDT index
func (fc GrandpaForcedChange) Index() uint { return 2 }

// GrandpaOnDisabled represents a GRANDPA authority being disabled
type GrandpaOnDisabled struct {
	ID uint64
}

// Index Returns VDT index
func (od GrandpaOnDisabled) Index() uint { return 3 }

// GrandpaPause represents an authority set pause
type GrandpaPause struct {
	Delay uint32
}

// Index Returns VDT index
func (p GrandpaPause) Index() uint { return 4 }

// GrandpaResume represents an authority set resume
type GrandpaResume struct {
	Delay uint32
}

// Index Returns VDT index
func (r GrandpaResume) Index() uint { return 5 }

// NextEpochData is the digest that contains the data for the upcoming BABE epoch.
// It is included in the first block of every epoch to describe the next epoch.
type NextEpochData struct {
	Authorities []AuthorityRaw
	Randomness  [RandomnessLength]byte
}

// Index Returns VDT index
func (d NextEpochData) Index() uint { return 1 }

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

// Index Returns VDT index
func (od BABEOnDisabled) Index() uint { return 2 }

// NextConfigData is the digest that contains changes to the BABE configuration.
// It is potentially included in the first block of an epoch to describe the next epoch.
type NextConfigData struct {
	C1             uint64
	C2             uint64
	SecondarySlots byte
}

// Index Returns VDT index
func (d NextConfigData) Index() uint { return 3 }

// ToConfigData returns the NextConfigData as ConfigData
func (d *NextConfigData) ToConfigData() *ConfigData {
	return &ConfigData{
		C1:             d.C1,
		C2:             d.C2,
		SecondarySlots: d.SecondarySlots,
	}
}
