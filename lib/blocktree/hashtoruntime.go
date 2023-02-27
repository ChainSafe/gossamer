// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"fmt"
	"sync"
)

type hashToRuntime struct {
	mutex   sync.RWMutex
	mapping map[Hash]Runtime
	// finalisedRuntime is the current finalised block runtime pointer.
	finalisedRuntime Runtime
	// currentBlockHashes holds block hashes from the canonical chain
	// for which the current finalised block runtime is being used.
	// This is used to prune the mapping of block hash to runtime
	// when a new runtime makes it at block finalisation.
	currentBlockHashes []Hash
}

func newHashToRuntime() *hashToRuntime {
	return &hashToRuntime{
		mapping: make(map[Hash]Runtime),
	}
}

func (h *hashToRuntime) get(hash Hash) (instance Runtime) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.mapping[hash]
}

func (h *hashToRuntime) set(hash Hash, instance Runtime) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.mapping[hash] = instance
}

func (h *hashToRuntime) delete(hash Hash) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.mapping, hash)
}

// onFinalisation handles pruning and recording on block finalisation.
// newCanonicalBlockHashes is the block hashes of the blocks newly finalised.
// The last element is the finalised block hash.
func (h *hashToRuntime) onFinalisation(newCanonicalBlockHashes, prunedForkBlockHashes []Hash) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	finalisedBlockHash := newCanonicalBlockHashes[len(newCanonicalBlockHashes)-1]
	newFinalisedRuntime, ok := h.mapping[finalisedBlockHash]
	if !ok {
		panic(fmt.Sprintf("runtime not found for finalised block hash %s", finalisedBlockHash))
	}

	stoppedRuntimes := make(map[Runtime]struct{})

	// Prune runtimes from pruned forks
	for _, blockHash := range prunedForkBlockHashes {
		runtimeFromFork, ok := h.mapping[blockHash]
		if !ok {
			panic(fmt.Sprintf("runtime not found for pruned forked block hash %s", blockHash))
		}
		if runtimeFromFork != newFinalisedRuntime {
			_, stopped := stoppedRuntimes[runtimeFromFork]
			if !stopped {
				runtimeFromFork.Stop()
				stoppedRuntimes[runtimeFromFork] = struct{}{}
			}
		}
		delete(h.mapping, blockHash)
	}

	if h.finalisedRuntime == newFinalisedRuntime {
		// Runtime from the previous finalised block is the same
		// as the runtime for the new finalised block.
		// Note this logic assumes the same runtime pointer won't
		// be re-used on a runtime upgrade.
		h.currentBlockHashes = append(h.currentBlockHashes, newCanonicalBlockHashes...)
		return
	}

	// Runtime from the previous finalised block is different
	// from the runtime for the new finalised block.
	if h.finalisedRuntime != nil {
		_, stopped := stoppedRuntimes[h.finalisedRuntime]
		if !stopped {
			h.finalisedRuntime.Stop()
			stoppedRuntimes[h.finalisedRuntime] = struct{}{}
		}
	}
	h.finalisedRuntime = newFinalisedRuntime

	// Clear all block hashes using the previous finalised runtime
	for _, blockHash := range h.currentBlockHashes {
		delete(h.mapping, blockHash)
	}

	// Check each new canonical chain block hash and prune all but the
	// new finalised block corresponding runtime.
	for i, blockHash := range newCanonicalBlockHashes {
		runtime, ok := h.mapping[blockHash]
		if !ok {
			panic(fmt.Sprintf("runtime not found for canonical chain block hash %s", blockHash))
		}

		if runtime == newFinalisedRuntime {
			// The block has the new finalised block runtime so stop stopping runtimes and pruning
			// block hashes from the mapping.
			// Reset the current block hashes to the remaining new canonical chain block hashes.
			h.currentBlockHashes = h.currentBlockHashes[:0] // empty slice but keep its capacity
			h.currentBlockHashes = append(h.currentBlockHashes, newCanonicalBlockHashes[i:]...)
			break
		}

		_, stopped := stoppedRuntimes[runtime]
		if !stopped {
			runtime.Stop()
			stoppedRuntimes[runtime] = struct{}{}
		}
		delete(h.mapping, blockHash)
	}
}
