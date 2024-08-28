// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type unreadyBlocks struct {
	mu               sync.Mutex
	incompleteBlocks map[common.Hash]*types.BlockData
	disjointChains   [][]*types.BlockData
}

func (u *unreadyBlocks) newHeader(blockHeader *types.Header) {
	u.mu.Lock()
	defer u.mu.Unlock()

	blockHash := blockHeader.Hash()
	u.incompleteBlocks[blockHash] = &types.BlockData{
		Hash:   blockHash,
		Header: blockHeader,
	}
}

func (u *unreadyBlocks) newFragment(frag []*types.BlockData) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.disjointChains = append(u.disjointChains, frag)
}

func (u *unreadyBlocks) updateDisjointFragments(chain []*types.BlockData) ([]*types.BlockData, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()

	indexToChange := -1
	for idx, disjointChain := range u.disjointChains {
		lastBlockArriving := chain[len(chain)-1]
		firstDisjointBlock := disjointChain[0]
		if formsSequence(lastBlockArriving, firstDisjointBlock) {
			indexToChange = idx
			break
		}
	}

	if indexToChange >= 0 {
		disjointChain := u.disjointChains[indexToChange]
		u.disjointChains = append(u.disjointChains[:indexToChange], u.disjointChains[indexToChange+1:]...)
		return append(chain, disjointChain...), true
	}

	return nil, false
}

func (u *unreadyBlocks) updateIncompleteBlocks(chain []*types.BlockData) []*types.BlockData {
	u.mu.Lock()
	defer u.mu.Unlock()

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
