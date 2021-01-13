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
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// TrieState is a wrapper around a transient trie that is used during the course of executing some runtime call.
// If the execution of the call is successful, the trie will be saved in the StorageState.
type TrieState struct {
	db   chaindb.Database
	t    *trie.Trie
	lock sync.RWMutex
}

// NewTrieState returns a new TrieState with the given trie
func NewTrieState(t *trie.Trie) (*TrieState, error) {
	r := rand.Intn(1 << 16) //nolint
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(r))

	// TODO: dynamically get os.TMPDIR
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")

	cfg := &chaindb.Config{
		DataDir:  testDatadirPath,
		InMemory: true,
	}

	db, err := chaindb.NewBadgerDB(cfg)
	if err != nil {
		return nil, err
	}

	entries := t.Entries()
	for k, v := range entries {
		err := db.Put([]byte(k), v)
		if err != nil {
			return nil, err
		}
	}

	ts := &TrieState{
		db: db,
		t:  t,
	}
	return ts, nil
}

// NewTestTrieState returns an initialized TrieState
func NewTestTrieState(t *testing.T, tr *trie.Trie) *TrieState {
	if tr == nil {
		tr = trie.NewEmptyTrie()
	}

	// testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")

	// cfg := &chaindb.Config{
	// 	DataDir: testDatadirPath,
	// 	InMemory: true,
	// }

	// db, err := chaindb.NewBadgerDB(cfg)
	// if err != nil {
	// 	t.Fatal("failed to create TestRuntimeStorage database")
	// }

	// return &TrieState{
	// 	db: db,
	// 	t:  tr,
	// }
	ts, err := NewTrieState(tr)
	if err != nil {
		t.Fatal("failed to create TrieState: ", err)
	}

	t.Cleanup(func() {
		_ = ts.db.Close()
		_ = os.RemoveAll(ts.db.Path())
	})

	return ts
}

// Trie returns the TrieState's underlying trie
func (s *TrieState) Trie() *trie.Trie {
	return s.t
}

// Copy performs a deep copy of the TrieState
func (s *TrieState) Copy() (*TrieState, error) {
	trieCopy, err := s.t.DeepCopy()
	if err != nil {
		return nil, err
	}

	return &TrieState{
		db: s.db,
		t:  trieCopy,
	}, nil
}

//nolint
// Commit ensures that the TrieState's trie and database match
// The database is the source of truth due to the runtime interpreter's undefined behaviour regarding the trie
func (s *TrieState) Commit() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.t = trie.NewEmptyTrie()
	iter := s.db.NewIterator()

	for iter.Next() {
		key := iter.Key()
		err := s.t.Put(key, iter.Value())
		if err != nil {
			return err
		}
	}

	iter.Release()
	return nil
}

// WriteTrieToDB writes the trie to the database
func (s *TrieState) WriteTrieToDB() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	for k, v := range s.t.Entries() {
		err := s.db.Put([]byte(k), v)
		if err != nil {
			return err
		}
	}

	return nil
}

// Free should be called once this trie state is no longer needed
func (s *TrieState) Free() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	iter := s.db.NewIterator()

	for iter.Next() {
		err := s.db.Del(iter.Key())
		if err != nil {
			return err
		}
	}

	iter.Release()
	return nil
}

// Set sets a key-value pair in the trie
func (s *TrieState) Set(key []byte, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.db.Put(key, value)
	if err != nil {
		return err
	}
	return s.t.Put(key, value)
}

// Get gets a value from the trie
func (s *TrieState) Get(key []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if has, _ := s.db.Has(key); has {
		return s.db.Get(key)
	}

	return s.t.Get(key)
}

// MustRoot returns the trie's root hash. It panics if it fails to compute the root.
func (s *TrieState) MustRoot() common.Hash {
	root, err := s.Root()
	if err != nil {
		panic(err)
	}

	return root
}

// Root returns the trie's root hash
func (s *TrieState) Root() (common.Hash, error) {
	err := s.Commit()
	if err != nil {
		return common.Hash{}, err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.Hash()
}

// Has returns whether or not a key exists
func (s *TrieState) Has(key []byte) (bool, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.db.Has(key)
}

// Delete deletes a key from the trie
func (s *TrieState) Delete(key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.db.Del(key)
	if err != nil {
		return err
	}

	return s.t.Delete(key)
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

// Entries returns every key-value pair in the trie
func (s *TrieState) Entries() map[string][]byte {
	s.lock.RLock()
	defer s.lock.RUnlock()

	iter := s.db.NewIterator()

	entries := make(map[string][]byte)
	for iter.Next() {
		entries[string(iter.Key())] = iter.Value()
	}

	iter.Release()
	return entries
}

// SetBalance sets the balance for a given public key
func (s *TrieState) SetBalance(key [32]byte, balance uint64) error {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return err
	}

	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, balance)

	return s.Set(skey, bb)
}

// GetBalance returns the balance for a given public key
func (s *TrieState) GetBalance(key [32]byte) (uint64, error) {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return 0, err
	}

	bal, err := s.Get(skey)
	if err != nil {
		return 0, err
	}

	if len(bal) != 8 {
		return 0, nil
	}

	return binary.LittleEndian.Uint64(bal), nil
}

// DeleteChildStorage deletes child storage from the trie
func (s *TrieState) DeleteChildStorage(key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.DeleteFromChild(key)
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

// NextKey returns the next key in the trie in lexicographical order. If it does not exist, it returns nil.
func (s *TrieState) NextKey(key []byte) []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.NextKey(key)
}

// ClearPrefix deletes all key-value pairs from the trie where the key starts with the given prefix
func (s *TrieState) ClearPrefix(prefix []byte) {
	s.lock.Lock()
	s.t.ClearPrefix(prefix)
	s.lock.Unlock()

	iter := s.db.NewIterator()

	for iter.Next() {
		key := iter.Key()
		if len(key) < len(prefix) {
			continue
		}

		if bytes.Equal(key[:len(prefix)], prefix) {
			_ = s.Delete(key)
		}
	}

	iter.Release()
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
