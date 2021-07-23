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
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Header is a state block header
type Header struct {
	ParentHash     common.Hash `json:"parentHash"`
	Number         *big.Int    `json:"number"`
	StateRoot      common.Hash `json:"stateRoot"`
	ExtrinsicsRoot common.Hash `json:"extrinsicsRoot"`
	Digest         Digest      `json:"digest"`
	hash           common.Hash
}

// Header is a state block header
type HeaderVdt struct {
	ParentHash     common.Hash `json:"parentHash"`
	Number         *big.Int    `json:"number"`
	StateRoot      common.Hash `json:"stateRoot"`
	ExtrinsicsRoot common.Hash `json:"extrinsicsRoot"`
	Digest         scale.VaryingDataTypeSlice      `json:"digest"`
	hash           common.Hash
}

// NewHeader creates a new block header and sets its hash field
func NewHeaderVdt(parentHash, stateRoot, extrinsicsRoot common.Hash, number *big.Int, digest scale.VaryingDataTypeSlice) (*HeaderVdt, error) {
	if number == nil {
		// Hash() will panic if number is nil
		return nil, errors.New("cannot have nil block number")
	}

	bh := &HeaderVdt{
		ParentHash:     parentHash,
		Number:         number,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         digest,
	}

	bh.Hash()
	return bh, nil
}

// NewHeader creates a new block header and sets its hash field
func NewHeader(parentHash, stateRoot, extrinsicsRoot common.Hash, number *big.Int, digest []DigestItem) (*Header, error) {
	if number == nil {
		// Hash() will panic if number is nil
		return nil, errors.New("cannot have nil block number")
	}

	bh := &Header{
		ParentHash:     parentHash,
		Number:         number,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         digest,
	}

	bh.Hash()
	return bh, nil
}

// NewEmptyHeader returns a new header with all zero values
func NewEmptyHeaderVdt() *HeaderVdt {
	var diVdt = scale.MustNewVaryingDataType(ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{})
	var vdtSlice = scale.NewVaryingDataTypeSlice(diVdt)
	return &HeaderVdt{
		Number: big.NewInt(0),
		Digest: vdtSlice,
	}
}

// NewEmptyHeader returns a new header with all zero values
func NewEmptyHeader() *Header {
	return &Header{
		Number: big.NewInt(0),
		Digest: []DigestItem{},
	}
}

// DeepCopy returns a deep copy of the header to prevent side effects down the road
func (bh *HeaderVdt) DeepCopy() *HeaderVdt {
	cp := NewEmptyHeaderVdt()
	copy(cp.ParentHash[:], bh.ParentHash[:])
	copy(cp.StateRoot[:], bh.StateRoot[:])
	copy(cp.ExtrinsicsRoot[:], bh.ExtrinsicsRoot[:])

	if bh.Number != nil {
		cp.Number = new(big.Int).Set(bh.Number)
	}

	if len(bh.Digest.Types) > 0 {
		//cp.Digest = make([]DigestItem, len(bh.Digest))
		copy(cp.Digest.Types[:], bh.Digest.Types[:])
	}

	return cp
}

// DeepCopy returns a deep copy of the header to prevent side effects down the road
func (bh *Header) DeepCopy() *Header {
	cp := NewEmptyHeader()
	copy(cp.ParentHash[:], bh.ParentHash[:])
	copy(cp.StateRoot[:], bh.StateRoot[:])
	copy(cp.ExtrinsicsRoot[:], bh.ExtrinsicsRoot[:])

	if bh.Number != nil {
		cp.Number = new(big.Int).Set(bh.Number)
	}

	if len(bh.Digest) > 0 {
		cp.Digest = make([]DigestItem, len(bh.Digest))
		copy(cp.Digest[:], bh.Digest[:])
	}

	return cp
}

// String returns the formatted header as a string
func (bh *HeaderVdt) String() string {
	return fmt.Sprintf("ParentHash=%s Number=%d StateRoot=%s ExtrinsicsRoot=%s Digest=%v Hash=%s",
		bh.ParentHash, bh.Number, bh.StateRoot, bh.ExtrinsicsRoot, bh.Digest, bh.Hash())
}

// String returns the formatted header as a string
func (bh *Header) String() string {
	return fmt.Sprintf("ParentHash=%s Number=%d StateRoot=%s ExtrinsicsRoot=%s Digest=%v Hash=%s",
		bh.ParentHash, bh.Number, bh.StateRoot, bh.ExtrinsicsRoot, bh.Digest, bh.Hash())
}

func (bh *HeaderVdt) Hash() common.Hash {
	if bh.hash == [32]byte{} {
		enc, err := bh.Encode()
		if err != nil {
			panic(err)
		}

		hash, err := common.Blake2bHash(enc)
		if err != nil {
			panic(err)
		}

		bh.hash = hash
	}

	return bh.hash
}

// Hash returns the hash of the block header
// If the internal hash field is nil, it hashes the block and sets the hash field.
// If hashing the header errors, this will panic.
func (bh *Header) Hash() common.Hash {
	if bh.hash == [32]byte{} {
		enc, err := bh.Encode()
		if err != nil {
			panic(err)
		}

		hash, err := common.Blake2bHash(enc)
		if err != nil {
			panic(err)
		}

		bh.hash = hash
	}

	return bh.hash
}

// Encode returns the SCALE encoding of a header
func (bh *HeaderVdt) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*bh)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

// Encode returns the SCALE encoding of a header
func (bh *Header) Encode() ([]byte, error) {
	// This works I believe
	var enc []byte
	ph, err := scale.Marshal(bh.ParentHash)
	if err != nil {
		return nil, err
	}
	enc = append(enc, ph...)

	num, err := scale.Marshal(bh.Number)
	if err != nil {
		return nil, err
	}
	enc = append(enc, num...)

	sr, err := scale.Marshal(bh.StateRoot)
	if err != nil {
		return nil, err
	}
	enc = append(enc, sr...)

	er, err := scale.Marshal(bh.ExtrinsicsRoot)
	if err != nil {
		return nil, err
	}
	enc = append(enc, er...)

	d, err := bh.Digest.Encode()
	if err != nil {
		return nil, err
	}
	enc = append(enc, d...)

	return enc, nil
}

// MustEncode returns the SCALE encoded header and panics if it fails to encode
func (bh *HeaderVdt) MustEncode() []byte {
	enc, err := bh.Encode()
	if err != nil {
		panic(err)
	}
	return enc
}

// MustEncode returns the SCALE encoded header and panics if it fails to encode
func (bh *Header) MustEncode() []byte {
	enc, err := bh.Encode()
	if err != nil {
		panic(err)
	}
	return enc
}

func Decode(bh HeaderVdt, in []byte) (HeaderVdt, error) {
	err := scale.Unmarshal(in, &bh)
	if err != nil {
		return HeaderVdt{}, err
	}
	return bh, nil
}

// Decode decodes the SCALE encoded input into this header
func (bh *HeaderVdt) Decode(in []byte) (*HeaderVdt, error) {
	var dec = NewEmptyHeaderVdt()
	err := scale.Unmarshal(in, dec)
	if err != nil {
		return nil, err
	}
	return dec, nil
}

// Decode decodes the SCALE encoded input into this header
func (bh *Header) Decode(buf *bytes.Buffer) (*Header, error) {
	sd := scale.NewDecoder(buf)

	var ph common.Hash
	err := sd.Decode(&ph)
	if err != nil {
		return nil, err
	}

	var num *big.Int
	err = sd.Decode(&num)
	if err != nil {
		return nil, err
	}

	var sr common.Hash
	err = sd.Decode(&sr)
	if err != nil {
		return nil, err
	}

	var er common.Hash
	err = sd.Decode(&er)
	if err != nil {
		return nil, err
	}

	d, err := DecodeDigest(buf)
	if err != nil {
		return nil, err
	}

	bh.ParentHash = ph
	bh.Number = num
	bh.StateRoot = sr
	bh.ExtrinsicsRoot = er
	bh.Digest = d
	return bh, nil
}

// AsOptional returns the Header as an optional.Header
func (bh *Header) AsOptional() *optional.Header {
	return optional.NewHeader(true, &optional.CoreHeader{
		ParentHash:     bh.ParentHash,
		Number:         bh.Number,
		StateRoot:      bh.StateRoot,
		ExtrinsicsRoot: bh.ExtrinsicsRoot,
		Digest:         &bh.Digest,
	})
}

// NewHeaderFromOptional returns a Header given an optional.Header. If the optional.Header is None, an error is returned.
func NewHeaderFromOptional(oh *optional.Header) (*Header, error) {
	if !oh.Exists() {
		return nil, errors.New("header is None")
	}

	h := oh.Value()

	if h.Number == nil {
		// Hash() will panic if number is nil
		return nil, errors.New("cannot have nil block number")
	}

	bh := &Header{
		ParentHash:     h.ParentHash,
		Number:         h.Number,
		StateRoot:      h.StateRoot,
		ExtrinsicsRoot: h.ExtrinsicsRoot,
		Digest:         *(h.Digest.(*Digest)),
	}

	bh.Hash()
	return bh, nil
}

// decodeOptionalHeader decodes a SCALE encoded optional Header into an *optional.Header
func decodeOptionalHeader(r *bytes.Buffer) (*optional.Header, error) {
	exists, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	if exists == 1 {
		header := &Header{
			ParentHash:     common.Hash{},
			Number:         big.NewInt(0),
			StateRoot:      common.Hash{},
			ExtrinsicsRoot: common.Hash{},
			Digest:         Digest{},
		}

		head, err := header.Decode(r)
		if err != nil {
			return nil, err
		}

		head.Hash()
		return head.AsOptional(), nil
	}

	return optional.NewHeader(false, nil), nil
}
