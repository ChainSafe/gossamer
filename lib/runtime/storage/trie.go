// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"golang.org/x/exp/maps"
)

// TrieState relies on `storageDiff` to perform changes over the current state.
// It has support for transactions using "nested" storageDiff changes
// If the execution of the call is successful, the changes will be applied to
// the current `state`
type TrieState struct {
	mtx          sync.RWMutex
	state        *trie.Trie
	transactions *list.List
}

// NewTrieState initialises and returns a new TrieState instance
func NewTrieState(initialState *trie.Trie) *TrieState {
	transactions := list.New()
	return &TrieState{
		transactions: transactions,
		state:        initialState,
	}
}

func (t *TrieState) getCurrentTransaction() *storageDiff {
	innerTransaction := t.transactions.Back()
	if innerTransaction == nil {
		return nil
	}
	return innerTransaction.Value.(*storageDiff)
}

func (t *TrieState) SetVersion(v trie.TrieLayout) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.state.SetVersion(v)
}

// StartTransaction begins a new nested storage transaction
// which will either be committed or rolled back at a later time.
func (t *TrieState) StartTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	nextChangeSet := t.getCurrentTransaction()
	if nextChangeSet == nil {
		nextChangeSet = newStorageDiff()
	}

	t.transactions.PushBack(nextChangeSet.snapshot())
}

// RollbackTransaction back all storage changes made since StartTransaction was called.
func (t *TrieState) RollbackTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.transactions.Len() < 1 {
		panic("no transactions to rollback")
	}

	t.transactions.Remove(t.transactions.Back())
}

// CommitTransaction all storage changes made since StartTransaction was called.
func (t *TrieState) CommitTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.transactions.Len() == 0 {
		panic("no transactions to commit")
	}

	if t.transactions.Len() > 1 {
		// We merge this transaction with its parent transaction
		t.transactions.Back().Prev().Value = t.transactions.Remove(t.transactions.Back())
	} else {
		// This is the last transaction so we apply all the changes to our state
		t.transactions.Remove(t.transactions.Back()).(*storageDiff).applyToTrie(t.state)
	}
}

// Trie returns the TrieState's underlying trie
func (t *TrieState) Trie() *trie.Trie {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.state
}

// Put puts a key-value pair in the trie
func (t *TrieState) Put(key, value []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	// If we have running transactions we apply the change there,
	// if not, we apply the changes directly on our state trie
	if t.getCurrentTransaction() != nil {
		t.getCurrentTransaction().upsert(string(key), value)
		return nil
	} else {
		return t.state.Put(key, value)
	}
}

// Get gets a value from the trie
func (t *TrieState) Get(key []byte) []byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	// If we find the key or it is deleted return from latest transaction
	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		val, deleted := currentTx.get(string(key))
		if val != nil || deleted {
			return val
		}
	}

	// If we didn't find the key in the latest transactions lookup from state
	return t.state.Get(key)
}

// MustRoot returns the trie's root hash. It panics if it fails to compute the root.
func (t *TrieState) MustRoot() common.Hash {
	hash, err := t.Root()
	if err != nil {
		panic(err)
	}

	return hash
}

// Root returns the trie's root hash
func (t *TrieState) Root() (common.Hash, error) {
	// Since the Root function is called without running transactions we can do:
	return t.state.Hash()
}

// Has returns whether or not a key exists
func (t *TrieState) Has(key []byte) bool {
	return t.Get(key) != nil
}

// Delete deletes a key from the trie
func (t *TrieState) Delete(key []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		t.getCurrentTransaction().delete(string(key))
		return nil
	}

	return t.state.Delete(key)
}

// NextKey returns the next key in the trie in lexicographical order. If it does not exist, it returns nil.
func (t *TrieState) NextKey(key []byte) []byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		allEntries := t.state.Entries()
		maps.Copy(allEntries, currentTx.upserts)

		keys := maps.Keys(allEntries)
		sort.Strings(keys)

		for _, k := range keys {
			if k > string(key) && !currentTx.deletes[k] {
				return allEntries[k]
			}
		}
	}

	return t.state.NextKey(key)
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (t *TrieState) ClearPrefix(prefix []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		trieKeys := t.state.Entries()
		currentTx.clearPrefix(prefix, maps.Keys(trieKeys), -1)
		return
	}

	return t.state.ClearPrefix(prefix)
}

// ClearPrefixLimit deletes key-value pairs from the trie where the key starts with the given prefix till limit reached
func (t *TrieState) ClearPrefixLimit(prefix []byte, limit uint32) (
	deleted uint32, allDeleted bool, err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		trieKeys := t.state.Entries()
		deleted, allDeleted = currentTx.clearPrefix(prefix, maps.Keys(trieKeys), int(limit))
		return deleted, allDeleted, nil
	}

	return t.state.ClearPrefixLimit(prefix, limit)
}

// TrieEntries returns every key-value pair in the trie
func (t *TrieState) TrieEntries() map[string][]byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	entries := make(map[string][]byte)

	// Get entries from original trie
	maps.Copy(entries, t.state.Entries())

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		// Overwrite it with last changes
		maps.Copy(entries, t.getCurrentTransaction().upserts)

		// Remove deleted keys
		for k := range t.getCurrentTransaction().deletes {
			delete(entries, k)
		}
	}

	return entries
}

// SetChildStorage sets a key-value pair in a child trie
func (t *TrieState) SetChildStorage(keyToChild, key, value []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		keyToChildStr := string(keyToChild)
		keyString := string(key)
		currentTx.upsertChild(keyToChildStr, keyString, value)
		return nil
	}

	return t.state.PutIntoChild(keyToChild, key, value)
}

func (t *TrieState) GetChildRoot(keyToChild []byte) (common.Hash, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	child, err := t.state.GetChild(keyToChild)
	if err != nil {
		return common.EmptyHash, err
	}

	return child.Hash()
}

// GetChildStorage returns a value from a child trie
func (t *TrieState) GetChildStorage(keyToChild, key []byte) ([]byte, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		val, deleted := currentTx.getFromChild(string(keyToChild), string(key))
		if val != nil || deleted {
			return val, nil
		}
	}

	// If we didnt find the key in the latest transactions lookup from state
	return t.state.GetFromChild(keyToChild, key)
}

// DeleteChild deletes a child trie from the main trie
func (t *TrieState) DeleteChild(keyToChild []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		currentTx.delete(string(keyToChild))
		return nil
	}

	return t.state.DeleteChild(keyToChild)
}

// DeleteChildLimit deletes up to limit of database entries by lexicographic order.
func (t *TrieState) DeleteChildLimit(key []byte, limit *[]byte) (
	deleted uint32, allDeleted bool, err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		deleteLimit := -1
		if limit != nil {
			deleteLimit = int(binary.LittleEndian.Uint32(*limit))
		}

		childKey := string(key)

		child, err := t.state.GetChild(key)

		childEntriesKeys := make([]string, 0)
		if err != nil {
			// If child trie does not exists and won't be created return err
			if currentTx.childChangeSet[childKey] == nil {
				return 0, false, err
			}
		} else {
			childEntriesKeys = maps.Keys(child.Entries())
		}

		deleted, allDeleted = currentTx.deleteChildLimit(childKey, childEntriesKeys, deleteLimit)
		return deleted, allDeleted, nil
	}

	child, err := t.state.GetChild(key)
	if err != nil {
		return 0, false, err
	}

	childTrieEntries := child.Entries()
	qtyEntries := uint32(len(childTrieEntries))
	if limit == nil {
		err = t.state.DeleteChild(key)
		if err != nil {
			return 0, false, fmt.Errorf("deleting child trie: %w", err)
		}

		return qtyEntries, true, nil
	}
	limitUint := binary.LittleEndian.Uint32(*limit)

	keys := maps.Keys(childTrieEntries)
	sort.Strings(keys)
	for _, k := range keys {
		// TODO have a transactional/atomic way to delete multiple keys in trie.
		// If one deletion fails, the child trie and its parent trie are then in
		// a bad intermediary state. Take also care of the caching of deleted Merkle
		// values within the tries, which is used for online pruning.
		// See https://github.com/ChainSafe/gossamer/issues/3032
		err = child.Delete([]byte(k))
		if err != nil {
			return deleted, allDeleted, fmt.Errorf("deleting from child trie located at key 0x%x: %w", key, err)
		}

		deleted++
		if deleted == limitUint {
			break
		}
	}

	allDeleted = deleted == qtyEntries
	return deleted, allDeleted, nil
}

// ClearChildStorage removes the child storage entry from the trie
func (t *TrieState) ClearChildStorage(keyToChild, key []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		keyToChildStr := string(keyToChild)
		keyStr := string(key)
		currentTx.deleteFromChild(keyToChildStr, keyStr)
		return nil
	}

	return t.state.ClearFromChild(keyToChild, key)
}

// ClearPrefixInChild clears all the keys from the child trie that have the given prefix
func (t *TrieState) ClearPrefixInChild(keyToChild, prefix []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		child, err := t.state.GetChild(keyToChild)
		childKeys := make([]string, 0)
		if err == nil {
			childKeys = maps.Keys(child.Entries())
		}

		currentTx.clearPrefixInChild(string(keyToChild), prefix, childKeys, -1)
		return nil
	}

	child, err := t.state.GetChild(keyToChild)
	if err != nil {
		return err
	}
	if child == nil {
		return nil
	}

	err = child.ClearPrefix(prefix)
	if err != nil {
		return fmt.Errorf("clearing prefix in child trie located at key 0x%x: %w", keyToChild, err)
	}

	return nil
}

func (t *TrieState) ClearPrefixInChildWithLimit(keyToChild, prefix []byte, limit uint32) (uint32, bool, error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		child, err := t.state.GetChild(keyToChild)
		childKeys := make([]string, 0)
		if err == nil {
			childKeys = maps.Keys(child.Entries())
		}

		deleted, allDeleted := currentTx.clearPrefixInChild(string(keyToChild), prefix, childKeys, int(limit))
		return deleted, allDeleted, nil
	}

	child, err := t.state.GetChild(keyToChild)
	if err != nil || child == nil {
		return 0, false, err
	}

	return child.ClearPrefixLimit(prefix, limit)
}

// GetChildNextKey returns the next lexicographical larger key from child storage. If it does not exist, it returns nil.
func (t *TrieState) GetChildNextKey(keyToChild, key []byte) ([]byte, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		// If we are going to delete this child we return error
		if currentTx.deletes[string(keyToChild)] {
			return nil, trie.ErrChildTrieDoesNotExist
		}

		if childChanges := currentTx.childChangeSet[string(keyToChild)]; childChanges != nil {
			allEntries := make(map[string][]byte)

			maps.Copy(allEntries, childChanges.upserts)
			child, err := t.state.GetChild(keyToChild)
			if err != nil {
				// Child trie does not exists and won't exists in the future
				if len(allEntries) == 0 {
					return nil, err
				}
			} else {
				allEntries = child.Entries()
			}
			keys := maps.Keys(allEntries)
			sort.Strings(keys)

			for _, k := range keys {
				if k > string(key) && !childChanges.deletes[k] {
					return allEntries[k], nil
				}
			}
			return nil, nil
		}
	}

	child, err := t.state.GetChild(keyToChild)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, nil
	}

	return child.NextKey(key), nil
}

// GetKeysWithPrefixFromChild ...
func (t *TrieState) GetKeysWithPrefixFromChild(keyToChild, prefix []byte) ([][]byte, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	if currentTx := t.getCurrentTransaction(); currentTx != nil {
		// If we are going to delete this child we return error
		if currentTx.deletes[string(keyToChild)] {
			return nil, trie.ErrChildTrieDoesNotExist
		}

		if childChanges := currentTx.childChangeSet[string(keyToChild)]; childChanges != nil {
			allEntries := make(map[string][]byte)

			maps.Copy(allEntries, childChanges.upserts)
			child, err := t.state.GetChild(keyToChild)
			if err != nil {
				// Child trie does not exists and won't exists in the future
				if len(allEntries) == 0 {
					return nil, err
				}
			} else {
				allEntries = child.Entries()
			}
			keys := maps.Keys(allEntries)

			values := make([][]byte, 0)

			for _, k := range keys {
				if bytes.HasPrefix([]byte(k), prefix) {
					values = append(values, allEntries[k])
				}
			}

			return values, nil
		}
	}

	child, err := t.state.GetChild(keyToChild)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, nil
	}
	return child.GetKeysWithPrefix(prefix), nil
}

// LoadCode returns the runtime code (located at :code)
func (t *TrieState) LoadCode() []byte {
	return t.Get(common.CodeKey)
}

// LoadCodeHash returns the hash of the runtime code (located at :code)
func (t *TrieState) LoadCodeHash() (common.Hash, error) {
	code := t.LoadCode()
	return common.Blake2bHash(code)
}

// GetChangedNodeHashes returns the two sets of hashes for all nodes
// inserted and deleted in the state trie since the last block produced (trie snapshot).
func (t *TrieState) GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.state.GetChangedNodeHashes()
}
