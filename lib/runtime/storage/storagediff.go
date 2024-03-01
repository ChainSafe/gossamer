// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"maps"
	"sync"

	"github.com/ChainSafe/gossamer/pkg/trie"
)

type storageDiff struct {
	upserts        map[string][]byte
	deletes        map[string]bool
	childChangeSet map[string]*storageDiff
	l              sync.RWMutex
}

func newChangeSet() *storageDiff {
	return &storageDiff{
		upserts:        make(map[string][]byte),
		deletes:        make(map[string]bool),
		childChangeSet: make(map[string]*storageDiff),
		l:              sync.RWMutex{},
	}
}

func (cs *storageDiff) get(key string) ([]byte, bool) {
	if cs == nil {
		return nil, false
	}

	cs.l.RLock()
	defer cs.l.RUnlock()

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

	cs.l.Lock()
	defer cs.l.Unlock()
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

	cs.l.Lock()
	defer cs.l.Unlock()

	delete(cs.childChangeSet, key)
	delete(cs.upserts, key)
	cs.deletes[key] = true
}

func (cs *storageDiff) getFromChild(keyToChild, key string) ([]byte, bool) {
	if cs == nil {
		return nil, false
	}

	cs.l.RLock()
	defer cs.l.RUnlock()

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

	cs.l.Lock()
	defer cs.l.Unlock()
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

	cs.l.Lock()
	defer cs.l.Unlock()

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

	cs.l.RLock()
	defer cs.l.RUnlock()

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

	cs.l.RLock()
	defer cs.l.RUnlock()

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
