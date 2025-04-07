// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package client

import "github.com/ChainSafe/gossamer/internal/primitives/runtime"

type BadBlockData[H runtime.Hash, N runtime.Number] struct {
	number N
	hash   H
}

type BlockRules[H runtime.Hash, N runtime.Number] struct {
	bad   map[H]struct{} // Set with bad blocks
	forks map[N]H
}

// Chain specification rules lookup result.
type LookupResult interface {
	isLookupResult()
}

type (
	// Specification rules do not contain any special rules about this block
	LookupResultNotSpecial struct{}
	// The block is known to be bad and should not be imported
	LookupResultKnownBad struct{}
	// There is a specified canonical block hash for the given height
	LookupResultExpected[H runtime.Hash] struct {
		hash H
	}
)

func (LookupResultNotSpecial) isLookupResult()  {}
func (LookupResultKnownBad) isLookupResult()    {}
func (LookupResultExpected[H]) isLookupResult() {}

func NewBlockRules[
	H runtime.Hash,
	N runtime.Number,
](forkBlocks []BadBlockData[H, N], badBlocks map[H]struct{}) *BlockRules[H, N] {
	forks := make(map[N]H)
	for _, block := range forkBlocks {
		forks[block.number] = block.hash
	}

	return &BlockRules[H, N]{
		bad:   badBlocks,
		forks: forks,
	}
}

func (br *BlockRules[H, N]) MarkBad(hash H) {
	br.bad[hash] = struct{}{}
}

func (br *BlockRules[H, N]) Lookup(number N, hash H) LookupResult {
	if hashForHeight, ok := br.forks[number]; ok {
		if hashForHeight != hash {
			return LookupResultExpected[H]{hashForHeight}
		}
	}

	if _, ok := br.bad[hash]; ok {
		return LookupResultKnownBad{}
	}

	return LookupResultNotSpecial{}
}
