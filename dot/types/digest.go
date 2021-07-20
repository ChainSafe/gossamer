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
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
	scale2 "github.com/ChainSafe/gossamer/pkg/scale"
)

// Digest represents the block digest. It consists of digest items.
type Digest []DigestItem

// NewEmptyDigest returns an empty digest
func NewEmptyDigest() Digest {
	return []DigestItem{}
}

// NewDigest returns a new Digest from the given DigestItems
func NewDigest(items ...DigestItem) Digest {
	return items
}

// Encode returns the SCALE encoded digest
func (d *Digest) Encode() ([]byte, error) {
	enc, err := scale.Encode(big.NewInt(int64(len(*d))))
	if err != nil {
		return nil, err
	}

	for _, item := range *d {
		encItem, err := item.Encode()
		if err != nil {
			return nil, err
		}

		enc = append(enc, encItem...)
	}

	return enc, nil
}

// Decode decodes a SCALE encoded digest and appends it to the given Digest
func (d *Digest) Decode(r *bytes.Buffer) error {
	var err error
	digest, err := DecodeDigest(r)
	if err != nil {
		return err
	}
	*d = digest
	return nil
}

// ConsensusEngineID is a 4-character identifier of the consensus engine that produced the digest.
type ConsensusEngineID [4]byte

// NewConsensusEngineID casts a byte array to ConsensusEngineID
// if the input is longer than 4 bytes, it takes the first 4 bytes
func NewConsensusEngineID(in []byte) (res ConsensusEngineID) {
	res = [4]byte{}
	copy(res[:], in)
	return res
}

// ToBytes turns ConsensusEngineID to a byte array
func (h ConsensusEngineID) ToBytes() []byte {
	b := [4]byte(h)
	return b[:]
}

// BabeEngineID is the hard-coded babe ID
var BabeEngineID = ConsensusEngineID{'B', 'A', 'B', 'E'}

// GrandpaEngineID is the hard-coded grandpa ID
var GrandpaEngineID = ConsensusEngineID{'F', 'R', 'N', 'K'}

// ChangesTrieRootDigestType is the byte representation of ChangesTrieRootDigest
var ChangesTrieRootDigestType = byte(2)

// PreRuntimeDigestType is the byte representation of PreRuntimeDigest
var PreRuntimeDigestType = byte(6)

// ConsensusDigestType is the byte representation of ConsensusDigest
var ConsensusDigestType = byte(4)

// SealDigestType is the byte representation of SealDigest
var SealDigestType = byte(5)

// DecodeDigest decodes the input into a Digest
func DecodeDigest(buf *bytes.Buffer) (Digest, error) {
	//enc, err := ioutil.ReadAll(r)
	//if err != nil {
	//	return nil, err
	//}

	//buf := &bytes.Buffer{}
	decoder := scale2.NewDecoder(buf)
	//_, _ = buf.Write(enc)
	var num *big.Int
	err := decoder.Decode(&num)
	if err != nil {
		return nil, err
	}

	digest := make([]DigestItem, num.Uint64())
	for i := 0; i < len(digest); i++ {
		d, err := DecodeDigestItem(decoder)
		if err != nil {
			return nil, err
		}
		digest[i] = d
	}
	return digest, nil
}

// DecodeDigestItem will decode byte array to DigestItem
func DecodeDigestItem(decoder *scale2.Decoder) (DigestItem, error) {
	var digestItemVdt = scale2.MustNewVaryingDataType(ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{})
	err := decoder.Decode(&digestItemVdt)
	if err != nil {
		return nil, err
	}

	switch val := digestItemVdt.Value().(type) {
	case ChangesTrieRootDigest:
		return &val, err
	case PreRuntimeDigest:
		return &val, err
	case ConsensusDigest:
		return &val, err
	case SealDigest:
		return &val, err
	}

	return nil, errors.New("invalid digest item type")
}

// DigestItem can be of one of four types of digest: ChangesTrieRootDigest, PreRuntimeDigest, ConsensusDigest, or SealDigest.
// see https://github.com/paritytech/substrate/blob/f548309478da3935f72567c2abc2eceec3978e9f/primitives/runtime/src/generic/digest.rs#L77
type DigestItem interface {
	String() string
	Type() byte
	Encode() ([]byte, error)
	Decode(buf *bytes.Buffer) error // Decode assumes the type byte (first byte) has been removed from the encoding.
}

// ChangesTrieRootDigest contains the root of the changes trie at a given block, if the runtime supports it.
type ChangesTrieRootDigest struct {
	Hash common.Hash
}

// Index Returns VDT index
func (d ChangesTrieRootDigest) Index() uint { return 2 }

// String returns the digest as a string
func (d *ChangesTrieRootDigest) String() string {
	return fmt.Sprintf("ChangesTrieRootDigest Hash=%s", d.Hash)
}

// Type returns the type
func (d *ChangesTrieRootDigest) Type() byte {
	return ChangesTrieRootDigestType
}

// Encode will encode the ChangesTrieRootDigestType into byte array
func (d *ChangesTrieRootDigest) Encode() ([]byte, error) {
	return append([]byte{ChangesTrieRootDigestType}, d.Hash[:]...), nil
}

// Decode will decode into ChangesTrieRootDigest Hash
func (d *ChangesTrieRootDigest) Decode(buf *bytes.Buffer) error {
	hash, err := common.ReadHash(buf)
	if err != nil {
		return err
	}

	copy(d.Hash[:], hash[:])
	return nil
}

// PreRuntimeDigest contains messages from the consensus engine to the runtime.
type PreRuntimeDigest struct {
	ConsensusEngineID ConsensusEngineID
	Data              []byte
}

// Index Returns VDT index
func (d PreRuntimeDigest) Index() uint { return 6 }

// NewBABEPreRuntimeDigest returns a PreRuntimeDigest with the BABE consensus ID
func NewBABEPreRuntimeDigest(data []byte) *PreRuntimeDigest {
	return &PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              data,
	}
}

// String returns the digest as a string
func (d *PreRuntimeDigest) String() string {
	return fmt.Sprintf("PreRuntimeDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// Type will return PreRuntimeDigestType
func (d *PreRuntimeDigest) Type() byte {
	return PreRuntimeDigestType
}

// Encode will encode PreRuntimeDigest ConsensusEngineID and Data
func (d *PreRuntimeDigest) Encode() ([]byte, error) {
	enc := []byte{PreRuntimeDigestType}
	enc = append(enc, d.ConsensusEngineID[:]...)

	// encode data
	//output, err := scale.Encode(d.Data)
	output, err := scale2.Marshal(d.Data)
	if err != nil {
		return nil, err
	}

	return append(enc, output...), nil
}

// Decode will decode PreRuntimeDigest ConsensusEngineID and Data
func (d *PreRuntimeDigest) Decode(buf *bytes.Buffer) error {
	id, err := common.Read4Bytes(buf)
	if err != nil {
		return err
	}

	copy(d.ConsensusEngineID[:], id)

	sd := scale.Decoder{Reader: buf}
	output, err := sd.Decode([]byte{})
	if err != nil {
		return err
	}

	d.Data = output.([]byte)
	return nil
}

// ConsensusDigest contains messages from the runtime to the consensus engine.
type ConsensusDigest struct {
	ConsensusEngineID ConsensusEngineID
	Data              []byte
}

// Index Returns VDT index
func (d ConsensusDigest) Index() uint { return 4 }

// String returns the digest as a string
func (d *ConsensusDigest) String() string {
	return fmt.Sprintf("ConsensusDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// Type returns the ConsensusDigest type
func (d *ConsensusDigest) Type() byte {
	return ConsensusDigestType
}

// Encode will encode ConsensusDigest ConsensusEngineID and Data
func (d *ConsensusDigest) Encode() ([]byte, error) {
	enc := []byte{ConsensusDigestType}
	enc = append(enc, d.ConsensusEngineID[:]...)
	// encode data
	output, err := scale.Encode(d.Data)
	if err != nil {
		return nil, err
	}

	return append(enc, output...), nil
}

// Decode will decode into ConsensusEngineID and Data
func (d *ConsensusDigest) Decode(buf *bytes.Buffer) error {
	id, err := common.Read4Bytes(buf)
	if err != nil {
		return err
	}

	copy(d.ConsensusEngineID[:], id)

	sd := scale.Decoder{Reader: buf}
	output, err := sd.Decode([]byte{})
	if err != nil {
		return err
	}

	d.Data = output.([]byte)
	return nil
}

// DataType returns the data type of the runtime-to-consensus engine message
func (d *ConsensusDigest) DataType() byte {
	return d.Data[0]
}

// SealDigest contains the seal or signature. This is only used by native code.
type SealDigest struct {
	ConsensusEngineID ConsensusEngineID
	Data              []byte
}

// Index Returns VDT index
func (err SealDigest) Index() uint { return 5 }

// String returns the digest as a string
func (d *SealDigest) String() string {
	return fmt.Sprintf("SealDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// Type will return SealDigest type
func (d *SealDigest) Type() byte {
	return SealDigestType
}

// Encode will encode SealDigest ConsensusEngineID and Data
func (d *SealDigest) Encode() ([]byte, error) {
	enc := []byte{SealDigestType}
	enc = append(enc, d.ConsensusEngineID[:]...)
	// encode data
	output, err := scale.Encode(d.Data)
	if err != nil {
		return nil, err
	}
	return append(enc, output...), nil
}

// Decode will decode into  SealDigest ConsensusEngineID and Data
func (d *SealDigest) Decode(buf *bytes.Buffer) error {
	id, err := common.Read4Bytes(buf)
	if err != nil {
		return err
	}

	copy(d.ConsensusEngineID[:], id)

	// decode data
	sd := scale.Decoder{Reader: buf}

	output, err := sd.Decode([]byte{})
	if err != nil {
		return err
	}

	d.Data = output.([]byte)
	return nil
}
