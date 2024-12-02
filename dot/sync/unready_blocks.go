// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"iter"
	"maps"
	"slices"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type Fragment struct {
	chain []*types.BlockData
}

func NewFragment(chain []*types.BlockData) *Fragment {
	return &Fragment{chain}
}

// Filter returns a new fragments with blocks that satisfies the predicate p
func (f *Fragment) Filter(p func(*types.BlockData) bool) *Fragment {
	filtered := make([]*types.BlockData, 0, len(f.chain))
	for _, bd := range f.chain {
		if p(bd) {
			filtered = append(filtered, bd)
		}
	}
	return NewFragment(filtered)
}

// Find returns the first occurrence of a types.BlockData that
// satisfies the predicate p
func (f *Fragment) Find(p func(*types.BlockData) bool) *types.BlockData {
	for _, bd := range f.chain {
		if p(bd) {
			return bd
		}
	}

	return nil
}

// First returns the first block in the fragment or nil otherwise
func (f *Fragment) First() *types.BlockData {
	if len(f.chain) > 0 {
		return f.chain[0]
	}

	return nil
}

// Last returns the last block in the fragment or nil otherwise
func (f *Fragment) Last() *types.BlockData {
	if len(f.chain) > 0 {
		return f.chain[len(f.chain)-1]
	}

	return nil
}

// Len returns the amount of blocks in the fragment
func (f *Fragment) Len() int {
	return len(f.chain)
}

// Iter returns an iterator of the blocks in the fragment
// it enables the caller to use range keyword in the Fragment instance
func (f *Fragment) Iter() iter.Seq[*types.BlockData] {
	return func(yield func(*types.BlockData) bool) {
		for _, bd := range f.chain {
			if !yield(bd) {
				return
			}
		}
	}
}

// Concat returns a new fragment containing the concatenation
// between this fragment and the given as argument fragment
func (f *Fragment) Concat(snd *Fragment) *Fragment {
	return &Fragment{
		chain: slices.Concat(f.chain, snd.chain),
	}
}

type unreadyBlocks struct {
	mtx               sync.RWMutex
	incompleteBlocks  map[common.Hash]*types.BlockData
	disjointFragments []*Fragment
}

func newUnreadyBlocks() *unreadyBlocks {
	return &unreadyBlocks{
		incompleteBlocks:  make(map[common.Hash]*types.BlockData),
		disjointFragments: make([]*Fragment, 0),
	}
}

func (u *unreadyBlocks) newIncompleteBlock(blockHeader *types.Header) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	blockHash := blockHeader.Hash()
	u.incompleteBlocks[blockHash] = &types.BlockData{
		Hash:   blockHash,
		Header: blockHeader,
	}
}

func (u *unreadyBlocks) newDisjointFragment(frag *Fragment) {
	u.mtx.Lock()
	defer u.mtx.Unlock()
	u.disjointFragments = append(u.disjointFragments, frag)
}

// updateDisjointFragments given a set of blocks check if it
// connects to a disjoint fragment, and returns a new fragment
func (u *unreadyBlocks) updateDisjointFragments(chain *Fragment) (*Fragment, bool) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	for idx, disjointChain := range u.disjointFragments {
		var outFragment *Fragment
		if chain.Last().IsParent(disjointChain.First()) {
			outFragment = chain.Concat(disjointChain)
		}

		if disjointChain.Last().IsParent(chain.First()) {
			outFragment = disjointChain.Concat(chain)
		}

		if outFragment != nil {
			u.disjointFragments = slices.Delete(u.disjointFragments, idx, idx+1)
			return outFragment, true
		}
	}

	return nil, false
}

// updateIncompleteBlocks given a set of blocks check if they can fullfil
// incomplete blocks, the blocks that can be completed will be removed from
// the incompleteBlocks map and returned
func (u *unreadyBlocks) updateIncompleteBlocks(chain *Fragment) []*Fragment {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	completeBlocks := make([]*Fragment, 0)

	for blockData := range chain.Iter() {
		incomplete, ok := u.incompleteBlocks[blockData.Hash]
		if !ok {
			continue
		}

		incomplete.Body = blockData.Body
		incomplete.Justification = blockData.Justification

		delete(u.incompleteBlocks, blockData.Hash)
		completeBlocks = append(completeBlocks, NewFragment([]*types.BlockData{incomplete}))
	}

	return completeBlocks
}

func (u *unreadyBlocks) isIncomplete(blockHash common.Hash) bool {
	u.mtx.RLock()
	defer u.mtx.RUnlock()

	_, ok := u.incompleteBlocks[blockHash]
	return ok
}

// inDisjointFragment iterate through the disjoint fragments and
// check if the block hash and number already exists in one of them
func (u *unreadyBlocks) inDisjointFragment(blockHash common.Hash, blockNumber uint) bool {
	u.mtx.RLock()
	defer u.mtx.RUnlock()

	for _, frag := range u.disjointFragments {
		bd := frag.Find(func(bd *types.BlockData) bool {
			return bd.Header.Number == blockNumber && bd.Hash == blockHash
		})

		if bd != nil {
			return true
		}
	}

	return false
}

func (u *unreadyBlocks) removeIncompleteBlocks(del func(key common.Hash, value *types.BlockData) bool) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	maps.DeleteFunc(u.incompleteBlocks, del)
}

// pruneFragments will iterate over the disjoint fragments and check if they
// can be removed based on the del param
func (u *unreadyBlocks) pruneDisjointFragments(del func(*Fragment) bool) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	u.disjointFragments = slices.DeleteFunc(u.disjointFragments, del)
}

// LowerThanOrEq returns true if the fragment contains
// a block that has a number lower than highest finalized number
func LowerThanOrEq(blockNumber uint) func(*Fragment) bool {
	return func(f *Fragment) bool {
		return f.Find(func(bd *types.BlockData) bool {
			return bd.Header.Number <= blockNumber
		}) != nil
	}
}
