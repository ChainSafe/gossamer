// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// DigestItem is a varying date type that holds type identifier and a scaled encoded message payload.
type digestItem struct {
	ConsensusEngineID ConsensusEngineID
	Data              []byte
}

type DigestItem struct {
	inner any
}

type DigestItemValues interface {
	PreRuntimeDigest | ConsensusDigest | SealDigest | RuntimeEnvironmentUpdated
}

func newDigestItem[Value DigestItemValues](value Value) DigestItem {
	item := DigestItem{}
	setDigestItem[Value](&item, value)
	return item
}

func setDigestItem[Value DigestItemValues](mvdt *DigestItem, value Value) {
	mvdt.inner = value
}

func (mvdt *DigestItem) SetValue(value any) (err error) {
	switch value := value.(type) {
	case PreRuntimeDigest:
		setDigestItem(mvdt, value)
		return
	case ConsensusDigest:
		setDigestItem(mvdt, value)
		return
	case SealDigest:
		setDigestItem(mvdt, value)
		return
	case RuntimeEnvironmentUpdated:
		setDigestItem(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt DigestItem) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case PreRuntimeDigest:
		return 6, mvdt.inner, nil
	case ConsensusDigest:
		return 4, mvdt.inner, nil
	case SealDigest:
		return 5, mvdt.inner, nil
	case RuntimeEnvironmentUpdated:
		return 8, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt DigestItem) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt DigestItem) ValueAt(index uint) (value any, err error) {
	switch index {
	case 6:
		return PreRuntimeDigest{}, nil
	case 4:
		return ConsensusDigest{}, nil
	case 5:
		return SealDigest{}, nil
	case 8:
		return RuntimeEnvironmentUpdated{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewDigestItem returns a new VaryingDataType to represent a DigestItem
func NewDigestItem() DigestItem {
	return DigestItem{}
}

// Digest is slice of DigestItem
type Digest []DigestItem

func (d *Digest) Add(values ...any) (err error) {
	for _, value := range values {
		item := DigestItem{}
		err := item.SetValue(value)
		if err != nil {
			return err
		}
		appended := append(*d, item)
		*d = appended
	}
	return nil
}

// NewDigest returns a new Digest as a varying data type slice.
func NewDigest() Digest {
	return []DigestItem{}
}

// ConsensusEngineID is a 4-character identifier of the consensus engine that produced the digest.
type ConsensusEngineID [4]byte

// ToBytes turns ConsensusEngineID to a byte array
func (h ConsensusEngineID) ToBytes() []byte {
	b := [4]byte(h)
	return b[:]
}

func (h ConsensusEngineID) String() string {
	return fmt.Sprintf("0x%x", h.ToBytes())
}

// BabeEngineID is the hard-coded babe ID
var BabeEngineID = ConsensusEngineID{'B', 'A', 'B', 'E'}

// GrandpaEngineID is the hard-coded grandpa ID
var GrandpaEngineID = ConsensusEngineID{'F', 'R', 'N', 'K'}

// PreRuntimeDigest contains messages from the consensus engine to the runtime.
type PreRuntimeDigest digestItem

// // Index returns VDT index
// func (PreRuntimeDigest) Index() uint { return 6 }

// NewBABEPreRuntimeDigest returns a PreRuntimeDigest with the BABE consensus ID
func NewBABEPreRuntimeDigest(data []byte) *PreRuntimeDigest {
	return &PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              data,
	}
}

// String returns the digest as a string
func (d PreRuntimeDigest) String() string {
	return fmt.Sprintf("PreRuntimeDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// ConsensusDigest contains messages from the runtime to the consensus engine.
type ConsensusDigest digestItem

// // Index returns VDT index
// func (ConsensusDigest) Index() uint { return 4 }

// String returns the digest as a string
func (d ConsensusDigest) String() string {
	return fmt.Sprintf("ConsensusDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// SealDigest contains the seal or signature. This is only used by native code.
type SealDigest digestItem

// // Index returns VDT index
// func (SealDigest) Index() uint { return 5 }

// String returns the digest as a string
func (d SealDigest) String() string {
	return fmt.Sprintf("SealDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// RuntimeEnvironmentUpdated contains is an indicator for the light clients that the runtime environment is updated
type RuntimeEnvironmentUpdated struct{}

// // Index returns VDT index
// func (RuntimeEnvironmentUpdated) Index() uint { return 8 }

// String returns the digest as a string
func (RuntimeEnvironmentUpdated) String() string {
	return "RuntimeEnvironmentUpdated"
}
