// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hashing"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

// An abstraction over justification for a block's validity under a consensus algorithm.
//
// Essentially a finality proof. The exact formulation will vary between consensus
// algorithms. In the case where there are multiple valid proofs, inclusion within
// the block itself would allow swapping justifications to change the block's hash
// (and thus fork the chain). Sending a `Justification` alongside a block instead
// bypasses this problem.
//
// Each justification is provided as an encoded blob, and is tagged with an ID
// to identify the consensus engine that generated the proof (we might have
// multiple justifications from different engines for the same block).
type Justification struct {
	ConsensusEngineID
	EncodedJustification
}

// The encoded justification specific to a consensus engine.
type EncodedJustification []byte

// Collection of justifications for a given block, multiple justifications may
// be provided by different consensus engines for the same block.
type Justifications []Justification

// IntoJustification returns a copy of the encoded justification for the given consensus
// engine, if it exists
func (j Justifications) IntoJustification(engineID ConsensusEngineID) *EncodedJustification {
	for _, justification := range j {
		if justification.ConsensusEngineID == engineID {
			return &justification.EncodedJustification
		}
	}
	return nil
}

// Consensus engine unique ID.
type ConsensusEngineID [4]byte

// Header number.
type Number interface {
	~uint | ~uint32 | ~uint64
}

type Hash interface {
	constraints.Ordered

	Bytes() []byte
	String() string
}

// Abstraction around hashing
// Stupid bug in the Rust compiler believes derived
// traits must be fulfilled by all type parameters.
type Hasher[H Hash] interface {
	hashdb.Hasher[H]
	// Produce the hash of some byte-slice.
	Hash(s []byte) H

	// Produce the hash of some codec-encodable value.
	HashOf(s any) H
}

// Blake2-256 Hash implementation.
type BlakeTwo256 struct{}

// Produce the hash of some byte-slice.
func (bt256 BlakeTwo256) Hash(s []byte) hash.H256 {
	h := hashing.Blake2_256(s)
	return hash.H256(h[:])
}

// Produce the hash of some codec-encodable value.
func (bt256 BlakeTwo256) HashOf(s any) hash.H256 {
	bytes := scale.MustMarshal(s)
	return bt256.Hash(bytes)
}

var _ Hasher[hash.H256] = BlakeTwo256{}

// Something which fulfils the abstract idea of a Substrate header. It has types for a `Number`,
// a `Hash` and a `Hashing`. It provides access to an `extrinsics_root`, `state_root` and
// `parent_hash`, as well as a `digest` and a block `number`.
//
// You can also create a `new` one from those fields.
type Header[N Number, H Hash] interface {
	// Returns a reference to the header number.
	Number() N
	// Returns a reference to the parent hash.
	ParentHash() H
	// Returns the hash of the header.
	Hash() H
}

// Something which fulfils the abstract idea of a Substrate block. It has types for
// `Extrinsic` pieces of information as well as a `Header`.
//
// You can get an iterator over each of the `extrinsics` and retrieve the `header`.
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

// Something that acts like an `Extrinsic`.
type Extrinsic interface {
	// Is this `Extrinsic` signed?
	// If no information are available about signed/unsigned, `None` should be returned.
	IsSigned() *bool
}
