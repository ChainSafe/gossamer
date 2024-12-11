// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package testing

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
)

// / An opaque extrinsic wrapper type.
type ExtrinsicsWrapper[T any] struct {
	T T
}

func (ExtrinsicsWrapper[T]) IsSigned() *bool {
	return nil
}

// / Testing block
type Block[T any] struct {
	/// Block header
	header generic.Header[uint64, hash.H256, runtime.BlakeTwo256]
	/// List of extrinsics
	extrinsics []ExtrinsicsWrapper[T]
}

// Header returns the header.
func (b Block[T]) Header() runtime.Header[uint64, hash.H256] {
	return &b.header
}

// Extrinsics returns the block extrinsics.
func (b Block[T]) Extrinsics() []runtime.Extrinsic {
	e := make([]runtime.Extrinsic, 0)
	for _, ext := range b.extrinsics {
		e = append(e, ext)
	}
	return e
}

// Deconstruct returns both header and extrinsics.
func (b Block[T]) Deconstruct() (header runtime.Header[uint64, hash.H256], extrinsics []runtime.Extrinsic) {
	return b.Header(), b.Extrinsics()
}

// Hash returns the block hash.
func (b Block[T]) Hash() hash.H256 {
	hasher := runtime.BlakeTwo256{}
	return hasher.HashEncoded(b.header)
}
