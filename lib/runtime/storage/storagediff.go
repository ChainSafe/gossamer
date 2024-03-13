// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"bytes"
	"sort"
	"sync"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/codec"
)

type storageDiff struct {
	upserts        map[string][]byte
	deletes        map[string]bool
	childChangeSet map[string]*storageDiff
	mtx            sync.RWMutex
}

func newChangeSet() *storageDiff {
	return &storageDiff{
		upserts:        make(map[string][]byte),
		deletes:        make(map[string]bool),
		childChangeSet: make(map[string]*storageDiff),
		mtx:            sync.RWMutex{},
	}
}

func (cs *storageDiff) get(key string) ([]byte, bool) {
	if cs == nil {
		return nil, false
	}

	cs.mtx.RLock()
	defer cs.mtx.RUnlock()

	// Check in recent upserts if not found check if we want to delete it
	if val, ok := cs.upserts[key]; ok {
		return val, false
	} else if deleted := cs.deletes[key]; deleted {
		return nil, true
	}

	return nil, false
}

func (cs *storageDiff) upsert(key string, value []byte) {
	if cs == nil {
		return
	}

	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	// If we previously deleted this trie we have to undo that deletion
	if cs.deletes[key] {
		delete(cs.deletes, key)
	}

	cs.upserts[key] = value
}

func (cs *storageDiff) delete(key string) {
	if cs == nil {
		return
	}

	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	delete(cs.childChangeSet, key)
	delete(cs.upserts, key)
	cs.deletes[key] = true
}

func (cs *storageDiff) deleteChildLimit(keyToChild string,
	childKeys []string, limit int) (
	deleted uint32, allDeleted bool) {
	childChanges := cs.childChangeSet[keyToChild]
	if childChanges == nil {
		childChanges = newChangeSet()
	}

	if limit == -1 {
		cs.delete(keyToChild)
		deletedKeys := len(cs.childChangeSet[keyToChild].upserts) + len(childKeys)
		return uint32(deletedKeys), true
	}

	newKeys := maps.Keys(cs.upserts)
	allKeys := append(newKeys, childKeys...)
	sort.Strings(allKeys)

	for _, k := range allKeys {
		if limit == 0 {
			break
		}
		childChanges.delete(k)
		deleted++
		// Do not consider keys created during actual block execution
		// https://spec.polkadot.network/chap-host-api#id-version-2-prototype-2
		if !slices.Contains(newKeys, k) {
			limit--
		}
	}
	cs.childChangeSet[keyToChild] = childChanges

	return deleted, deleted == uint32(len(allKeys))
}

func (cs *storageDiff) clearPrefixInChild(keyToChild string, prefix []byte, childKeys []string) {
	childChanges := cs.childChangeSet[keyToChild]
	if childChanges == nil {
		childChanges = newChangeSet()
	}
	childChanges.clearPrefix(prefix, childKeys, -1)
	cs.childChangeSet[keyToChild] = childChanges
}

func (cs *storageDiff) clearPrefix(prefix []byte, trieKeys []string, limit int) (deleted uint32, allDeleted bool) {
	prefix = codec.KeyLEToNibbles(prefix)
	prefix = bytes.TrimSuffix(prefix, []byte{0})
	newKeys := maps.Keys(cs.upserts)
	allKeys := append(newKeys, trieKeys...)
	deleted = 0
	sort.Strings(allKeys)
	for _, k := range allKeys {
		if limit == 0 {
			break
		}
		keyBytes := []byte(k)
		bytes.HasPrefix(keyBytes, prefix)
		cs.delete(k)
		deleted++
		if !slices.Contains(newKeys, k) {
			limit--
		}
	}

	return deleted, deleted == uint32(len(allKeys))
}

func (cs *storageDiff) getFromChild(keyToChild, key string) ([]byte, bool) {
	if cs == nil {
		return nil, false
	}

	cs.mtx.RLock()
	defer cs.mtx.RUnlock()

	childTrieChanges := cs.childChangeSet[keyToChild]
	if childTrieChanges != nil {
		if val, ok := childTrieChanges.upserts[key]; ok {
			return val, false
		} else if deleted := childTrieChanges.deletes[key]; deleted {
			return nil, true
		}
	}

	return nil, false
}

func (cs *storageDiff) upsertChild(keyToChild, key string, value []byte) {
	if cs == nil {
		return
	}

	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	// If we previously deleted this child trie we have to undo that deletion
	if cs.deletes[keyToChild] {
		delete(cs.deletes, keyToChild)
	}

	childChanges := cs.childChangeSet[keyToChild]
	if childChanges == nil {
		childChanges = newChangeSet()
	}

	childChanges.upserts[key] = value
	cs.childChangeSet[keyToChild] = childChanges
}

func (cs *storageDiff) deleteFromChild(keyToChild, key string) {
	if cs == nil {
		return
	}

	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	childChanges := cs.childChangeSet[keyToChild]
	if childChanges == nil {
		childChanges = newChangeSet()
	} else {
		delete(cs.childChangeSet, keyToChild)
	}

	childChanges.childChangeSet[keyToChild].deletes[key] = true
}

func (cs *storageDiff) snapshot() *storageDiff {
	if cs == nil {
		panic("Trying to create snapshot from nil change set")
	}

	cs.mtx.RLock()
	defer cs.mtx.RUnlock()

	childChangeSetCopy := make(map[string]*storageDiff)
	for k, v := range cs.childChangeSet {
		childChangeSetCopy[k] = v.snapshot()
	}

	return &storageDiff{
		upserts:        maps.Clone(cs.upserts),
		deletes:        maps.Clone(cs.deletes),
		childChangeSet: childChangeSetCopy,
	}
}

func (cs *storageDiff) applyToTrie(t *trie.Trie) {
	if cs == nil {
		panic("trying to apply nil change set")
	}

	cs.mtx.RLock()
	defer cs.mtx.RUnlock()

	// Apply trie upserts
	for k, v := range cs.upserts {
		err := t.Put([]byte(k), v)
		if err != nil {
			panic("Error applying upserts changes to trie")
		}
	}

	// Apply child trie upserts
	for childKeyString, childChangeSet := range cs.childChangeSet {
		childKey := []byte(childKeyString)

		for k, v := range childChangeSet.upserts {
			err := t.PutIntoChild(childKey, []byte(k), v)
			if err != nil {
				panic("Error applying child trie changes to trie")
			}
		}

		for k := range childChangeSet.deletes {
			err := t.ClearFromChild(childKey, []byte(k))
			if err != nil {
				panic("Error applying child trie keys deletion to trie")
			}
		}
	}

	// Apply trie deletions
	for k := range cs.deletes {
		key := []byte(k)
		child, _ := t.GetChild(key)
		if child != nil {
			err := t.DeleteChild(key)
			if err != nil {
				panic("Error deleting child trie from trie")
			}
		} else {
			err := t.Delete([]byte(k))
			if err != nil {
				panic("Error deleting key from trie")
			}
		}

	}
}
