// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

// ChildStorageKeyPrefix is the prefix for all child storage keys
var ChildStorageKeyPrefix = []byte(":child_storage:default:")

var ErrChildTrieDoesNotExist = errors.New("child trie does not exist")

// PutChild inserts a child trie into the main trie at key :child_storage:[keyToChild]
// A child trie is added as a node (K, V) in the main trie. K is the child storage key
// associated to the child trie, and V is the root hash of the child trie.
func (t *Trie) PutChild(keyToChild []byte, child *Trie) error {
	childHash, err := child.Hash()
	if err != nil {
		return err
	}

	key := make([]byte, len(ChildStorageKeyPrefix)+len(keyToChild))
	copy(key, ChildStorageKeyPrefix)
	copy(key[len(ChildStorageKeyPrefix):], keyToChild)

	fmt.Println("HERE", string(key))
	t.Put(key, childHash.ToBytes())
	t.childTries[childHash] = child
	return nil
}

// GetChild returns the child trie at key :child_storage:[keyToChild]
func (t *Trie) GetChild(keyToChild []byte) (*Trie, error) {
	key := make([]byte, len(ChildStorageKeyPrefix)+len(keyToChild))
	copy(key, ChildStorageKeyPrefix)
	copy(key[len(ChildStorageKeyPrefix):], keyToChild)

	childHash := t.Get(key)
	if childHash == nil {
		return nil, fmt.Errorf("%w at key 0x%x%x", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}

	return t.childTries[common.BytesToHash(childHash)], nil
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
		return nil, fmt.Errorf("%w at key 0x%x%x", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}

	val := child.Get(key)
	return val, nil
}

// DeleteChild deletes the child storage trie
func (t *Trie) DeleteChild(keyToChild []byte) {
	key := make([]byte, len(ChildStorageKeyPrefix)+len(keyToChild))
	copy(key, ChildStorageKeyPrefix)
	copy(key[len(ChildStorageKeyPrefix):], keyToChild)

	t.Delete(key)
}

// ClearFromChild removes the child storage entry
func (t *Trie) ClearFromChild(keyToChild, key []byte) error {
	child, err := t.GetChild(keyToChild)
	if err != nil {
		return err
	}
	if child == nil {
		return fmt.Errorf("%w at key 0x%x%x", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}
	child.Delete(key)
	return nil
}
