// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	triesGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_storage_tries",
		Name:      "cached_total",
		Help:      "total number of tries cached in memory",
	})
	setCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "gossamer_storage_tries",
		Name:      "set_total",
		Help:      "total number of tries cached set in memory",
	})
	deleteCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "gossamer_storage_tries",
		Name:      "delete_total",
		Help:      "total number of tries deleted from memory",
	})
)

// Tries is a thread safe map of root hash
// to trie.
type Tries struct {
	rootToTrie    map[common.Hash]*trie.Trie
	mapMutex      sync.RWMutex
	db            chaindb.Database
	triesGauge    prometheus.Gauge
	setCounter    prometheus.Counter
	deleteCounter prometheus.Counter
}

// NewTries creates a new thread safe map of root hash
// to trie using the trie given as a first trie.
func NewTries(db chaindb.Database, t *trie.Trie) (trs *Tries, err error) {
	return &Tries{
		rootToTrie: map[common.Hash]*trie.Trie{
			t.MustHash(): t,
		},
		db:            db,
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}, nil
}

// getValue retrieves the value in the trie specified by trieRoot
// and at the node with the key given. It returns an error if
// the value is not found.
// Note it does NOT cache the trie in memory if it is not found
// in memory but found in the database, unlike the getTrie method.
func (t *Tries) getValue(trieRoot common.Hash, key []byte) (
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
func (t *Tries) softSetTrieInMemory(root common.Hash, trie *trie.Trie) {
	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()

	_, exists := t.rootToTrie[root]
	if exists {
		return
	}

	t.triesGauge.Inc()
	t.setCounter.Inc()
	t.rootToTrie[root] = trie
}

func (t *Tries) deleteTrieFromMemory(root common.Hash) {
	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()
	delete(t.rootToTrie, root)
	// Note we use .Set instead of .Dec in case nothing
	// was deleted since nothing existed at the hash given.
	t.triesGauge.Set(float64(len(t.rootToTrie)))
	t.deleteCounter.Inc()
}

// getTrie retrieves the trie corresponding by the root hash given,
// by first trying from memory and then from the persistent database.
// If it is absent from memory but found in the database,
// the trie is cached in memory. If it is not found at all,
// an error is returned.
func (t *Tries) getTrie(root common.Hash) (tr *trie.Trie, err error) {
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

	t.rootToTrie[root] = tr
	return tr, nil
}

// triesInMemory returns the current numbers of tries
// stored in memory.
func (t *Tries) triesInMemory() int {
	t.mapMutex.RLock()
	defer t.mapMutex.RUnlock()
	return len(t.rootToTrie)
}
