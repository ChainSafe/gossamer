// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
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
	t       *trie.Trie
	oldTrie *trie.Trie // this is the trie before BeginStorageTransaction is called. set to nil if it isn't called
	lock    sync.RWMutex
}

// NewTrieState returns a new TrieState with the given trie
func NewTrieState(t *trie.Trie) *TrieState {
	if t == nil {
		t = trie.NewEmptyTrie()
	}

	return &TrieState{
		t: t,
	}
}

// Trie returns the TrieState's underlying trie
func (s *TrieState) Trie() *trie.Trie {
	return s.t
}

// Snapshot creates a new "version" of the trie. The trie before Snapshot is called
// can no longer be modified, all further changes are on a new "version" of the trie.
// It returns the new version of the trie.
func (s *TrieState) Snapshot() *trie.Trie {
	return s.t.Snapshot()
}

// BeginStorageTransaction begins a new nested storage transaction
// which will either be committed or rolled back at a later time.
func (s *TrieState) BeginStorageTransaction() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.oldTrie = s.t
	s.t = s.t.Snapshot()
}

// CommitStorageTransaction commits all storage changes made since BeginStorageTransaction was called.
func (s *TrieState) CommitStorageTransaction() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.oldTrie = nil
}

// RollbackStorageTransaction rolls back all storage changes made since BeginStorageTransaction was called.
func (s *TrieState) RollbackStorageTransaction() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.t = s.oldTrie
	s.oldTrie = nil
}

// Put puts a key-value pair in the trie
func (s *TrieState) Put(key, value []byte, version trie.Version) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.Put(key, value, version)
}

// Get gets a value from the trie
func (s *TrieState) Get(key []byte) []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.Get(key)
}

// MustRoot returns the trie's root hash. It panics if it fails to compute the root.
func (s *TrieState) MustRoot() common.Hash {
	return s.t.MustHash()
}

// Root returns the trie's root hash
func (s *TrieState) Root() (common.Hash, error) {
	return s.t.Hash()
}

// Has returns whether or not a key exists
func (s *TrieState) Has(key []byte) bool {
	return s.Get(key) != nil
}

// Delete deletes a key from the trie
func (s *TrieState) Delete(key []byte) (err error) {
	val := s.t.Get(key)
	if val == nil {
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	err = s.t.Delete(key)
	if err != nil {
		return fmt.Errorf("deleting from trie: %w", err)
	}

	return nil
}

// NextKey returns the next key in the trie in lexicographical order. If it does not exist, it returns nil.
func (s *TrieState) NextKey(key []byte) []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.NextKey(key)
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (s *TrieState) ClearPrefix(prefix []byte) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.ClearPrefix(prefix)
}

// ClearPrefixLimit deletes key-value pairs from the trie where the key starts with the given prefix till limit reached
func (s *TrieState) ClearPrefixLimit(prefix []byte, limit uint32) (
	deleted uint32, allDeleted bool, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.t.ClearPrefixLimit(prefix, limit)
}

// TrieEntries returns every key-value pair in the trie
func (s *TrieState) TrieEntries() map[string][]byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.Entries()
}

// SetChild sets the child trie at the given key
func (s *TrieState) SetChild(keyToChild []byte, child *trie.Trie) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.SetChild(keyToChild, child)
}

// SetChildStorage sets a key-value pair in a child trie
func (s *TrieState) SetChildStorage(keyToChild, key, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.PutIntoChild(keyToChild, key, value)
}

// GetChild returns the child trie at the given key
func (s *TrieState) GetChild(keyToChild []byte) (*trie.Trie, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.GetChild(keyToChild)
}

// GetChildStorage returns a value from a child trie
func (s *TrieState) GetChildStorage(keyToChild, key []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.GetFromChild(keyToChild, key)
}

// DeleteChild deletes a child trie from the main trie
func (s *TrieState) DeleteChild(key []byte) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.DeleteChild(key)
}

// DeleteChildLimit deletes up to limit of database entries by lexicographic order.
func (s *TrieState) DeleteChildLimit(key []byte, limit *[]byte) (
	deleted uint32, allDeleted bool, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	trieSnapshot := s.t.Snapshot()

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

		s.t = trieSnapshot
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

	s.t = trieSnapshot

	allDeleted = deleted == qtyEntries
	return deleted, allDeleted, nil
}

// ClearChildStorage removes the child storage entry from the trie
func (s *TrieState) ClearChildStorage(keyToChild, key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.ClearFromChild(keyToChild, key)
}

// ClearPrefixInChild clears all the keys from the child trie that have the given prefix
func (s *TrieState) ClearPrefixInChild(keyToChild, prefix []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	child, err := s.t.GetChild(keyToChild)
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

func (s *TrieState) ClearPrefixInChildWithLimit(keyToChild, prefix []byte, limit uint32) (uint32, bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	child, err := s.t.GetChild(keyToChild)
	if err != nil || child == nil {
		return 0, false, err
	}

	return child.ClearPrefixLimit(prefix, limit)
}

// GetChildNextKey returns the next lexicographical larger key from child storage. If it does not exist, it returns nil.
func (s *TrieState) GetChildNextKey(keyToChild, key []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	child, err := s.t.GetChild(keyToChild)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, nil
	}
	return child.NextKey(key), nil
}

// GetKeysWithPrefixFromChild ...
func (s *TrieState) GetKeysWithPrefixFromChild(keyToChild, prefix []byte) ([][]byte, error) {
	child, err := s.GetChild(keyToChild)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, nil
	}
	return child.GetKeysWithPrefix(prefix), nil
}

// LoadCode returns the runtime code (located at :code)
func (s *TrieState) LoadCode() []byte {
	return s.Get(common.CodeKey)
}

// LoadCodeHash returns the hash of the runtime code (located at :code)
func (s *TrieState) LoadCodeHash() (common.Hash, error) {
	code := s.LoadCode()
	return common.Blake2bHash(code)
}

// GetChangedNodeHashes returns the two sets of hashes for all nodes
// inserted and deleted in the state trie since the last block produced (trie snapshot).
func (s *TrieState) GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.GetChangedNodeHashes()
}
