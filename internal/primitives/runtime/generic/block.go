// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package generic

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// BlockID is used to identify a block.
type BlockID interface {
	isBlockID()
}

// BlockIDTypes is the interface constraint of BlockID.
type BlockIDTypes[H runtime.Hash, N runtime.Number] interface {
	BlockIDHash[H] | BlockIDNumber[N]
}

// NewBlockID is the constructor for BlockID.
func NewBlockID[H runtime.Hash, N runtime.Number, T BlockIDTypes[H, N]](blockID T) BlockID {
	return BlockID(blockID)
}

// BlockIDHash is id by block header hash.
type BlockIDHash[H runtime.Hash] struct {
	Hash H
}

func (BlockIDHash[H]) isBlockID() {}
func (id BlockIDHash[H]) String() string {
	return fmt.Sprintf("%s", id.Hash)
}

// BlockIDNumber is id by block number.
type BlockIDNumber[N runtime.Number] struct {
	Number N
}

func (BlockIDNumber[N]) isBlockID() {}
func (id BlockIDNumber[H]) String() string {
	return fmt.Sprintf("%d", id.Number)
}

// Block is a block.
type Block[
	N runtime.Number,
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] struct {
	// The block header.
	header Header
	// The accompanying extrinsics.
	extrinsics []E
}

// Header returns the header.
func (b Block[N, H, Hasher, E, Header]) Header() Header {
	return b.header
}

// Extrinsics returns the block extrinsics.
func (b Block[N, H, Hasher, E, Header]) Extrinsics() []E {
	return b.extrinsics
}

// Deconstruct returns both header and extrinsics.
func (b Block[N, H, Hasher, E, Header]) Deconstruct() (header Header, extrinsics []E) {
	return b.Header(), b.Extrinsics()
}

// Hash returns the block hash.
func (b Block[N, H, Hasher, E, Header]) Hash() H {
	hasher := *new(Hasher)
	return hasher.HashEncoded(b.header)
}

// NewBlock is the constructor for Block.
func NewBlock[
	Hasher runtime.Hasher[H],
	E runtime.Extrinsic,
	N runtime.Number,
	H runtime.Hash,
	Header runtime.Header[N, H],
](
	header Header, extrinsics []E) Block[N, H, Hasher, E, Header] {
	return Block[N, H, Hasher, E, Header]{
		header:     header,
		extrinsics: extrinsics,
	}
}

type SignedBlock[
	N runtime.Number,
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] struct {
	Block          Block[N, H, Hasher, E, Header]
	Justifications runtime.Justifications
}

func NewSignedBlock[
	N runtime.Number,
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
](
	block Block[N, H, Hasher, E, Header],
	justifications runtime.Justifications,
) *SignedBlock[N, H, Hasher, E, Header] {
	return &SignedBlock[N, H, Hasher, E, Header]{
		Block:          block,
		Justifications: justifications,
	}
}

var _ runtime.Block[hash.H256, uint, runtime.OpaqueExtrinsic, Header[uint, hash.H256, runtime.BlakeTwo256]] = Block[
	uint, hash.H256, runtime.BlakeTwo256, runtime.OpaqueExtrinsic, Header[uint, hash.H256, runtime.BlakeTwo256]]{}
