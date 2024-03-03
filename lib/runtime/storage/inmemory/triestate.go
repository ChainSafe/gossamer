// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"golang.org/x/exp/maps"
)

// InMemoryTrieState is a wrapper around a transient trie that is used during the course of executing some runtime call.
// If the execution of the call is successful, the trie will be saved in the StorageState.
type InMemoryTrieState struct {
	mtx          sync.RWMutex
	transactions *list.List
}

func NewTrieState(state *trie.InMemoryTrie) *InMemoryTrieState {
	transactions := list.New()
	transactions.PushBack(state)
	return &InMemoryTrieState{
		transactions: transactions,
	}
}

func (t *InMemoryTrieState) getCurrentTrie() *trie.InMemoryTrie {
	return t.transactions.Back().Value.(*trie.InMemoryTrie)
}

func (t *InMemoryTrieState) updateCurrentTrie(new *trie.InMemoryTrie) {
	t.transactions.Back().Value = new
}

// StartTransaction begins a new nested storage transaction
// which will either be committed or rolled back at a later time.
func (t *InMemoryTrieState) StartTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	t.transactions.PushBack(t.getCurrentTrie().Snapshot())
}

// Rollback rolls back all storage changes made since StartTransaction was called.
func (t *InMemoryTrieState) RollbackTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.transactions.Len() <= 1 {
		panic("no transactions to rollback")
	}

	t.transactions.Remove(t.transactions.Back())
}

// Commit commits all storage changes made since StartTransaction was called.
func (t *InMemoryTrieState) CommitTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.transactions.Len() <= 1 {
		panic("no transactions to commit")
	}

	t.transactions.Back().Prev().Value = t.transactions.Remove(t.transactions.Back())
}

func (t *InMemoryTrieState) SetVersion(v trie.TrieLayout) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.getCurrentTrie().SetVersion(v)
}

// Trie returns the TrieState's underlying trie
func (t *InMemoryTrieState) Trie() *trie.InMemoryTrie {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie()
}

// Snapshot creates a new "version" of the trie. The trie before Snapshot is called
// can no longer be modified, all further changes are on a new "version" of the trie.
// It returns the new version of the trie.
func (t *InMemoryTrieState) Snapshot() *trie.InMemoryTrie {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().Snapshot()
}

// Put puts a key-value pair in the trie
func (t *InMemoryTrieState) Put(key, value []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return t.getCurrentTrie().Put(key, value)
}

// Get gets a value from the trie
func (t *InMemoryTrieState) Get(key []byte) []byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.getCurrentTrie().Get(key)
}

// MustRoot returns the trie's root hash. It panics if it fails to compute the root.
func (t *InMemoryTrieState) MustRoot() common.Hash {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().MustHash()
}

// Root returns the trie's root hash
func (t *InMemoryTrieState) Root() (common.Hash, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().Hash()
}

// Has returns whether or not a key exists
func (t *InMemoryTrieState) Has(key []byte) bool {
	return t.Get(key) != nil
}

// Delete deletes a key from the trie
func (t *InMemoryTrieState) Delete(key []byte) (err error) {
	val := t.getCurrentTrie().Get(key)
	if val == nil {
		return nil
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()

	err = t.getCurrentTrie().Delete(key)
	if err != nil {
		return fmt.Errorf("deleting from trie: %w", err)
	}

	return nil
}

// NextKey returns the next key in the trie in lexicographical order. If it does not exist, it returns nil.
func (t *InMemoryTrieState) NextKey(key []byte) []byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.getCurrentTrie().NextKey(key)
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (t *InMemoryTrieState) ClearPrefix(prefix []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.getCurrentTrie().ClearPrefix(prefix)
}

// ClearPrefixLimit deletes key-value pairs from the trie where the key starts with the given prefix till limit reached
func (t *InMemoryTrieState) ClearPrefixLimit(prefix []byte, limit uint32) (
	deleted uint32, allDeleted bool, err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return t.getCurrentTrie().ClearPrefixLimit(prefix, limit)
}

// TrieEntries returns every key-value pair in the trie
func (t *InMemoryTrieState) TrieEntries() map[string][]byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.getCurrentTrie().Entries()
}

// SetChild sets the child trie at the given key
func (t *InMemoryTrieState) SetChild(keyToChild []byte, child trie.Trie) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.getCurrentTrie().SetChild(keyToChild, child.(*trie.InMemoryTrie))
}

// SetChildStorage sets a key-value pair in a child trie
func (t *InMemoryTrieState) SetChildStorage(keyToChild, key, value []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return t.getCurrentTrie().PutIntoChild(keyToChild, key, value)
}

// GetChild returns the child trie at the given key
func (t *InMemoryTrieState) GetChild(keyToChild []byte) (trie.Trie, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().GetChild(keyToChild)
}

// GetChildStorage returns a value from a child trie
func (t *InMemoryTrieState) GetChildStorage(keyToChild, key []byte) ([]byte, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().GetFromChild(keyToChild, key)
}

// DeleteChild deletes a child trie from the main trie
func (t *InMemoryTrieState) DeleteChild(key []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return t.getCurrentTrie().DeleteChild(key)
}

// DeleteChildLimit deletes up to limit of database entries by lexicographic order.
func (t *InMemoryTrieState) DeleteChildLimit(key []byte, limit *[]byte) (
	deleted uint32, allDeleted bool, err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	trieSnapshot := t.getCurrentTrie().Snapshot()

	tr, err := trieSnapshot.GetChild(key)
	if err != nil {
		return 0, false, err
	}

	childTrieEntries := tr.Entries()
	qtyEntries := uint32(len(childTrieEntries))
	if limit == nil {
		err = trieSnapshot.DeleteChild(key)
		if err != nil {
			return 0, false, fmt.Errorf("deleting child trie: %w", err)
		}

		t.updateCurrentTrie(trieSnapshot)
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
		err = tr.Delete([]byte(k))
		if err != nil {
			return deleted, allDeleted, fmt.Errorf("deleting from child trie located at key 0x%x: %w", key, err)
		}

		deleted++
		if deleted == limitUint {
			break
		}
	}
	t.updateCurrentTrie(trieSnapshot)
	allDeleted = deleted == qtyEntries
	return deleted, allDeleted, nil
}

// ClearChildStorage removes the child storage entry from the trie
func (t *InMemoryTrieState) ClearChildStorage(keyToChild, key []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.getCurrentTrie().ClearFromChild(keyToChild, key)
}

// ClearPrefixInChild clears all the keys from the child trie that have the given prefix
func (t *InMemoryTrieState) ClearPrefixInChild(keyToChild, prefix []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	child, err := t.getCurrentTrie().GetChild(keyToChild)
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

func (t *InMemoryTrieState) ClearPrefixInChildWithLimit(keyToChild, prefix []byte, limit uint32) (uint32, bool, error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	child, err := t.getCurrentTrie().GetChild(keyToChild)
	if err != nil || child == nil {
		return 0, false, err
	}

	return child.ClearPrefixLimit(prefix, limit)
}

// GetChildNextKey returns the next lexicographical larger key from child storage. If it does not exist, it returns nil.
func (t *InMemoryTrieState) GetChildNextKey(keyToChild, key []byte) ([]byte, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	child, err := t.getCurrentTrie().GetChild(keyToChild)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, nil
	}
	return child.NextKey(key), nil
}

// GetKeysWithPrefixFromChild ...
func (t *InMemoryTrieState) GetKeysWithPrefixFromChild(keyToChild, prefix []byte) ([][]byte, error) {
	child, err := t.GetChild(keyToChild)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, nil
	}
	return child.GetKeysWithPrefix(prefix), nil
}

// LoadCode returns the runtime code (located at :code)
func (t *InMemoryTrieState) LoadCode() []byte {
	return t.Get(common.CodeKey)
}

// LoadCodeHash returns the hash of the runtime code (located at :code)
func (t *InMemoryTrieState) LoadCodeHash() (common.Hash, error) {
	code := t.LoadCode()
	return common.Blake2bHash(code)
}

// GetChangedNodeHashes returns the two sets of hashes for all nodes
// inserted and deleted in the state trie since the last block produced (trie snapshot).
func (t *InMemoryTrieState) GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.getCurrentTrie().GetChangedNodeHashes()
}
