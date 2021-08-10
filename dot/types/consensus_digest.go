package types

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// The follow are the consensus digest types for grandpa
var (
	GrandpaScheduledChangeType = byte(1)
	GrandpaForcedChangeType    = byte(2)
	GrandpaOnDisabledType      = byte(3)
	GrandpaPauseType           = byte(4)
	GrandpaResumeType          = byte(5)
)

// The follow are the consensus digest types for BABE
var (
	NextEpochDataType  = byte(1)
	BABEOnDisabledType = byte(2)
	NextConfigDataType = byte(3)
)

var BabeConsensusDigest = scale.MustNewVaryingDataType(NextEpochData{}, BABEOnDisabled{}, NextConfigData{})
var GrandpaConsensusDigest = scale.MustNewVaryingDataType(GrandpaScheduledChange{}, GrandpaForcedChange{}, GrandpaOnDisabled{}, GrandpaPause{}, GrandpaResume{})

type GrandpaScheduledChange struct {
	Auths []GrandpaAuthoritiesRaw
	Delay uint32
}

func (sc GrandpaScheduledChange) Index() uint { return 1 }

type GrandpaForcedChange struct {
	Auths []GrandpaAuthoritiesRaw
	Delay uint32
}

func (fc GrandpaForcedChange) Index() uint { return 2 }

// GrandpaOnDisabled represents a GRANDPA authority being disabled
type GrandpaOnDisabled struct {
	ID uint64
}

func (od GrandpaOnDisabled) Index() uint { return 3 }

// GrandpaPause represents an authority set pause
type GrandpaPause struct {
	Delay uint32
}

func (p GrandpaPause) Index() uint { return 4 }

// GrandpaResume represents an authority set resume
type GrandpaResume struct {
	Delay uint32
}

func (r GrandpaResume) Index() uint { return 5 }

// NextEpochData is the digest that contains the data for the upcoming BABE epoch.
// It is included in the first block of every epoch to describe the next epoch.
type NextEpochData struct {
	Authorities []AuthorityRaw
	Randomness  [RandomnessLength]byte
}


func (d NextEpochData) Index() uint { return 1 }

func (d *NextEpochData) ToEpochData() (*EpochDataNew, error) {
	auths, err := BABEAuthorityRawToAuthorityNew(d.Authorities)
	if err != nil {
		return nil, err
	}

	return &EpochDataNew{
		Authorities: auths,
		Randomness:  d.Randomness,
	}, nil
}

// BABEOnDisabled represents a GRANDPA authority being disabled
type BABEOnDisabled struct {
	ID uint32
}

func (od BABEOnDisabled) Index() uint { return 2 }

// NextConfigData is the digest that contains changes to the BABE configuration.
// It is potentially included in the first block of an epoch to describe the next epoch.
type NextConfigData struct {
	C1             uint64
	C2             uint64
	SecondarySlots byte
}

func (d NextConfigData) Index() uint { return 3 }

// ToConfigData returns the NextConfigData as ConfigData
func (d *NextConfigData) ToConfigData() *ConfigData {
	return &ConfigData{
		C1:             d.C1,
		C2:             d.C2,
		SecondarySlots: d.SecondarySlots,
	}
}
