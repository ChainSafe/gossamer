// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"sync"

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
	triesGauge    prometheus.Gauge
	setCounter    prometheus.Counter
	deleteCounter prometheus.Counter
}

// NewTries creates a new thread safe map of root hash
// to trie using the trie given as a first trie.
func NewTries(t *trie.Trie) (trs *Tries, err error) {
	return &Tries{
		rootToTrie: map[common.Hash]*trie.Trie{
			t.MustHash(): t,
		},
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}, nil
}

// softSet sets the given trie at the given root hash
// in the memory map only if it is not already set.
func (t *Tries) softSet(root common.Hash, trie *trie.Trie) {
	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()

	_, has := t.rootToTrie[root]
	if has {
		return
	}

	t.triesGauge.Inc()
	t.setCounter.Inc()
	t.rootToTrie[root] = trie
}

func (t *Tries) delete(root common.Hash) {
	t.mapMutex.Lock()
	defer t.mapMutex.Unlock()
	delete(t.rootToTrie, root)
	// Note we use .Set instead of .Dec in case nothing
	// was deleted since nothing existed at the hash given.
	t.triesGauge.Set(float64(len(t.rootToTrie)))
	t.deleteCounter.Inc()
}

// get retrieves the trie corresponding to the root hash given
// from the in-memory thread safe map.
func (t *Tries) get(root common.Hash) (tr *trie.Trie) {
	t.mapMutex.RLock()
	defer t.mapMutex.RUnlock()
	return t.rootToTrie[root]
}

// len returns the current numbers of tries
// stored in the in-memory map.
func (t *Tries) len() int {
	t.mapMutex.RLock()
	defer t.mapMutex.RUnlock()
	return len(t.rootToTrie)
}
