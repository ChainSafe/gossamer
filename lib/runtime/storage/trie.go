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
	"github.com/ChainSafe/gossamer/lib/trie"
	"golang.org/x/exp/maps"
)

// TrieState is a wrapper around a transient trie that is used during the course of executing some runtime call.
// If the execution of the call is successful, the trie will be saved in the StorageState.
type TrieState struct {
	mtx          sync.RWMutex
	transactions *list.List
}

func NewTrieState(state *trie.Trie) *TrieState {
	transactions := list.New()
	transactions.PushBack(state)
	return &TrieState{
		transactions: transactions,
	}
}

func (t *TrieState) getCurrentTrie() *trie.Trie {
	return t.transactions.Back().Value.(*trie.Trie)
}

func (t *TrieState) updateCurrentTrie(new *trie.Trie) {
	t.transactions.Back().Value = new
}

// StartTransaction begins a new nested storage transaction
// which will either be committed or rolled back at a later time.
func (t *TrieState) StartTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	fmt.Printf("STARTING TRANSACTION\n")
	t.transactions.PushBack(t.getCurrentTrie().Snapshot())
}

// Rollback rolls back all storage changes made since StartTransaction was called.
func (t *TrieState) RollbackTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.transactions.Len() <= 1 {
		panic("no transactions to rollback")
	}
	fmt.Printf("ROLLBACK TRANSACTION\n")

	t.transactions.Remove(t.transactions.Back())
}

// Commit commits all storage changes made since StartTransaction was called.
func (t *TrieState) CommitTransaction() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.transactions.Len() <= 1 {
		panic("no transactions to commit")
	}

	fmt.Printf("COMMITING TRANSACTION\n")
	t.transactions.Back().Prev().Value = t.transactions.Remove(t.transactions.Back())
}

func (t *TrieState) SetVersion(v trie.TrieLayout) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.getCurrentTrie().SetVersion(v)
}

// Trie returns the TrieState's underlying trie
func (t *TrieState) Trie() *trie.Trie {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie()
}

// Snapshot creates a new "version" of the trie. The trie before Snapshot is called
// can no longer be modified, all further changes are on a new "version" of the trie.
// It returns the new version of the trie.
func (t *TrieState) Snapshot() *trie.Trie {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().Snapshot()
}

// Put puts a key-value pair in the trie
func (t *TrieState) Put(key, value []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	fmt.Printf("inserting value\nkey: 0x%x\nvalue: 0x%x\n", key, value)
	return t.getCurrentTrie().Put(key, value)
}

// Get gets a value from the trie
func (t *TrieState) Get(key []byte) []byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.getCurrentTrie().Get(key)
}

// MustRoot returns the trie's root hash. It panics if it fails to compute the root.
func (t *TrieState) MustRoot() common.Hash {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().MustHash()
}

// Root returns the trie's root hash
func (t *TrieState) Root() (common.Hash, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().Hash()
}

// Has returns whether or not a key exists
func (t *TrieState) Has(key []byte) bool {
	return t.Get(key) != nil
}

// Delete deletes a key from the trie
func (t *TrieState) Delete(key []byte) (err error) {
	val := t.getCurrentTrie().Get(key)
	if val == nil {
		return nil
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()

	fmt.Printf("deleting key: 0x%x\n", key)

	err = t.getCurrentTrie().Delete(key)
	if err != nil {
		return fmt.Errorf("deleting from trie: %w", err)
	}

	return nil
}

// NextKey returns the next key in the trie in lexicographical order. If it does not exist, it returns nil.
func (t *TrieState) NextKey(key []byte) []byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.getCurrentTrie().NextKey(key)
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (t *TrieState) ClearPrefix(prefix []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.getCurrentTrie().ClearPrefix(prefix)
}

// ClearPrefixLimit deletes key-value pairs from the trie where the key starts with the given prefix till limit reached
func (t *TrieState) ClearPrefixLimit(prefix []byte, limit uint32) (
	deleted uint32, allDeleted bool, err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return t.getCurrentTrie().ClearPrefixLimit(prefix, limit)
}

// TrieEntries returns every key-value pair in the trie
func (t *TrieState) TrieEntries() map[string][]byte {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.getCurrentTrie().Entries()
}

// SetChild sets the child trie at the given key
func (t *TrieState) SetChild(keyToChild []byte, child *trie.Trie) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.getCurrentTrie().SetChild(keyToChild, child)
}

// SetChildStorage sets a key-value pair in a child trie
func (t *TrieState) SetChildStorage(keyToChild, key, value []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	fmt.Printf("setting child storage!")
	return t.getCurrentTrie().PutIntoChild(keyToChild, key, value)
}

// GetChild returns the child trie at the given key
func (t *TrieState) GetChild(keyToChild []byte) (*trie.Trie, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.getCurrentTrie().GetChild(keyToChild)
}

// GetChildStorage returns a value from a child trie
func (t *TrieState) GetChildStorage(keyToChild, key []byte) ([]byte, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	fmt.Printf("getting child storage!")
	return t.getCurrentTrie().GetFromChild(keyToChild, key)
}

// DeleteChild deletes a child trie from the main trie
func (t *TrieState) DeleteChild(key []byte) (err error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return t.getCurrentTrie().DeleteChild(key)
}

// DeleteChildLimit deletes up to limit of database entries by lexicographic order.
func (t *TrieState) DeleteChildLimit(key []byte, limit *[]byte) (
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
func (t *TrieState) ClearChildStorage(keyToChild, key []byte) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.getCurrentTrie().ClearFromChild(keyToChild, key)
}

// ClearPrefixInChild clears all the keys from the child trie that have the given prefix
func (t *TrieState) ClearPrefixInChild(keyToChild, prefix []byte) error {
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

func (t *TrieState) ClearPrefixInChildWithLimit(keyToChild, prefix []byte, limit uint32) (uint32, bool, error) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	child, err := t.getCurrentTrie().GetChild(keyToChild)
	if err != nil || child == nil {
		return 0, false, err
	}

	return child.ClearPrefixLimit(prefix, limit)
}

// GetChildNextKey returns the next lexicographical larger key from child storage. If it does not exist, it returns nil.
func (t *TrieState) GetChildNextKey(keyToChild, key []byte) ([]byte, error) {
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
func (t *TrieState) GetKeysWithPrefixFromChild(keyToChild, prefix []byte) ([][]byte, error) {
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
	return t.getCurrentTrie().GetChangedNodeHashes()
}
