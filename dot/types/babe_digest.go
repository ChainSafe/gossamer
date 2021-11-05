// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"errors"

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

	switch msg := babeDigest.Value().(type) {
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
func NewBabePrimaryPreDigest(authorityIndex uint32, slotNumber uint64, vrfOutput [sr25519.VRFOutputLength]byte, vrfProof [sr25519.VRFProofLength]byte) *BabePrimaryPreDigest {
	return &BabePrimaryPreDigest{
		VRFOutput:      vrfOutput,
		VRFProof:       vrfProof,
		AuthorityIndex: authorityIndex,
		SlotNumber:     slotNumber,
	}
}

// ToPreRuntimeDigest returns the BabePrimaryPreDigest as a PreRuntimeDigest
func (d *BabePrimaryPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
	digest := NewBabeDigest()
	err := digest.Set(*d)
	if err != nil {
		return nil, err
	}
	enc, err := scale.Marshal(digest)
	if err != nil {
		return nil, err
	}
	return NewBABEPreRuntimeDigest(enc), nil
}

// Index Returns VDT index
func (d BabePrimaryPreDigest) Index() uint { return 1 }

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

// ToPreRuntimeDigest returns the BabePrimaryPreDigest as a PreRuntimeDigest
func (d *BabeSecondaryPlainPreDigest) ToPreRuntimeDigest() (*PreRuntimeDigest, error) {
	digest := NewBabeDigest()
	err := digest.Set(*d)
	if err != nil {
		return nil, err
	}
	enc, err := scale.Marshal(digest)
	if err != nil {
		return nil, err
	}
	return NewBABEPreRuntimeDigest(enc), nil
}

// Index Returns VDT index
func (d BabeSecondaryPlainPreDigest) Index() uint { return 2 }

// BabeSecondaryVRFPreDigest is included in a block built by a secondary slot authorized producer
type BabeSecondaryVRFPreDigest struct {
	AuthorityIndex uint32
	SlotNumber     uint64
	VrfOutput      [sr25519.VRFOutputLength]byte
	VrfProof       [sr25519.VRFProofLength]byte
}

// NewBabeSecondaryVRFPreDigest returns a new NewBabeSecondaryVRFPreDigest
func NewBabeSecondaryVRFPreDigest(authorityIndex uint32, slotNumber uint64, vrfOutput [sr25519.VRFOutputLength]byte, vrfProof [sr25519.VRFProofLength]byte) *BabeSecondaryVRFPreDigest {
	return &BabeSecondaryVRFPreDigest{
		VrfOutput:      vrfOutput,
		VrfProof:       vrfProof,
		AuthorityIndex: authorityIndex,
		SlotNumber:     slotNumber,
	}
}

// Index Returns VDT index
func (d BabeSecondaryVRFPreDigest) Index() uint { return 3 }
