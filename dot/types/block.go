// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Block defines a state block
type Block struct {
	Header Header
	Body   Body
}

// NewBlock returns a new Block
func NewBlock(header Header, body Body) Block {
	return Block{
		Header: header,
		Body:   body,
	}
}

// NewEmptyBlock returns a new empty Block
func NewEmptyBlock() Block {
	return Block{
		Header: *NewEmptyHeader(),
		Body:   Body(nil),
	}
}

// String returns the formatted Block string
func (b *Block) String() string {
	return fmt.Sprintf("header: %v\nbody: %v",
		&b.Header,
		b.Body,
	)
}

// Empty returns a boolean indicating is the Block is empty
func (b *Block) Empty() bool {
	return b.Header.Empty() && len(b.Body) == 0
}

// Encode returns the SCALE encoding of a block
func (b *Block) Encode() ([]byte, error) {
	enc, err := scale.Marshal(b.Header)
	if err != nil {
		return nil, err
	}

	// get a SCALE encoded block body
	encodedBody, err := scale.Marshal(b.Body)
	if err != nil {
		return nil, err
	}
	return append(enc, encodedBody...), nil
}

// MustEncode returns the SCALE encoded block and panics if it fails to encode
func (b *Block) MustEncode() []byte {
	enc, err := b.Encode()
	if err != nil {
		panic(err)
	}
	return enc
}

// DeepCopy returns a copy of the block
func (b *Block) DeepCopy() (Block, error) {
	head, err := b.Header.DeepCopy()
	if err != nil {
		return Block{}, err
	}
	return Block{
		Header: *head,
		Body:   b.Body.DeepCopy(),
	}, nil
}

// ToBlockData converts a Block to BlockData
func (b *Block) ToBlockData() *BlockData {
	return &BlockData{
		Hash:   b.Header.Hash(),
		Header: &b.Header,
		Body:   &b.Body,
	}
}
