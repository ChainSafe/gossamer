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
	"math/big"

	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/common/optional"
)

// Extrinsic is a generic transaction whose format is verified in the runtime
type Extrinsic []byte

// Block defines a state block
type Block struct {
	Header      *Header
	Body        *Body
	arrivalTime uint64 // arrival time of this block
}

// GetBlockArrivalTime returns the arrival time for a block
func (b *Block) GetBlockArrivalTime() uint64 {
	return b.arrivalTime
}

// SetBlockArrivalTime sets the arrival time for a block
func (b *Block) SetBlockArrivalTime(t uint64) {
	b.arrivalTime = t
}

// Header is a state block header
type Header struct {
	ParentHash     common.Hash `json:"parentHash"`
	Number         *big.Int    `json:"number"`
	StateRoot      common.Hash `json:"stateRoot"`
	ExtrinsicsRoot common.Hash `json:"extrinsicsRoot"`
	Digest         [][]byte    `json:"digest"`
	hash           common.Hash
}

// NewHeader creates a new block header and sets its hash field
func NewHeader(parentHash common.Hash, number *big.Int, stateRoot common.Hash, extrinsicsRoot common.Hash, digest [][]byte) (*Header, error) {
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

// DeepCopy returns a deep copy of the header to prevent side effects down the road
func (bh *Header) DeepCopy() *Header {
	//copy everything but pointers / array
	safeCopyHeader := *bh
	//copy number ptr
	if bh.Number != nil {
		safeCopyHeader.Number = new(big.Int).Set(bh.Number)
	}
	//copy digest byte array
	if len(bh.Digest) > 0 {
		safeCopyHeader.Digest = make([][]byte, len(bh.Digest))
		copy(safeCopyHeader.Digest, bh.Digest)
	}

	return &safeCopyHeader
}

// Hash returns the hash of the block header
// If the internal hash field is nil, it hashes the block and sets the hash field.
// If hashing the header errors, this will panic.
func (bh *Header) Hash() common.Hash {
	if bh.hash == [32]byte{} {
		enc, err := scale.Encode(bh)
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

func (bh *Header) Encode() ([]byte, error) {
	return scale.Encode(bh)
}

func (bh *Header) AsOptional() *optional.Header {
	return optional.NewHeader(true, &optional.CoreHeader{
		ParentHash:     bh.ParentHash,
		Number:         bh.Number,
		StateRoot:      bh.StateRoot,
		ExtrinsicsRoot: bh.ExtrinsicsRoot,
		Digest:         bh.Digest,
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
		Digest:         h.Digest,
	}

	bh.Hash()
	return bh, nil
}

// Body is the extrinsics inside a state block
type Body []byte

// NewBodyFromOptional returns a Body given an optional.Body. If the optional.Body is None, an error is returned.
func NewBodyFromOptional(ob *optional.Body) (*Body, error) {
	if !ob.Exists {
		return nil, errors.New("body is None")
	}

	b := ob.Value
	res := Body([]byte(*b))
	return &res, nil
}

func (b *Body) AsOptional() *optional.Body {
	ob := optional.CoreBody([]byte(*b))
	return optional.NewBody(true, &ob)
}

/// BlockData is stored within the BlockDB
type BlockData struct {
	Hash          common.Hash
	Header        *optional.Header
	Body          *optional.Body
	Receipt       *optional.Bytes
	MessageQueue  *optional.Bytes
	Justification *optional.Bytes
}

func (bd *BlockData) Encode() []byte {
	enc := bd.Hash[:]

	return enc
}
