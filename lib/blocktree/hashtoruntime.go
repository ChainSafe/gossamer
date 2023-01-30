// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"sync"
)

type hashToRuntime struct {
	mutex   sync.RWMutex
	mapping map[Hash]Runtime
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
