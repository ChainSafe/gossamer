// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// NewBabeDigest returns a new VaryingDataType to represent a BabeDigest
func NewBabeDigest() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(BabePrimaryPreDigest{}, BabeSecondaryPlainPreDigest{}, BabeSecondaryVRFPreDigest{})
}

// DecodeBabePreDigest decodes the input into a BabePreRuntimeDigest
func DecodeBabePreDigest(in []byte) (scale.VaryingDataTypeValue, error) {
	babeDigest := NewBabeDigest()
	err := scale.Unmarshal(in, &babeDigest)
	if err != nil {
		return nil, err
	}

	babeDigestValue, err := babeDigest.Value()
	if err != nil {
		return nil, err
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
func (d *BabePrimaryPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
	return toPreRuntimeDigest(*d)
}

// Index Returns VDT index
func (BabePrimaryPreDigest) Index() uint { return 1 }

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
func (d *BabeSecondaryPlainPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
	return toPreRuntimeDigest(*d)
}

// Index Returns VDT index
func (BabeSecondaryPlainPreDigest) Index() uint { return 2 }

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
func (d *BabeSecondaryVRFPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
	return toPreRuntimeDigest(*d)
}

// Index Returns VDT index
func (BabeSecondaryVRFPreDigest) Index() uint { return 3 }

// toPreRuntimeDigest returns the VaryingDataTypeValue as a PreRuntimeDigest
func toPreRuntimeDigest(value scale.VaryingDataTypeValue) (*PreRuntimeDigest, error) {
	digest := NewBabeDigest()
	err := digest.Set(value)
	if err != nil {
		return nil, fmt.Errorf("cannot set varying data type value to babe digest: %w", err)
	}

	enc, err := scale.Marshal(digest)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal babe digest: %w", err)
	}

	return NewBABEPreRuntimeDigest(enc), nil
}
