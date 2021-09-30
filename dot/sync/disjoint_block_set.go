// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package sync

import (
	"math/big"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// DisjointBlockSet represents a set of incomplete blocks, or blocks
// with an unknown parent. it is implemented by *disjointBlockSet
type DisjointBlockSet interface {
	addHashAndNumber(common.Hash, *big.Int)
	addHeader(*types.Header)
	addBlock(*types.Block)
	removeBlock(common.Hash)
	removeLowerBlocks(num *big.Int)
	hasBlock(common.Hash) bool
	getBlock(common.Hash) *pendingBlock
	getBlocks() []*pendingBlock
	size() int
}

// pendingBlock stores a block that we know of but it not yet ready to be processed
// this is a different type than *types.Block because we may wish to set the block
// hash and number without knowing the entire header yet
// this allows us easily to check which fields are missing
type pendingBlock struct {
	hash   common.Hash
	number *big.Int
	header *types.Header
	body   *types.Body
}

// disjointBlockSet contains a list of incomplete (pending) blocks
// the header may have empty fields; they may have hash and number only,
// or they may have all their header fields, or they may be complete.
//
// if the header is complete, but the body is missing, then we need to request
// the block body.
//
// if the block is complete, we may not know of its parent.
type disjointBlockSet struct {
	// TODO: put a limit on the size of this set
	sync.RWMutex
	blocks map[common.Hash]*pendingBlock
}

func newDisjointBlockSet() *disjointBlockSet {
	return &disjointBlockSet{
		blocks: make(map[common.Hash]*pendingBlock),
	}
}

func (s *disjointBlockSet) addHashAndNumber(hash common.Hash, number *big.Int) {
	s.Lock()
	defer s.Unlock()

	if _, has := s.blocks[hash]; has {
		return
	}

	s.blocks[hash] = &pendingBlock{
		hash:   hash,
		number: number,
	}
}

func (s *disjointBlockSet) addHeader(header *types.Header) {
	s.Lock()
	defer s.Unlock()

	hash := header.Hash()
	b, has := s.blocks[hash]
	if has {
		b.header = header
		return
	}

	s.blocks[hash] = &pendingBlock{
		hash:   hash,
		number: header.Number,
		header: header,
	}
}

func (s *disjointBlockSet) addBlock(block *types.Block) {
	s.Lock()
	defer s.Unlock()

	hash := block.Header.Hash()
	b, has := s.blocks[hash]
	if has {
		b.header = &block.Header
		b.body = &block.Body
		return
	}

	s.blocks[hash] = &pendingBlock{
		hash:   hash,
		number: block.Header.Number,
		header: &block.Header,
		body:   &block.Body,
	}
}

func (s *disjointBlockSet) removeBlock(hash common.Hash) {
	s.Lock()
	defer s.Unlock()
	delete(s.blocks, hash)
}

// removeLowerBlocks removes all blocks with a number equal or less than the given number
// from the set. it should be called when a new block is finalised to cleanup the set.
func (s *disjointBlockSet) removeLowerBlocks(num *big.Int) {
	s.Lock()
	defer s.Unlock()

	for hash, b := range s.blocks {
		if b.number.Cmp(num) <= 0 {
			delete(s.blocks, hash)
		}
	}
}

func (s *disjointBlockSet) hasBlock(hash common.Hash) bool {
	s.RLock()
	defer s.RUnlock()
	_, has := s.blocks[hash]
	return has
}

func (s *disjointBlockSet) size() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.blocks)
}

func (s *disjointBlockSet) getBlock(hash common.Hash) *pendingBlock {
	s.RLock()
	defer s.RUnlock()
	return s.blocks[hash]
}

func (s *disjointBlockSet) getBlocks() []*pendingBlock {
	s.RLock()
	defer s.RUnlock()

	blocks := make([]*pendingBlock, len(s.blocks))
	i := 0
	for _, b := range s.blocks {
		blocks[i] = b
		i++
	}
	return blocks
}
