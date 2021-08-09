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

var BabeConsensusDigest = scale.MustNewVaryingDataType(NextEpochDataNew{}, BABEOnDisabled{}, NextConfigData{})
var GrandpaConsensusDigest = scale.MustNewVaryingDataType(GrandpaScheduledChangeNew{}, GrandpaForcedChangeNew{}, GrandpaOnDisabled{}, GrandpaPause{}, GrandpaResume{})


// GrandpaScheduledChange represents a GRANDPA scheduled authority change
type GrandpaScheduledChange struct {
	Auths []*GrandpaAuthoritiesRaw
	Delay uint32
}

type GrandpaScheduledChangeNew struct {
	Auths []GrandpaAuthoritiesRaw
	Delay uint32
}

func (sc GrandpaScheduledChangeNew) Index() uint { return 1 }

// Encode returns a SCALE encoded GrandpaScheduledChange with first type byte
func (sc *GrandpaScheduledChange) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*sc)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaScheduledChangeType}, enc...), nil
}

// GrandpaForcedChange represents a GRANDPA forced authority change
type GrandpaForcedChange struct {
	Auths []*GrandpaAuthoritiesRaw
	Delay uint32
}

type GrandpaForcedChangeNew struct {
	Auths []GrandpaAuthoritiesRaw
	Delay uint32
}

func (fc GrandpaForcedChangeNew) Index() uint { return 2 }

// Encode returns a SCALE encoded GrandpaForcedChange with first type byte
func (fc *GrandpaForcedChange) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*fc)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaForcedChangeType}, enc...), nil
}

// GrandpaOnDisabled represents a GRANDPA authority being disabled
type GrandpaOnDisabled struct {
	ID uint64
}

func (od GrandpaOnDisabled) Index() uint { return 3 }


// Encode returns a SCALE encoded GrandpaOnDisabled with first type byte
func (od *GrandpaOnDisabled) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*od)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaOnDisabledType}, enc...), nil
}

// GrandpaPause represents an authority set pause
type GrandpaPause struct {
	Delay uint32
}

func (p GrandpaPause) Index() uint { return 4 }

// Encode returns a SCALE encoded GrandpaPause with first type byte
func (p *GrandpaPause) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*p)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaPauseType}, enc...), nil
}

// GrandpaResume represents an authority set resume
type GrandpaResume struct {
	Delay uint32
}

func (r GrandpaResume) Index() uint { return 5 }

// Encode returns a SCALE encoded GrandpaResume with first type byte
func (r *GrandpaResume) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*r)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaResumeType}, enc...), nil
}

// NextEpochDataNew is the digest that contains the data for the upcoming BABE epoch.
// It is included in the first block of every epoch to describe the next epoch.
type NextEpochDataNew struct {
	Authorities []AuthorityRaw
	Randomness  [RandomnessLength]byte
}


func (d NextEpochDataNew) Index() uint { return 1 }

func (d *NextEpochDataNew) ToEpochData() (*EpochDataNew, error) {
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
