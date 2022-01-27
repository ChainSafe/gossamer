// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

type tries struct {
	rootToTrie map[common.Hash]*trie.Trie
	db         chaindb.Database
	mapMutex   sync.RWMutex
}

func newTries(db chaindb.Database, t *trie.Trie) *tries {
	return &tries{
		rootToTrie: map[common.Hash]*trie.Trie{
			t.MustHash(): t,
		},
		db: db,
	}
}

// getValue retrieves the value in the trie specified by trieRoot
// and at the node with the key given. It returns an error if
// the value is not found.
// Note it does NOT cache the trie in memory if it is not found
// in memory but found in the database, unlike the getTrie method.
func (t *tries) getValue(trieRoot common.Hash, key []byte) (
	value []byte, err error) {
	// Try to get from memory
	t.mapMutex.RLock()
	tr, has := t.rootToTrie[trieRoot]
	t.mapMutex.RUnlock()
	if has {
		value = tr.Get(key)
		return value, nil
	}

	// Get from persistent database
	value, err = trie.GetFromDB(t.db, trieRoot, key)
	if err != nil {
		return nil, fmt.Errorf("cannot get value from database: %w", err)
	}

	return value, nil
}

// softSetTrieInMemory sets the given trie at the given root hash
// in the memory map only if it is not already set.
func (t *tries) softSetTrieInMemory(root common.Hash, trie *trie.Trie) {
	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()

	_, exists := t.rootToTrie[root]
	if exists {
		return
	}

	t.rootToTrie[root] = trie
}

func (t *tries) deleteTrieFromMemory(root common.Hash) {
	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()
	delete(t.rootToTrie, root)
}

var (
	ErrTrieUnexpectedRootHash = errors.New("trie has an unexpected root hash")
)

// getTrie retrieves the trie corresponding by the root hash given,
// by first trying from memory and then from the persistent database.
// If it is absent from memory but found in the database,
// the trie is cached in memory. If it is not found at all,
// an error is returned.
func (t *tries) getTrie(root common.Hash) (tr *trie.Trie, err error) {
	if root == trie.EmptyHash {
		return trie.NewEmptyTrie(), nil
	}

	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()

	tr, has := t.rootToTrie[root]
	if has {
		return tr, nil
	}

	tr = trie.NewEmptyTrie()
	err = tr.Load(t.db, root)
	if err != nil {
		return nil, fmt.Errorf("cannot load root from database: %w", err)
	}

	calculatedRootHash := tr.MustHash()
	if !calculatedRootHash.Equal(root) {
		return nil, fmt.Errorf("%w: expected %s but calculated hash is %s",
			ErrTrieUnexpectedRootHash, root, calculatedRootHash)
	}

	t.rootToTrie[root] = tr
	return tr, nil
}

// triesInMemory returns the current numbers of tries
// stored in memory.
func (t *tries) triesInMemory() int {
	t.mapMutex.RLock()
	defer t.mapMutex.RUnlock()
	return len(t.rootToTrie)
}
