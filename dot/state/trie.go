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

package state

import (
	"encoding/binary"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
)

// TrieState is a wrapper around a transient trie that is used during the course of executing some runtime call.
// If the execution of the call is successful, the trie will be saved in the StorageState.
type TrieState struct {
	t    *trie.Trie
	lock sync.RWMutex
}

func NewTrieState(t *trie.Trie) *TrieState {
	return &TrieState{
		t: t,
	}
}

func (s *TrieState) Trie() *trie.Trie {
	return s.t
}

func (s *TrieState) Set(key []byte, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.Put(key, value)
}

func (s *TrieState) Get(key []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.Get(key)
}

func (s *TrieState) Root() (common.Hash, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.Hash()
}

func (s *TrieState) SetChild(keyToChild []byte, child *trie.Trie) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.PutChild(keyToChild, child)
}

func (s *TrieState) SetChildStorage(keyToChild, key, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.PutIntoChild(keyToChild, key, value)
}

func (s *TrieState) GetChild(keyToChild []byte) (*trie.Trie, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.GetChild(keyToChild)
}

func (s *TrieState) GetChildStorage(keyToChild, key []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.GetFromChild(keyToChild, key)
}

func (s *TrieState) Delete(key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.t.Delete(key)
}

func (s *TrieState) Entries() map[string][]byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.t.Entries()
}

func (s *TrieState) SetBalance(key [32]byte, balance uint64) error {
	skey, err := common.BalanceKey(key)
	if err != nil {
		return err
	}

	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, balance)

	return s.Set(skey, bb)
}

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
