// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	// ttl is the time that a block can stay in this set before being cleared.
	ttl                 = 10 * time.Minute
	clearBlocksInterval = time.Minute
)

var (
	errUnknownBlock = errors.New("cannot add justification for unknown block")
	errSetAtLimit   = errors.New("cannot add block; set is at capacity")
)

// DisjointBlockSet represents a set of incomplete blocks, or blocks
// with an unknown parent. it is implemented by *disjointBlockSet
type DisjointBlockSet interface {
	run(done <-chan struct{})
	addHashAndNumber(common.Hash, *big.Int) error
	addHeader(*types.Header) error
	addBlock(*types.Block) error
	addJustification(common.Hash, []byte) error
	removeBlock(common.Hash)
	removeLowerBlocks(num *big.Int)
	hasBlock(common.Hash) bool
	getBlock(common.Hash) *pendingBlock
	getBlocks() []*pendingBlock
	getChildren(common.Hash) map[common.Hash]struct{}
	getReadyDescendants(curr common.Hash, ready []*types.BlockData) []*types.BlockData
	size() int
}

// pendingBlock stores a block that we know of but it not yet ready to be processed
// this is a different type than *types.Block because we may wish to set the block
// hash and number without knowing the entire header yet
// this allows us easily to check which fields are missing
type pendingBlock struct {
	hash          common.Hash
	number        *big.Int
	header        *types.Header
	body          *types.Body
	justification []byte

	// the time when this block should be cleared from the set.
	// if the block is re-added to the set, this time get updated.
	clearAt time.Time
}

func newPendingBlock(hash common.Hash, number *big.Int,
	header *types.Header, body *types.Body, clearAt time.Time) *pendingBlock {
	return &pendingBlock{
		hash:    hash,
		number:  number,
		header:  header,
		body:    body,
		clearAt: clearAt,
	}
}

func (b *pendingBlock) toBlockData() *types.BlockData {
	if b.justification == nil {
		return &types.BlockData{
			Hash:   b.hash,
			Header: b.header,
			Body:   b.body,
		}
	}

	return &types.BlockData{
		Hash:          b.hash,
		Header:        b.header,
		Body:          b.body,
		Justification: &b.justification,
	}
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
	sync.RWMutex
	limit int

	// map of block hash -> block data
	blocks map[common.Hash]*pendingBlock

	// map of parent hash -> child hashes
	parentToChildren map[common.Hash]map[common.Hash]struct{}

	timeNow func() time.Time
}

func newDisjointBlockSet(limit int) *disjointBlockSet {
	return &disjointBlockSet{
		blocks:           make(map[common.Hash]*pendingBlock),
		parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
		limit:            limit,
		timeNow:          time.Now,
	}
}

func (s *disjointBlockSet) run(done <-chan struct{}) {
	ticker := time.NewTicker(clearBlocksInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.clearBlocks()
		case <-done:
			return
		}
	}
}

func (s *disjointBlockSet) clearBlocks() {
	s.Lock()
	defer s.Unlock()

	for _, block := range s.blocks {
		if s.timeNow().Sub(block.clearAt) > 0 {
			s.removeBlockInner(block.hash)
		}
	}
}

func (s *disjointBlockSet) addToParentMap(parent, child common.Hash) {
	children, has := s.parentToChildren[parent]
	if !has {
		children = make(map[common.Hash]struct{})
		s.parentToChildren[parent] = children
	}

	children[child] = struct{}{}
}

func (s *disjointBlockSet) addHashAndNumber(hash common.Hash, number *big.Int) error {
	s.Lock()
	defer s.Unlock()

	if b, has := s.blocks[hash]; has {
		b.clearAt = s.timeNow().Add(ttl)
		return nil
	}

	if len(s.blocks) == s.limit {
		return errSetAtLimit
	}

	s.blocks[hash] = newPendingBlock(hash, number, nil, nil, s.timeNow().Add(ttl))
	return nil
}

func (s *disjointBlockSet) addHeader(header *types.Header) error {
	s.Lock()
	defer s.Unlock()

	hash := header.Hash()
	if b, has := s.blocks[hash]; has {
		b.header = header
		b.clearAt = s.timeNow().Add(ttl)
		return nil
	}

	if len(s.blocks) == s.limit {
		return errSetAtLimit
	}

	s.blocks[hash] = newPendingBlock(hash, header.Number, header, nil, s.timeNow().Add(ttl))
	s.addToParentMap(header.ParentHash, hash)
	return nil
}

func (s *disjointBlockSet) addBlock(block *types.Block) error {
	s.Lock()
	defer s.Unlock()

	hash := block.Header.Hash()
	if b, has := s.blocks[hash]; has {
		b.header = &block.Header
		b.body = &block.Body
		b.clearAt = s.timeNow().Add(ttl)
		return nil
	}

	if len(s.blocks) == s.limit {
		return errSetAtLimit
	}

	s.blocks[hash] = newPendingBlock(hash, block.Header.Number, &block.Header, &block.Body, s.timeNow().Add(ttl))
	s.addToParentMap(block.Header.ParentHash, hash)
	return nil
}

func (s *disjointBlockSet) addJustification(hash common.Hash, just []byte) error {
	s.Lock()
	defer s.Unlock()

	b, has := s.blocks[hash]
	if has {
		b.justification = just
		b.clearAt = time.Now().Add(ttl)
		return nil
	}

	// block number must not be nil if we are storing a justification for it
	return errUnknownBlock
}

func (s *disjointBlockSet) removeBlock(hash common.Hash) {
	s.Lock()
	defer s.Unlock()
	s.removeBlockInner(hash)
}

// this function does not lock!!
// it should only be called by other functions in this file that lock the set beforehand.
func (s *disjointBlockSet) removeBlockInner(hash common.Hash) {
	block, has := s.blocks[hash]
	if !has {
		return
	}

	// clear block from parent->child map if its parent was known
	if block.header != nil {
		delete(s.parentToChildren[block.header.ParentHash], hash)
		if len(s.parentToChildren[block.header.ParentHash]) == 0 {
			delete(s.parentToChildren, block.header.ParentHash)
		}
	}

	delete(s.blocks, hash)
}

// removeLowerBlocks removes all blocks with a number equal or less than the given number
// from the set. it should be called when a new block is finalised to cleanup the set.
func (s *disjointBlockSet) removeLowerBlocks(num *big.Int) {
	blocks := s.getBlocks()
	for _, block := range blocks {
		if block.number.Cmp(num) <= 0 {
			s.removeBlock(block.hash)
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

func (s *disjointBlockSet) getChildren(hash common.Hash) map[common.Hash]struct{} {
	s.RLock()
	defer s.RUnlock()
	return s.parentToChildren[hash]
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

// getReadyDescendants recursively checks for descendants that are ready to be processed
func (s *disjointBlockSet) getReadyDescendants(curr common.Hash, ready []*types.BlockData) []*types.BlockData {
	children := s.getChildren(curr)
	if len(children) == 0 {
		return ready
	}

	for c := range children {
		b := s.getBlock(c)
		if b == nil || b.header == nil || b.body == nil {
			continue
		}

		// if the entire block's data is known, it's ready!
		ready = append(ready, b.toBlockData())
		ready = s.getReadyDescendants(c, ready)
	}

	return ready
}
