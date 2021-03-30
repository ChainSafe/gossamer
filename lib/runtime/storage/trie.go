// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package storage

import (
	"encoding/binary"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// TrieState is a wrapper around a transient trie that is used during the course of executing some runtime call.
// If the execution of the call is successful, the trie will be saved in the StorageState.
type TrieState struct {
	t       *trie.Trie
	oldTrie *trie.Trie // this is the trie before BeginStorageTransaction is called. set to nil if it isn't called
	lock    sync.RWMutex
}

// NewTrieState returns a new TrieState with the given trie
func NewTrieState(t *trie.Trie) (*TrieState, error) {
	if t == nil {
		t = trie.NewEmptyTrie()
	}

	ts := &TrieState{
		t: t,
	}

	return ts, nil
}

// Trie returns the TrieState's underlying trie
func (s *TrieState) Trie() *trie.Trie {
	return s.t
}

// Snapshot creates a new "version" of the trie. The trie before Snapshot is called
// can no longer be modified, all further changes are on a new "version" of the trie.
// It returns the previous version of the trie.
func (s *TrieState) Snapshot() *trie.Trie {
	return s.t.Snapshot()
}

// BeginStorageTransaction begins a new nested storage transaction which will either be committed or rolled back at a later time.
func (s *TrieState) BeginStorageTransaction() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.oldTrie = s.t.Snapshot()
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

// Set sets a key-value pair in the trie
func (s *TrieState) Set(key, value []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.t.Put(key, value)
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
func (s *TrieState) Delete(key []byte) {
	val := s.t.Get(key)
	if val == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.t.Delete(key)
}

// NextKey returns the next key in the trie in lexicographical order. If it does not exist, it returns nil.
func (s *TrieState) NextKey(key []byte) []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.NextKey(key)
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (s *TrieState) ClearPrefix(prefix []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.t.ClearPrefix(prefix)
	return nil
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
	return s.t.PutChild(keyToChild, child)
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
func (s *TrieState) DeleteChild(key []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.t.DeleteChild(key)
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

	child.ClearPrefix(prefix)
	return nil
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

// TODO: remove functions below

// SetBalance sets the balance for a given public key
func (s *TrieState) SetBalance(key [32]byte, balance uint64) error {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return err
	}

	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, balance)

	s.Set(skey, bb)
	return nil
}

// GetBalance returns the balance for a given public key
func (s *TrieState) GetBalance(key [32]byte) (uint64, error) {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return 0, err
	}

	bal := s.Get(skey)
	if len(bal) != 8 {
		return 0, nil
	}

	return binary.LittleEndian.Uint64(bal), nil
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
