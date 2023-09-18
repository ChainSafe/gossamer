// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"golang.org/x/exp/maps"
)

type hashToRuntime struct {
	mutex   sync.RWMutex
	mapping map[Hash]runtime.Instance
}

func newHashToRuntime() *hashToRuntime {
	return &hashToRuntime{
		mapping: make(map[Hash]runtime.Instance),
	}
}

func (h *hashToRuntime) get(hash Hash) (instance runtime.Instance) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.mapping[hash]
}

func (h *hashToRuntime) set(hash Hash, instance runtime.Instance) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.mapping[hash] = instance
	inMemoryRuntimesGauge.Inc()
}

func (h *hashToRuntime) delete(hash Hash) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.mapping, hash)
	inMemoryRuntimesGauge.Dec()
}

func (h *hashToRuntime) hashes() (hashes []common.Hash) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return maps.Keys(h.mapping)
}

// onFinalisation handles pruning and recording on block finalisation.
// newCanonicalBlockHashes is the block hashes of the blocks newly finalised.
// The last element is the finalised block hash.
func (h *hashToRuntime) onFinalisation(newCanonicalBlockHashes []common.Hash) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if len(h.mapping) == 0 {
		logger.Warnf("no runtimes in the mapping")
		return
	}

	defer func() {
		totalInMemoryRuntimes := len(h.mapping)
		inMemoryRuntimesGauge.Set(float64(totalInMemoryRuntimes))
	}()

	finalisedHash := newCanonicalBlockHashes[len(newCanonicalBlockHashes)-1]
	// if there is only one runtime in the mapping then we should update
	// its key so we don't need to lookup the entire chain in order to find the ancestry
	if len(h.mapping) == 1 {
		uniqueAvailableInstance := maps.Values(h.mapping)[0]

		h.mapping = make(map[Hash]runtime.Instance)
		h.mapping[finalisedHash] = uniqueAvailableInstance
		return
	}

	// we procced from backwards since the last element in the newCanonicalBlockHashes
	// is the finalized one, verifying if there is a runtime instance closest to the finalized
	// hash. When we find it we clear all the map entries and keeping only the instance found
	// with the finalised hash as the key
	lastElementIdx := len(newCanonicalBlockHashes) - 1
	for idx := lastElementIdx; idx >= 0; idx-- {
		currentHash := newCanonicalBlockHashes[idx]
		inMemoryRuntime := h.mapping[currentHash]

		if inMemoryRuntime != nil {
			// stop all the running instances created by forks keeping
			// just the closest instance to the finalized block hash
			stoppedRuntimes := make(map[runtime.Instance]struct{})
			for _, runtimeToPrune := range h.mapping {
				if inMemoryRuntime != runtimeToPrune {
					_, stopped := stoppedRuntimes[runtimeToPrune]
					if !stopped {
						runtimeToPrune.Stop()
						stoppedRuntimes[runtimeToPrune] = struct{}{}
					}
				}
			}

			h.mapping = make(map[Hash]runtime.Instance)
			h.mapping[finalisedHash] = inMemoryRuntime
			return
		}
	}
}
