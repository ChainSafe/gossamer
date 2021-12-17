// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

// ChildStorageKeyPrefix is the prefix for all child storage keys
var ChildStorageKeyPrefix = []byte(":child_storage:default:")

const ErrChildTrieDoesNotExist = "child trie does not exist at key"

// PutChild inserts a child trie into the main trie at key :child_storage:[keyToChild]
func (t *Trie) PutChild(keyToChild []byte, child *Trie) error {
	childHash, err := child.Hash()
	if err != nil {
		return err
	}

	key := append(ChildStorageKeyPrefix, keyToChild...)
	value := [32]byte(childHash)

	t.Put(key, value[:])
	t.childTries[childHash] = child
	return nil
}

// GetChild returns the child trie at key :child_storage:[keyToChild]
func (t *Trie) GetChild(keyToChild []byte) (*Trie, error) {
	key := append(ChildStorageKeyPrefix, keyToChild...)
	childHash := t.Get(key)
	if childHash == nil {
		return nil, fmt.Errorf("%s %s%s", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}

	hash := [32]byte{}
	copy(hash[:], childHash)
	return t.childTries[common.Hash(hash)], nil
}

// PutIntoChild puts a key-value pair into the child trie located in the main trie at key :child_storage:[keyToChild]
func (t *Trie) PutIntoChild(keyToChild, key, value []byte) error {
	child, err := t.GetChild(keyToChild)
	if err != nil {
		return err
	}

	origChildHash, err := child.Hash()
	if err != nil {
		return err
	}

	child.Put(key, value)
	childHash, err := child.Hash()
	if err != nil {
		return err
	}

	delete(t.childTries, origChildHash)
	t.childTries[childHash] = child

	return t.PutChild(keyToChild, child)
}

// GetFromChild retrieves a key-value pair from the child trie located
// in the main trie at key :child_storage:[keyToChild]
func (t *Trie) GetFromChild(keyToChild, key []byte) ([]byte, error) {
	child, err := t.GetChild(keyToChild)
	if err != nil {
		return nil, err
	}

	if child == nil {
		return nil, fmt.Errorf("%s %s%s", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}

	val := child.Get(key)
	return val, nil
}

// DeleteChild deletes the child storage trie
func (t *Trie) DeleteChild(keyToChild []byte) {
	key := append(ChildStorageKeyPrefix, keyToChild...)
	t.Delete(key)
}

// ClearFromChild removes the child storage entry
func (t *Trie) ClearFromChild(keyToChild, key []byte) error {
	child, err := t.GetChild(keyToChild)
	if err != nil {
		return err
	}
	if child == nil {
		return fmt.Errorf("%s %s%s", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}
	child.Delete(key)
	return nil
}
