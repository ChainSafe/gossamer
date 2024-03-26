// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"bytes"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/ChainSafe/gossamer/pkg/trie"
)

// storageDiff is a structure that stores the differences between consecutive
// states of a trie, such as those occurring during the execution of a block.
// It records updates (upserts), deletions, and changes to child tries.
// This mechanism facilitates applying state transitions efficiently.
// Changes accumulated in storageDiff can be applied to a trie using
// the `applyToTrie` method
// Note: this structure is not thread safe, be careful
type storageDiff struct {
	upserts        map[string][]byte
	deletes        map[string]bool
	childChangeSet map[string]*storageDiff
}

// newChangeSet initialises and returns a new storageDiff instance
func newStorageDiff() *storageDiff {
	return &storageDiff{
		upserts:        make(map[string][]byte),
		deletes:        make(map[string]bool),
		childChangeSet: make(map[string]*storageDiff),
	}
}

// get retrieves the value associated with the key if it's present in the
// change set and returns a boolean indicating if the key is marked for deletion
func (cs *storageDiff) get(key string) ([]byte, bool) {
	if cs == nil {
		return nil, false
	}

	// Check in recent upserts if not found check if we want to delete it
	if val, ok := cs.upserts[key]; ok {
		return val, false
	} else if deleted := cs.deletes[key]; deleted {
		return nil, true
	}

	return nil, false
}

// upsert records a new value for the key, or updates an existing value.
// If the key was previously marked for deletion, that deletion is undone
func (cs *storageDiff) upsert(key string, value []byte) {
	if cs == nil {
		return
	}

	// If we previously deleted this trie we have to undo that deletion
	if cs.deletes[key] {
		delete(cs.deletes, key)
	}

	cs.upserts[key] = value
}

// delete marks a key for deletion and removes it from upserts and
// child changesets, if present.
func (cs *storageDiff) delete(key string) {
	if cs == nil {
		return
	}

	delete(cs.childChangeSet, key)
	delete(cs.upserts, key)
	cs.deletes[key] = true
}

// deleteChildLimit deletes lexicographical sorted keys from a child trie with
// a maximum limit, potentially marking the entire child trie for deletion
// if the limit is exceeded.
// Keys created during the current block execution do not count toward the limit
// https://spec.polkadot.network/chap-host-api#id-version-2-prototype-2
func (cs *storageDiff) deleteChildLimit(keyToChild string,
	currentChildKeys []string, limit int) (
	deleted uint32, allDeleted bool) {

	childChanges := cs.childChangeSet[keyToChild]
	if childChanges == nil {
		childChanges = newStorageDiff()
	}

	if limit == -1 {
		cs.delete(keyToChild)
		deletedKeys := len(childChanges.upserts) + len(currentChildKeys)
		return uint32(deletedKeys), true
	}

	allKeys := slices.Clone(currentChildKeys)
	newKeys := maps.Keys(childChanges.upserts)
	allKeys = append(allKeys, newKeys...)
	sort.Strings(allKeys)

	for _, k := range allKeys {
		if limit == 0 {
			break
		}
		childChanges.delete(k)
		deleted++
		// Do not consider keys created during actual block execution
		if !slices.Contains(newKeys, k) {
			limit--
		}
	}
	cs.childChangeSet[keyToChild] = childChanges

	return deleted, deleted == uint32(len(allKeys))
}

// clearPrefixInChild clears keys with a specific prefix within a child trie.
func (cs *storageDiff) clearPrefixInChild(keyToChild string, prefix []byte,
	childKeys []string, limit int) (deleted uint32, allDeleted bool) {
	childChanges := cs.childChangeSet[keyToChild]
	if childChanges == nil {
		childChanges = newStorageDiff()
	}
	deleted, allDeleted = childChanges.clearPrefix(prefix, childKeys, limit)
	cs.childChangeSet[keyToChild] = childChanges

	return deleted, allDeleted
}

// clearPrefix removes all keys matching a specified prefix, within an
// optional limit. It returns the number of keys deleted and a boolean
// indicating if all keys with the prefix were removed.
func (cs *storageDiff) clearPrefix(prefix []byte, trieKeys []string, limit int) (deleted uint32, allDeleted bool) {
	allKeys := slices.Clone(trieKeys)
	newKeys := maps.Keys(cs.upserts)
	allKeys = append(allKeys, newKeys...)

	deleted = 0
	sort.Strings(allKeys)
	for _, k := range allKeys {
		if limit == 0 {
			break
		}
		keyBytes := []byte(k)
		if bytes.HasPrefix(keyBytes, prefix) {
			cs.delete(k)
			deleted++
			if !slices.Contains(newKeys, k) {
				limit--
			}
		}
	}

	return deleted, deleted == uint32(len(allKeys))
}

// getFromChild attempts to retrieve a value associated with a specific key
// from a child trie's change set identified by keyToChild.
// It returns the value and a boolean indicating if it was marked for deletion.
func (cs *storageDiff) getFromChild(keyToChild, key string) ([]byte, bool) {
	if cs == nil {
		return nil, false
	}

	childTrieChanges := cs.childChangeSet[keyToChild]
	if childTrieChanges != nil {
		value, deleted := childTrieChanges.get(key)
		return value, deleted
	}

	return nil, false
}

// upsertChild inserts or updates a value associated with a key within a
// specific child trie. If the child trie or the key was previously marked for
// deletion, this marking is reversed, and the value is updated.
func (cs *storageDiff) upsertChild(keyToChild, key string, value []byte) {
	if cs == nil {
		return
	}

	// If we previously deleted this child trie we have to undo that deletion
	if cs.deletes[keyToChild] {
		delete(cs.deletes, keyToChild)
	}

	childChanges := cs.childChangeSet[keyToChild]
	if childChanges == nil {
		childChanges = newStorageDiff()
	}

	childChanges.upserts[key] = value
	cs.childChangeSet[keyToChild] = childChanges
}

// deleteFromChild marks a key for deletion within a specific child trie.
func (cs *storageDiff) deleteFromChild(keyToChild, key string) {
	if cs == nil {
		return
	}

	childChanges := cs.childChangeSet[keyToChild]
	if childChanges == nil {
		childChanges = newStorageDiff()
	}

	childChanges.delete(key)
	cs.childChangeSet[keyToChild] = childChanges
}

// snapshot creates a deep copy of the current change set, including all upserts,
// deletions, and child trie change sets.
func (cs *storageDiff) snapshot() *storageDiff {
	if cs == nil {
		panic("Trying to create snapshot from nil change set")
	}

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

// applyToTrie applies all accumulated changes in the change set to the
// provided trie. This includes insertions, deletions, and modifications in both
// the main trie and child tries.
// In case of errors during the application of changes, the method will panic
func (cs *storageDiff) applyToTrie(t *trie.Trie) {
	if cs == nil {
		panic("trying to apply nil change set")
	}

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
				if !strings.Contains(err.Error(), trie.ErrChildTrieDoesNotExist.Error()) {
					panic("Error applying child trie keys deletion to trie")
				}
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
