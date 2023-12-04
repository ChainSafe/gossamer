// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
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

const MaxInMemoryTries = 100

// Tries is a thread safe map of root hash
// to trie.
type Tries struct {
	rootToTrie    *lrucache.LRUCache[common.Hash, *trie.Trie]
	triesGauge    prometheus.Gauge
	setCounter    prometheus.Counter
	deleteCounter prometheus.Counter
}

// NewTries creates a new thread safe map of root hash
// to trie.
func NewTries() (tries *Tries) {
	cache := lrucache.NewLRUCache[common.Hash, *trie.Trie](MaxInMemoryTries)

	return &Tries{
		rootToTrie:    cache,
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}
}

// SetEmptyTrie sets the empty trie in the tries.
// Note the empty trie is the same for the v0 and the v1
// state trie versions.
func (t *Tries) SetEmptyTrie() {
	t.softSet(trie.EmptyHash, trie.NewEmptyTrie())
}

// SetTrie sets the trie at its root hash in the tries map.
func (t *Tries) SetTrie(trie *trie.Trie) {
	t.softSet(trie.MustHash(), trie)
}

// softSet sets the given trie at the given root hash
// in the memory map only if it is not already set.
func (t *Tries) softSet(root common.Hash, trie *trie.Trie) {
	if t.rootToTrie.SoftPut(root, trie) {
		t.triesGauge.Inc()
		t.setCounter.Inc()
	}
}

func (t *Tries) delete(root common.Hash) {
	if t.rootToTrie.Delete(root) {
		t.triesGauge.Dec()
		t.deleteCounter.Inc()
	}
}

// get retrieves the trie corresponding to the root hash given
// from the in-memory thread safe map.
func (t *Tries) get(root common.Hash) (tr *trie.Trie) {
	return t.rootToTrie.Get(root)
}

// len returns the current numbers of tries
// stored in the in-memory map.
func (t *Tries) len() int {
	return t.rootToTrie.Len()
}
