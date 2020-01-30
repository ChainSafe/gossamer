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
	"io"
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

func NewBlock(header *Header, body *Body, arrivalTime uint64) *Block {
	return &Block{
		Header:      header,
		Body:        body,
		arrivalTime: arrivalTime,
	}
}

func NewEmptyBlock() *Block {
	return &Block{
		Header: new(Header),
		Body:   new(Body),
	}
}

// GetBlockArrivalTime returns the arrival time for a block
func (b *Block) GetBlockArrivalTime() uint64 {
	return b.arrivalTime
}

// SetBlockArrivalTime sets the arrival time for a block
func (b *Block) SetBlockArrivalTime(t uint64) {
	b.arrivalTime = t
}

func (b *Block) Encode() ([]byte, error) {
	enc, err := scale.Encode(b.Header)
	if err != nil {
		return nil, err
	}

	// fix since scale doesn't handle *types.Body types, but does handle []byte
	encBody, err := scale.Encode([]byte(*b.Body))
	if err != nil {
		return nil, err
	}

	return append(enc, encBody...), nil
}

func (b *Block) Decode(in []byte) error {
	_, err := scale.Decode(in, b)
	return err
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

func (bh *Header) Decode(in []byte) error {
	_, err := scale.Decode(in, bh)
	return err
}

// AsOptional returns the Header as an optional.Header
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

func NewBody(b []byte) *Body {
	body := Body(b)
	return &body
}

// NewBodyFromOptional returns a Body given an optional.Body. If the optional.Body is None, an error is returned.
func NewBodyFromOptional(ob *optional.Body) (*Body, error) {
	if !ob.Exists {
		return nil, errors.New("body is None")
	}

	b := ob.Value
	res := Body([]byte(b))
	return &res, nil
}

// AsOptional returns the Body as an optional.Body
func (b *Body) AsOptional() *optional.Body {
	ob := optional.CoreBody([]byte(*b))
	return optional.NewBody(true, ob)
}

// BlockData is stored within the BlockDB
type BlockData struct {
	Hash          common.Hash
	Header        *optional.Header
	Body          *optional.Body
	Receipt       *optional.Bytes
	MessageQueue  *optional.Bytes
	Justification *optional.Bytes
}

// Encode performs SCALE encoding of the BlockData
func (bd *BlockData) Encode() ([]byte, error) {
	enc := bd.Hash[:]

	if bd.Header.Exists() {
		venc, err := scale.Encode(bd.Header.Value())
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	if bd.Body.Exists {
		venc, err := scale.Encode([]byte(bd.Body.Value))
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	if bd.Receipt.Exists() {
		venc, err := scale.Encode(bd.Receipt.Value())
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	if bd.MessageQueue.Exists() {
		venc, err := scale.Encode(bd.MessageQueue.Value())
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	if bd.Justification.Exists() {
		venc, err := scale.Encode(bd.Justification.Value())
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	return enc, nil
}

// Decode decodes the SCALE encoded input to BlockData
func (bd *BlockData) Decode(r io.Reader) error {
	hash, err := common.ReadHash(r)
	if err != nil {
		return err
	}
	bd.Hash = hash

	sd := scale.Decoder{Reader: r}

	headerExists, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	if headerExists == 1 {
		header := &Header{
			ParentHash:     common.Hash{},
			Number:         big.NewInt(0),
			StateRoot:      common.Hash{},
			ExtrinsicsRoot: common.Hash{},
			Digest:         [][]byte{}, // TODO: digests are not decoded properly due to SCALE
		}
		_, err = sd.Decode(header)
		if err != nil {
			return err
		}

		header.Hash()
		bd.Header = header.AsOptional()

		// TODO: fix SCALE :(
		common.ReadByte(r)
	} else {
		bd.Header = optional.NewHeader(false, nil)
	}

	bodyExists, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	if bodyExists == 1 {
		b, err := sd.Decode([]byte{})
		if err != nil {
			return err
		}

		body := Body(b.([]byte))
		bd.Body = body.AsOptional()
	} else {
		bd.Body = optional.NewBody(false, nil)
	}

	receiptExists, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	if receiptExists == 1 {
		b, err := sd.Decode([]byte{})
		if err != nil {
			return err
		}

		bd.Receipt = optional.NewBytes(true, b.([]byte))
	} else {
		bd.Receipt = optional.NewBytes(false, nil)
	}

	msgQueueExists, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	if msgQueueExists == 1 {
		b, err := sd.Decode([]byte{})
		if err != nil {
			return err
		}

		bd.MessageQueue = optional.NewBytes(true, b.([]byte))
	} else {
		bd.MessageQueue = optional.NewBytes(false, nil)
	}

	justificationExists, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	if justificationExists == 1 {
		b, err := sd.Decode([]byte{})
		if err != nil {
			return err
		}

		bd.Justification = optional.NewBytes(true, b.([]byte))
	} else {
		bd.Justification = optional.NewBytes(false, nil)
	}

	return nil
}

func EncodeBlockDataArray(bds []*BlockData) ([]byte, error) {
	enc, err := scale.Encode(int32(len(bds)))
	if err != nil {
		return nil, err
	}

	for _, bd := range bds {
		benc, err := bd.Encode()
		if err != nil {
			return nil, err
		}
		enc = append(enc, benc...)
	}

	return enc, nil
}

func DecodeBlockDataArray(r io.Reader) ([]*BlockData, error) {
	sd := scale.Decoder{Reader: r}

	l, err := sd.Decode(int32(0))
	if err != nil {
		return nil, err
	}

	length := int(l.(int32))
	bds := make([]*BlockData, length)

	for i := 0; i < length; i++ {
		bd := new(BlockData)
		err = bd.Decode(r)
		if err != nil {
			return bds, err
		}

		bds[i] = bd
	}

	return bds, err
}
