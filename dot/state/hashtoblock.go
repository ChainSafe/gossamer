// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// hashToBlockMap implements a thread safe map of block header hashes
// to block pointers. It has helper methods to fit the needs of callers
// in this package.
type hashToBlockMap struct {
	mutex   sync.RWMutex
	mapping map[common.Hash]*types.Block
}

func newHashToBlockMap() *hashToBlockMap {
	return &hashToBlockMap{
		mapping: make(map[common.Hash]*types.Block),
	}
}

// getBlock returns a pointer to the block stored at the hash given,
// or nil if not found.
func (h *hashToBlockMap) getBlock(hash common.Hash) (block *types.Block) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.mapping[hash]
}

// getBlockHeader returns a pointer to the header of the block stored at the
// hash given, or nil if not found.
func (h *hashToBlockMap) getBlockHeader(hash common.Hash) (header *types.Header) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	block := h.mapping[hash]
	if block == nil {
		return nil
	}
	return &block.Header
}

// getBlockBody returns a pointer to the body of the block stored at the
// hash given, or nil if not found.
func (h *hashToBlockMap) getBlockBody(hash common.Hash) (body *types.Body) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	block := h.mapping[hash]
	if block == nil {
		return nil
	}
	return &block.Body
}

// store stores a block and uses its header hash digest as key.
func (h *hashToBlockMap) store(block *types.Block) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.mapping[block.Header.Hash()] = block
}

// delete deletes the block stored at the hash given, and returns
// a pointer to the header of the block deleted from the map,
// or nil if the block is not found.
func (h *hashToBlockMap) delete(hash common.Hash) (deletedHeader *types.Header) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	block := h.mapping[hash]
	delete(h.mapping, hash)
	if block == nil {
		return nil
	}
	return &block.Header
}
