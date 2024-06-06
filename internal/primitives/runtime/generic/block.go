// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package generic

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// Something to identify a block.
type BlockID any

// BlockIDTypes is the interface constraint of `BlockID`.
type BlockIDTypes[H, N any] interface {
	BlockIDHash[H] | BlockIDNumber[N]
}

// NewBlockID is the constructor for `BlockID`.
func NewBlockID[H, N any, T BlockIDTypes[H, N]](blockID T) BlockID {
	return BlockID(blockID)
}

// BlockIDHash is id by block header hash.
type BlockIDHash[H any] struct {
	Inner H
}

// BlockIDNumber is id by block number.
type BlockIDNumber[N any] struct {
	Inner N
}

// Block is a block.
type Block[N runtime.Number, H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	// The block header.
	header runtime.Header[N, H]
	// The accompanying extrinsics.
	extrinsics []runtime.Extrinsic
}

// Header returns the header.
func (b Block[N, H, Hasher]) Header() runtime.Header[N, H] {
	return b.header
}

// Extrinsics returns the block extrinsics.
func (b Block[N, H, Hasher]) Extrinsics() []runtime.Extrinsic {
	return b.extrinsics
}

// Deconstruct returns both header and extrinsics.
func (b Block[N, H, Hasher]) Deconstruct() (header runtime.Header[N, H], extrinsics []runtime.Extrinsic) {
	return b.Header(), b.Extrinsics()
}

// Hash returns the block hash.
func (b Block[N, H, Hasher]) Hash() H {
	hasher := *new(Hasher)
	return hasher.HashOf(b.header)
}

// NewBlock is the constructor for `Block`.
func NewBlock[N runtime.Number, H runtime.Hash, Hasher runtime.Hasher[H]](
	header runtime.Header[N, H], extrinsics []runtime.Extrinsic) Block[N, H, Hasher] {
	return Block[N, H, Hasher]{
		header:     header,
		extrinsics: extrinsics,
	}
}

var _ runtime.Block[uint, hash.H256] = Block[uint, hash.H256, runtime.BlakeTwo256]{}
