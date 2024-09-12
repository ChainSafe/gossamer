// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"maps"
	"slices"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type unreadyBlocks struct {
	mtx               sync.RWMutex
	incompleteBlocks  map[common.Hash]*types.BlockData
	disjointFragments [][]*types.BlockData
}

func newUnreadyBlocks() *unreadyBlocks {
	return &unreadyBlocks{
		incompleteBlocks:  make(map[common.Hash]*types.BlockData),
		disjointFragments: make([][]*types.BlockData, 0),
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

func (u *unreadyBlocks) newDisjointFragemnt(frag []*types.BlockData) {
	u.mtx.Lock()
	defer u.mtx.Unlock()
	u.disjointFragments = append(u.disjointFragments, frag)
}

// updateDisjointFragments given a set of blocks check if it
// connects to a disjoint fragment, if so we remove the fragment from the
// disjoint set and return the fragment concatenated with the chain argument
func (u *unreadyBlocks) updateDisjointFragments(chain []*types.BlockData) ([]*types.BlockData, bool) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	indexToChange := -1
	for idx, disjointChain := range u.disjointFragments {
		lastBlockArriving := chain[len(chain)-1]
		firstDisjointBlock := disjointChain[0]
		if formsSequence(lastBlockArriving, firstDisjointBlock) {
			indexToChange = idx
			break
		}
	}

	if indexToChange >= 0 {
		disjointChain := u.disjointFragments[indexToChange]
		u.disjointFragments = append(u.disjointFragments[:indexToChange], u.disjointFragments[indexToChange+1:]...)
		return append(chain, disjointChain...), true
	}

	return nil, false
}

// updateIncompleteBlocks given a set of blocks check if they can fullfil
// incomplete blocks, the blocks that can be completed will be removed from
// the incompleteBlocks map and returned
func (u *unreadyBlocks) updateIncompleteBlocks(chain []*types.BlockData) []*types.BlockData {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	completeBlocks := make([]*types.BlockData, 0)
	for _, blockData := range chain {
		incomplete, ok := u.incompleteBlocks[blockData.Hash]
		if !ok {
			continue
		}

		incomplete.Body = blockData.Body
		incomplete.Justification = blockData.Justification

		delete(u.incompleteBlocks, blockData.Hash)
		completeBlocks = append(completeBlocks, incomplete)
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
// check if the block hash an number already exists in one of them
func (u *unreadyBlocks) inDisjointFragment(blockHash common.Hash, blockNumber uint) bool {
	u.mtx.RLock()
	defer u.mtx.RUnlock()

	for _, frag := range u.disjointFragments {
		target := &types.BlockData{Header: &types.Header{Number: blockNumber}}
		idx, found := slices.BinarySearchFunc(frag, target,
			func(a, b *types.BlockData) int {
				switch {
				case a.Header.Number == b.Header.Number:
					return 0
				case a.Header.Number < b.Header.Number:
					return -1
				default:
					return 1
				}
			})

		if found && frag[idx].Hash == blockHash {
			return true
		}
	}

	return false
}

// removeIrrelevantFragments checks if there is blocks in the fragments that can be pruned
// given the finalised block number
func (u *unreadyBlocks) removeIrrelevantFragments(finalisedNumber uint) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	maps.DeleteFunc(u.incompleteBlocks, func(_ common.Hash, value *types.BlockData) bool {
		return value.Header.Number <= finalisedNumber
	})

	idxsToRemove := make([]int, 0, len(u.disjointFragments))
	for fragmentIdx, fragment := range u.disjointFragments {
		// the fragments are sorted in ascending order
		// starting from the latest item and going backwards
		// we have a higher chance to find the idx that has
		// a block with number lower or equal the finalised one
		idx := len(fragment) - 1
		for idx >= 0 {
			if fragment[idx].Header.Number <= finalisedNumber {
				break
			}
			idx--
		}

		updatedFragment := fragment[idx+1:]
		if len(updatedFragment) == 0 {
			idxsToRemove = append(idxsToRemove, fragmentIdx)
		} else {
			u.disjointFragments[fragmentIdx] = updatedFragment
		}
	}

	for _, idx := range idxsToRemove {
		u.disjointFragments = append(u.disjointFragments[:idx], u.disjointFragments[idx+1:]...)
	}
}
