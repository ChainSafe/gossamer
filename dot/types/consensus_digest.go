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

// GrandpaScheduledChange represents a GRANDPA scheduled authority change
type GrandpaScheduledChange struct {
	Auths []*GrandpaAuthoritiesRaw
	Delay uint32
}

// Encode returns a SCALE encoded GrandpaScheduledChange with first type byte
func (sc *GrandpaScheduledChange) Encode() ([]byte, error) {
	d, err := scale.Marshal(sc)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaScheduledChangeType}, d...), nil
}

// GrandpaForcedChange represents a GRANDPA forced authority change
type GrandpaForcedChange struct {
	Auths []*GrandpaAuthoritiesRaw
	Delay uint32
}

// Encode returns a SCALE encoded GrandpaForcedChange with first type byte
func (fc *GrandpaForcedChange) Encode() ([]byte, error) {
	d, err := scale.Marshal(fc)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaForcedChangeType}, d...), nil
}

// GrandpaOnDisabled represents a GRANDPA authority being disabled
type GrandpaOnDisabled struct {
	ID uint64
}

// Encode returns a SCALE encoded GrandpaOnDisabled with first type byte
func (od *GrandpaOnDisabled) Encode() ([]byte, error) {
	d, err := scale.Marshal(od)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaOnDisabledType}, d...), nil
}

// GrandpaPause represents an authority set pause
type GrandpaPause struct {
	Delay uint32
}

// Encode returns a SCALE encoded GrandpaPause with first type byte
func (p *GrandpaPause) Encode() ([]byte, error) {
	d, err := scale.Marshal(p)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaPauseType}, d...), nil
}

// GrandpaResume represents an authority set resume
type GrandpaResume struct {
	Delay uint32
}

// Encode returns a SCALE encoded GrandpaResume with first type byte
func (r *GrandpaResume) Encode() ([]byte, error) {
	d, err := scale.Marshal(r)
	if err != nil {
		return nil, err
	}

	return append([]byte{GrandpaResumeType}, d...), nil
}

// NextEpochData is the digest that contains the data for the upcoming BABE epoch.
// It is included in the first block of every epoch to describe the next epoch.
type NextEpochData struct {
	Authorities []*AuthorityRaw
	Randomness  [RandomnessLength]byte
}

// Encode returns a SCALE encoded NextEpochData with first type byte
func (d *NextEpochData) Encode() ([]byte, error) {
	enc, err := scale.Marshal(d)
	if err != nil {
		return nil, err
	}

	return append([]byte{NextEpochDataType}, enc...), nil
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

// Encode returns a SCALE encoded BABEOnDisabled with first type byte
func (od *BABEOnDisabled) Encode() ([]byte, error) {
	d, err := scale.Marshal(od)
	if err != nil {
		return nil, err
	}

	return append([]byte{BABEOnDisabledType}, d...), nil
}

// NextConfigData is the digest that contains changes to the BABE configuration.
// It is potentially included in the first block of an epoch to describe the next epoch.
type NextConfigData struct {
	C1             uint64
	C2             uint64
	SecondarySlots byte
}

// Encode returns a SCALE encoded NextConfigData with first type byte
func (d *NextConfigData) Encode() ([]byte, error) {
	enc, err := scale.Marshal(d)
	if err != nil {
		return nil, err
	}

	return append([]byte{NextConfigDataType}, enc...), nil
}

// ToConfigData returns the NextConfigData as ConfigData
func (d *NextConfigData) ToConfigData() *ConfigData {
	return &ConfigData{
		C1:             d.C1,
		C2:             d.C2,
		SecondarySlots: d.SecondarySlots,
	}
}
