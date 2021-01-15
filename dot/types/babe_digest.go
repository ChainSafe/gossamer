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
	"encoding/binary"
	"errors"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

var _ BabePreRuntimeDigest = &BabePrimaryPreDigest{}
var _ BabePreRuntimeDigest = &BabeSecondaryPlainPreDigest{}

// BabePreRuntimeDigest must be implemented by all BABE pre-runtime digest types
type BabePreRuntimeDigest interface {
	Type() byte
	Encode() []byte
	Decode(r io.Reader) error
	AuthorityIndex() uint64
	SlotNumber() uint64
}

var (
	BabePrimaryPreDigestType        = byte(1)
	BabeSecondaryPlainPreDigestType = byte(2)
	BabeSecondaryVRFPreDigestType   = byte(3)
)

func DecodeBabePreDigest(r io.Reader) (BabePreRuntimeDigest, error) {
	typ, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	switch typ {
	case BabePrimaryPreDigestType:
		d := new(BabePrimaryPreDigest)
		return d, d.Decode(r)
	case BabeSecondaryPlainPreDigestType:
		d := new(BabeSecondaryPlainPreDigest)
		return d, d.Decode(r)
	case BabeSecondaryVRFPreDigestType:
		d := new(BabeSecondaryVRFPreDigest)
		return d, d.Decode(r)
	}

	return nil, errors.New("cannot decode data with invalid BABE pre-runtime digest type")
}

// BabePrimaryPreDigest as defined in Polkadot RE Spec, definition 5.10 in section 5.1.4
type BabePrimaryPreDigest struct {
	authorityIndex 	uint64
	slotNumber         uint64
	vrfOutput          [sr25519.VrfOutputLength]byte
	vrfProof           [sr25519.VrfProofLength]byte
}

func NewBabePrimaryPreDigest(vrfOutput [sr25519.VrfOutputLength]byte, vrfProof [sr25519.VrfProofLength]byte, authorityIndex, slotNumber uint64) *BabePrimaryPreDigest {
	return &BabePrimaryPreDigest{
		vrfOutput: vrfOutput,
		vrfProof: vrfProof,
		authorityIndex: authorityIndex,
		slotNumber: slotNumber,
	}
}

func (d *BabePrimaryPreDigest) Type() byte {
	return BabePrimaryPreDigestType
}

// Encode performs SCALE encoding of a BABEPrimaryPreDigest
func (d *BabePrimaryPreDigest) Encode() []byte {
	enc := []byte{BabePrimaryPreDigestType}
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, d.authorityIndex)
	enc = append(enc, buf...)
	binary.LittleEndian.PutUint64(buf, d.slotNumber)
	enc = append(enc, buf...)
	enc = append(enc, d.vrfOutput[:]...)
	enc = append(enc, d.vrfProof[:]...)
	return enc
}

// Decode performs SCALE decoding of an encoded BabePrimaryPreDigest, assuming type byte is removed
func (d *BabePrimaryPreDigest) Decode(r io.Reader) (err error) {
	d.authorityIndex, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	d.slotNumber, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	d.vrfOutput, err = common.Read32Bytes(r)
	if err != nil {
		return err
	}

	d.vrfProof, err = common.Read64Bytes(r)
	if err != nil {
		return err
	}
	return nil
}

func (d *BabePrimaryPreDigest) AuthorityIndex() uint64 {
	return d.authorityIndex
}

func (d *BabePrimaryPreDigest) SlotNumber() uint64 {
	return d.slotNumber
}

type BabeSecondaryPlainPreDigest struct{
	authorityIndex uint64
	slotNumber uint64
}

func NewBabeSecondaryPlainPreDigest(authorityIndex, slotNumber uint64) *BabeSecondaryPlainPreDigest {
	return &BabeSecondaryPlainPreDigest{
		authorityIndex: authorityIndex,
		slotNumber: slotNumber,
	}
}

func (d *BabeSecondaryPlainPreDigest) Type() byte {
	return BabeSecondaryPlainPreDigestType
}

// Encode performs SCALE encoding of a BABEPrimaryPreDigest
func (d *BabeSecondaryPlainPreDigest) Encode() []byte {
	enc := []byte{BabeSecondaryPlainPreDigestType}
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, d.authorityIndex)
	enc = append(enc, buf...)
	binary.LittleEndian.PutUint64(buf, d.slotNumber)
	enc = append(enc, buf...)
	return enc
}

// Decode performs SCALE decoding of an encoded BABEPrimaryPreDigest
func (d *BabeSecondaryPlainPreDigest) Decode(r io.Reader) (err error) {
	d.authorityIndex, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	d.slotNumber, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	return nil
}

func (d *BabeSecondaryPlainPreDigest) AuthorityIndex() uint64 {
	return d.authorityIndex
}

func (d *BabeSecondaryPlainPreDigest) SlotNumber() uint64 {
	return d.slotNumber
}

type BabeSecondaryVRFPreDigest struct{
	authorityIndex 	uint64
	slotNumber         uint64
	vrfOutput          [sr25519.VrfOutputLength]byte
	vrfProof           [sr25519.VrfProofLength]byte
}

func NewBabeSecondaryVRFPreDigest(vrfOutput [sr25519.VrfOutputLength]byte, vrfProof [sr25519.VrfProofLength]byte, authorityIndex, slotNumber uint64) *BabeSecondaryVRFPreDigest {
	return &BabeSecondaryVRFPreDigest{
		vrfOutput: vrfOutput,
		vrfProof: vrfProof,
		authorityIndex: authorityIndex,
		slotNumber: slotNumber,
	}
}

func (d *BabeSecondaryVRFPreDigest) Type() byte {
	return BabeSecondaryVRFPreDigestType
}

// Encode performs SCALE encoding of a BABEPrimaryPreDigest
func (d *BabeSecondaryVRFPreDigest) Encode() []byte {
	enc := []byte{BabeSecondaryVRFPreDigestType}
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, d.authorityIndex)
	enc = append(enc, buf...)
	binary.LittleEndian.PutUint64(buf, d.slotNumber)
	enc = append(enc, buf...)
	enc = append(enc, d.vrfOutput[:]...)
	enc = append(enc, d.vrfProof[:]...)
	return enc
}

// Decode performs SCALE decoding of an encoded BabePrimaryPreDigest, assuming type byte is removed
func (d *BabeSecondaryVRFPreDigest) Decode(r io.Reader) (err error) {
	d.authorityIndex, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	d.slotNumber, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	d.vrfOutput, err = common.Read32Bytes(r)
	if err != nil {
		return err
	}

	d.vrfProof, err = common.Read64Bytes(r)
	if err != nil {
		return err
	}
	return nil
}

func (d *BabeSecondaryVRFPreDigest) AuthorityIndex() uint64 {
	return d.authorityIndex
}

func (d *BabeSecondaryVRFPreDigest) SlotNumber() uint64 {
	return d.slotNumber
}