// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type BabeDigestValues interface {
	BabePrimaryPreDigest | BabeSecondaryPlainPreDigest | BabeSecondaryVRFPreDigest
}

type BabeDigest struct {
	inner any
}

func setBabeDigest[Value BabeDigestValues](mvdt *BabeDigest, value Value) {
	mvdt.inner = value
}

func (mvdt *BabeDigest) SetValue(value any) (err error) {
	switch value := value.(type) {
	case BabePrimaryPreDigest:
		setBabeDigest(mvdt, value)
		return

	case BabeSecondaryPlainPreDigest:
		setBabeDigest(mvdt, value)
		return

	case BabeSecondaryVRFPreDigest:
		setBabeDigest(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt BabeDigest) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case BabePrimaryPreDigest:
		return 0, mvdt.inner, nil

	case BabeSecondaryPlainPreDigest:
		return 1, mvdt.inner, nil

	case BabeSecondaryVRFPreDigest:
		return 2, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt BabeDigest) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt BabeDigest) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(BabePrimaryPreDigest), nil

	case 1:
		return *new(BabeSecondaryPlainPreDigest), nil

	case 2:
		return *new(BabeSecondaryVRFPreDigest), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewBabeDigest returns a new VaryingDataType to represent a BabeDigest
func NewBabeDigest() BabeDigest {
	return BabeDigest{}
}

// DecodeBabePreDigest decodes the input into a BabePreRuntimeDigest
func DecodeBabePreDigest(in []byte) (any, error) {
	babeDigest := NewBabeDigest()
	err := scale.Unmarshal(in, &babeDigest)
	if err != nil {
		return nil, err
	}

	babeDigestValue, err := babeDigest.Value()
	if err != nil {
		return nil, fmt.Errorf("getting babe digest value: %w", err)
	}
	switch msg := babeDigestValue.(type) {
	case BabePrimaryPreDigest, BabeSecondaryPlainPreDigest, BabeSecondaryVRFPreDigest:
		return msg, nil
	}

	return nil, errors.New("cannot decode data with invalid BABE pre-runtime digest type")
}

// BabePrimaryPreDigest as defined in Polkadot RE Spec, definition 5.10 in section 5.1.4
type BabePrimaryPreDigest struct {
	AuthorityIndex uint32
	SlotNumber     uint64
	VRFOutput      [sr25519.VRFOutputLength]byte
	VRFProof       [sr25519.VRFProofLength]byte
}

// NewBabePrimaryPreDigest returns a new BabePrimaryPreDigest
func NewBabePrimaryPreDigest(authorityIndex uint32,
	slotNumber uint64, vrfOutput [sr25519.VRFOutputLength]byte,
	vrfProof [sr25519.VRFProofLength]byte) *BabePrimaryPreDigest {
	return &BabePrimaryPreDigest{
		VRFOutput:      vrfOutput,
		VRFProof:       vrfProof,
		AuthorityIndex: authorityIndex,
		SlotNumber:     slotNumber,
	}
}

// ToPreRuntimeDigest returns the BabePrimaryPreDigest as a PreRuntimeDigest
func (d BabePrimaryPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
	return toPreRuntimeDigest(d)
}

// Index returns VDT index
func (BabePrimaryPreDigest) Index() uint { return 1 }

func (d BabePrimaryPreDigest) String() string {
	return fmt.Sprintf("BabePrimaryPreDigest{AuthorityIndex=%d, SlotNumber=%d, "+
		"VRFOutput=0x%x, VRFProof=0x%x}",
		d.AuthorityIndex, d.SlotNumber, d.VRFOutput, d.VRFProof)
}

// BabeSecondaryPlainPreDigest is included in a block built by a secondary slot authorized producer
type BabeSecondaryPlainPreDigest struct {
	AuthorityIndex uint32
	SlotNumber     uint64
}

// NewBabeSecondaryPlainPreDigest returns a new BabeSecondaryPlainPreDigest
func NewBabeSecondaryPlainPreDigest(authorityIndex uint32, slotNumber uint64) *BabeSecondaryPlainPreDigest {
	return &BabeSecondaryPlainPreDigest{
		AuthorityIndex: authorityIndex,
		SlotNumber:     slotNumber,
	}
}

// ToPreRuntimeDigest returns the BabeSecondaryPlainPreDigest as a PreRuntimeDigest
func (d BabeSecondaryPlainPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
	return toPreRuntimeDigest(d)
}

// Index returns VDT index
func (BabeSecondaryPlainPreDigest) Index() uint { return 2 }

func (d BabeSecondaryPlainPreDigest) String() string {
	return fmt.Sprintf("BabeSecondaryPlainPreDigest{AuthorityIndex=%d, SlotNumber: %d}",
		d.AuthorityIndex, d.SlotNumber)
}

// BabeSecondaryVRFPreDigest is included in a block built by a secondary slot authorized producer
type BabeSecondaryVRFPreDigest struct {
	AuthorityIndex uint32
	SlotNumber     uint64
	VrfOutput      [sr25519.VRFOutputLength]byte
	VrfProof       [sr25519.VRFProofLength]byte
}

// NewBabeSecondaryVRFPreDigest returns a new NewBabeSecondaryVRFPreDigest
func NewBabeSecondaryVRFPreDigest(authorityIndex uint32,
	slotNumber uint64, vrfOutput [sr25519.VRFOutputLength]byte,
	vrfProof [sr25519.VRFProofLength]byte) *BabeSecondaryVRFPreDigest {
	return &BabeSecondaryVRFPreDigest{
		VrfOutput:      vrfOutput,
		VrfProof:       vrfProof,
		AuthorityIndex: authorityIndex,
		SlotNumber:     slotNumber,
	}
}

// ToPreRuntimeDigest returns the BabeSecondaryVRFPreDigest as a PreRuntimeDigest
func (d BabeSecondaryVRFPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
	return toPreRuntimeDigest(d)
}

// Index returns VDT index
func (BabeSecondaryVRFPreDigest) Index() uint { return 3 }

func (d BabeSecondaryVRFPreDigest) String() string {
	return fmt.Sprintf("BabeSecondaryVRFPreDigest{AuthorityIndex=%d, SlotNumber=%d, "+
		"VrfOutput=0x%x, VrfProof=0x%x",
		d.AuthorityIndex, d.SlotNumber, d.VrfOutput, d.VrfProof)
}

// toPreRuntimeDigest returns the VaryingDataTypeValue as a PreRuntimeDigest
func toPreRuntimeDigest(value any) (*PreRuntimeDigest, error) {
	digest := NewBabeDigest()
	err := digest.SetValue(value)
	if err != nil {
		return nil, fmt.Errorf("cannot set varying data type value to babe digest: %w", err)
	}

	enc, err := scale.Marshal(digest)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal babe digest: %w", err)
	}

	return NewBABEPreRuntimeDigest(enc), nil
}
