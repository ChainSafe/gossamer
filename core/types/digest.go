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

	"github.com/ChainSafe/gossamer/common"
)

// ConsensusEngineId is a 4-character identifier of the consensus engine that produced the digest.
type ConsensusEngineId [4]byte

var BabeEngineId = ConsensusEngineId{'B', 'A', 'B', 'E'}

var ChangesTrieRootDigestType = byte(0)
var PreRuntimeDigestType = byte(1)
var ConsensusDigestType = byte(2)
var SealDigestType = byte(4)

func DecodeDigestItem(in []byte) (DigestItem, error) {
	if len(in) < 2 {
		return nil, errors.New("cannot decode invalid digest encoding")
	}

	switch in[0] {
	case ChangesTrieRootDigestType:
		d := new(ChangesTrieRootDigest)
		err := d.Decode(in[1:])
		return d, err
	case PreRuntimeDigestType:
		d := new(PreRuntimeDigest)
		err := d.Decode(in[1:])
		return d, err
	case ConsensusDigestType:
		d := new(ConsensusDigest)
		err := d.Decode(in[1:])
		return d, err
	case SealDigestType:
		d := new(SealDigest)
		err := d.Decode(in[1:])
		return d, err
	}

	return nil, errors.New("invalid digest item type")
}

// DigestItem can be of one of four types of digest: ChangesTrieRootDigest, PreRuntimeDigest, ConsensusDigest, or SealDigest.
// see https://github.com/paritytech/substrate/blob/f548309478da3935f72567c2abc2eceec3978e9f/primitives/runtime/src/generic/digest.rs#L77
type DigestItem interface {
	Type() byte
	Encode() []byte
	Decode([]byte) error // Decode assumes the type byte (first byte) has been removed from the encoding.
}

// ChangesTrieRootDigest contains the root of the changes trie at a given block, if the runtime supports it.
type ChangesTrieRootDigest struct {
	Hash common.Hash
}

func (d *ChangesTrieRootDigest) Type() byte {
	return ChangesTrieRootDigestType
}

func (d *ChangesTrieRootDigest) Encode() []byte {
	return append([]byte{ChangesTrieRootDigestType}, d.Hash[:]...)
}

func (d *ChangesTrieRootDigest) Decode(in []byte) error {
	if len(in) < 32 {
		return errors.New("input is too short: need 32 bytes")
	}

	copy(d.Hash[:], in)
	return nil
}

// PreRuntimeDigest contains messages from the consensus engine to the runtime.
type PreRuntimeDigest struct {
	ConsensusEngineId ConsensusEngineId
	Data              []byte
}

func (d *PreRuntimeDigest) Type() byte {
	return PreRuntimeDigestType
}

func (d *PreRuntimeDigest) Encode() []byte {
	enc := []byte{PreRuntimeDigestType}
	enc = append(enc, d.ConsensusEngineId[:]...)
	return append(enc, d.Data...)
}

func (d *PreRuntimeDigest) Decode(in []byte) error {
	if len(in) < 4 {
		return errors.New("input is too short: need at least 4 bytes")
	}

	copy(d.ConsensusEngineId[:], in[:4])
	d.Data = in[4:]
	return nil
}

// ConsensusDigest contains messages from the runtime to the consensus engine.
type ConsensusDigest struct {
	ConsensusEngineId ConsensusEngineId
	Data              []byte
}

func (d *ConsensusDigest) Type() byte {
	return ConsensusDigestType
}

func (d *ConsensusDigest) Encode() []byte {
	enc := []byte{ConsensusDigestType}
	enc = append(enc, d.ConsensusEngineId[:]...)
	return append(enc, d.Data...)
}

func (d *ConsensusDigest) Decode(in []byte) error {
	if len(in) < 4 {
		return errors.New("input is too short: need at least 4 bytes")
	}

	copy(d.ConsensusEngineId[:], in[:4])
	d.Data = in[4:]
	return nil
}

// SealDigest contains the seal or signature. This is only used by native code.
type SealDigest struct {
	ConsensusEngineId ConsensusEngineId
	Data              []byte
}

func (d *SealDigest) Type() byte {
	return SealDigestType
}

func (d *SealDigest) Encode() []byte {
	enc := []byte{SealDigestType}
	enc = append(enc, d.ConsensusEngineId[:]...)
	return append(enc, d.Data...)
}

func (d *SealDigest) Decode(in []byte) error {
	if len(in) < 4 {
		return errors.New("input is too short: need at least 4 bytes")
	}

	copy(d.ConsensusEngineId[:], in[:4])
	d.Data = in[4:]
	return nil
}
