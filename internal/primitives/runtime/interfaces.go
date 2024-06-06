// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hashing"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

// Number is the header number type
type Number interface {
	~uint | ~uint32 | ~uint64
}

// Hash type
type Hash interface {
	constraints.Ordered
	// Bytes returns a byte slice representation of Hash
	Bytes() []byte
	// String returns a unique string representation of the hash
	String() string
}

// Hasher is an interface around hashing
type Hasher[H Hash] interface {
	// Produce the hash of some byte-slice.
	Hash(s []byte) H

	// Produce the hash of some codec-encodable value.
	HashOf(s any) H
}

// Blake2-256 Hash implementation.
type BlakeTwo256 struct{}

// Produce the hash of some byte-slice.
func (bt256 BlakeTwo256) Hash(s []byte) hash.H256 {
	h := hashing.BlakeTwo256(s)
	return hash.H256(h[:])
}

// Produce the hash of some codec-encodable value.
func (bt256 BlakeTwo256) HashOf(s any) hash.H256 {
	bytes := scale.MustMarshal(s)
	return bt256.Hash(bytes)
}

var _ Hasher[hash.H256] = BlakeTwo256{}

// Header is the interface for a header. It has types for a `Number`,
// and `Hash`. It provides access to an `ExtrinsicsRoot`, `StateRoot` and
// `ParentHash`, as well as a `Digest` and a block `Number`.
type Header[N Number, H Hash] interface {
	// Returns a reference to the header number.
	Number() N
	// Sets the header number.
	SetNumber(number N)

	// Returns a reference to the extrinsics root.
	ExtrinsicsRoot() H
	// Sets the extrinsic root.
	SetExtrinsicsRoot(root H)

	// Returns a reference to the state root.
	StateRoot() H
	// Sets the state root.
	SetStateRoot(root H)

	// Returns a reference to the parent hash.
	ParentHash() H
	// Sets the parent hash.
	SetParentHash(hash H)

	// Returns a reference to the digest.
	Digest() Digest
	// Get a mutable reference to the digest.
	DigestMut() *Digest

	// Returns the hash of the header.
	Hash() H
}

// Block represents a block. It has types for `Extrinsic` pieces of information as well as a `Header`.
//
// You can iterate over each of the `Extrinsics` and retrieve the `Header`.
type Block[N Number, H Hash] interface {
	// Returns a reference to the header.
	Header() Header[N, H]
	// Returns a reference to the list of extrinsics.
	Extrinsics() []Extrinsic
	// Split the block into header and list of extrinsics.
	Deconstruct() (header Header[N, H], extrinsics []Extrinsic)
	// Returns the hash of the block.
	Hash() H
}

// Extrinisic is the interface for an `Extrinsic`.
type Extrinsic interface {
	// Is this `Extrinsic` signed?
	// If no information are available about signed/unsigned, `None` should be returned.
	IsSigned() *bool
}
