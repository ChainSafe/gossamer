// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

type tries struct {
	rootToTrie map[common.Hash]*trie.Trie
	mapMutex   sync.RWMutex
}

func newTries(t *trie.Trie) *tries {
	return &tries{
		rootToTrie: map[common.Hash]*trie.Trie{
			t.MustHash(): t,
		},
	}
}

// softSet sets the given trie at the given root hash
// in the memory map only if it is not already set.
func (t *tries) softSet(root common.Hash, trie *trie.Trie) {
	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()

	_, has := t.rootToTrie[root]
	if has {
		return
	}

	t.rootToTrie[root] = trie
}

func (t *tries) delete(root common.Hash) {
	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()
	delete(t.rootToTrie, root)
}

// get retrieves the trie corresponding to the root hash given
// from the in-memory thread safe map.
func (t *tries) get(root common.Hash) (tr *trie.Trie) {
	t.mapMutex.RLock()
	defer t.mapMutex.RUnlock()
	return t.rootToTrie[root]
}

// len returns the current numbers of tries
// stored in the in-memory map.
func (t *tries) len() int {
	t.mapMutex.RLock()
	defer t.mapMutex.RUnlock()
	return len(t.rootToTrie)
}
