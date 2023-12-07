// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	trieDBGetDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "gossamer_storage_tries",
		Name:      "get_trie_duration",
		Help:      "Time spent getting a trie from the trieDB (database / cache)",
	})
)

// TrieDB is a wrapper around a database.Table that stores tries and keep a cache of them in memory
// db is a database.Table that stores the tries to prevent colissions with keys in the same DB
type TrieDB struct {
	db                database.Table
	tries             *tries
	trieDBGetDuration prometheus.Histogram
}

// NewTrieDB creates a new TrieDB
func NewTrieDB(db database.Table) *TrieDB {
	return &TrieDB{
		db:                db,
		tries:             newTries(),
		trieDBGetDuration: trieDBGetDuration,
	}
}

// Delete deletes a trie from the database
func (tdb *TrieDB) Delete(root common.Hash) error {
	tdb.tries.delete(root)
	return tdb.db.Del(root.ToBytes())
}

// Put stores a dirty trie in the database
func (tdb *TrieDB) Put(t *trie.Trie) error {
	return t.WriteDirty(tdb.db)
}

// Get returns the trie with the given root
func (tdb *TrieDB) Get(root common.Hash) (*trie.Trie, error) {
	timer := prometheus.NewTimer(trieDBGetDuration)
	defer timer.ObserveDuration()
	// Get trie from memory
	t := tdb.tries.get(root)

	// If it doesn't exist, get it from the database and set it in memory
	if t == nil {
		var err error
		t, err = tdb.getFromDB(root)
		if err != nil {
			return nil, err
		}

		tdb.tries.softSet(root, t)
	}

	return t, nil
}

// GetKey returns the value for the given key in the trie with the given root
// TODO: I add this function to keep the compatibility but I think the caller should be responsible for getting
// the key from the trie.
func (tdb *TrieDB) GetKey(root common.Hash, key []byte) ([]byte, error) {
	t, err := tdb.Get(root)
	if err != nil {
		return nil, err
	}

	return t.Get(key), nil
}

func (tdb *TrieDB) deleteCached(root common.Hash) {
	tdb.tries.delete(root)
}

func (tdb *TrieDB) getFromDB(root common.Hash) (*trie.Trie, error) {
	t := trie.NewEmptyTrie()
	err := t.Load(tdb.db, root)
	if err != nil {
		return nil, err
	}

	return t, nil
}
