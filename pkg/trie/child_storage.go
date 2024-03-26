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

// setChild inserts a child trie into the main trie at key :child_storage:[keyToChild]
// A child trie is added as a node (K, V) in the main trie. K is the child storage key
// associated to the child trie, and V is the root hash of the child trie.
func (t *InMemoryTrie) SetChild(keyToChild []byte, child *InMemoryTrie) error {
	childHash, err := child.Hash()
	if err != nil {
		return err
	}

	key := make([]byte, len(ChildStorageKeyPrefix)+len(keyToChild))
	copy(key, ChildStorageKeyPrefix)
	copy(key[len(ChildStorageKeyPrefix):], keyToChild)

	err = t.Put(key, childHash.ToBytes())
	if err != nil {
		return fmt.Errorf("putting child trie root hash %s in trie: %w", childHash, err)
	}

	t.childTries[childHash] = child
	return nil
}

func (t *InMemoryTrie) getInternalChildTrie(keyToChild []byte) (*InMemoryTrie, error) {
	key := make([]byte, len(ChildStorageKeyPrefix)+len(keyToChild))
	copy(key, ChildStorageKeyPrefix)
	copy(key[len(ChildStorageKeyPrefix):], keyToChild)

	childHash := t.Get(key)
	if childHash == nil {
		return nil, fmt.Errorf("%w at key 0x%x%x", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}

	return t.childTries[common.BytesToHash(childHash)], nil
}

// GetChild returns the child trie at key :child_storage:[keyToChild]
func (t *InMemoryTrie) GetChild(keyToChild []byte) (Trie, error) {
	child, err := t.getInternalChildTrie(keyToChild)
	if child == nil {
		return nil, err
	}
	return child, err
}

// GetChildTries returns all child tries in this trie
func (t *InMemoryTrie) GetChildTries() map[common.Hash]Trie {
	children := make(map[common.Hash]Trie)
	for k, v := range t.childTries {
		children[k] = v
	}
	return children
}

// PutIntoChild puts a key-value pair into the child trie located in the main trie at key :child_storage:[keyToChild]
func (t *InMemoryTrie) PutIntoChild(keyToChild, key, value []byte) error {
	child, err := t.getInternalChildTrie(keyToChild)
	if err != nil {
		if errors.Is(err, ErrChildTrieDoesNotExist) {
			child = NewEmptyInmemoryTrie()
		} else {
			return fmt.Errorf("getting child: %w", err)
		}
	}
	child.version = t.version

	origChildHash, err := child.Hash()
	if err != nil {
		return err
	}

	err = child.Put(key, value)
	if err != nil {
		return fmt.Errorf("putting into child trie located at key 0x%x: %w", keyToChild, err)
	}

	delete(t.childTries, origChildHash)
	return t.SetChild(keyToChild, child)
}

// GetFromChild retrieves a key-value pair from the child trie located
// in the main trie at key :child_storage:[keyToChild]
func (t *InMemoryTrie) GetFromChild(keyToChild, key []byte) ([]byte, error) {
	child, err := t.GetChild(keyToChild)
	if err != nil {
		return nil, err
	}

	val := child.Get(key)
	return val, nil
}

// DeleteChild deletes the child storage trie
func (t *InMemoryTrie) DeleteChild(keyToChild []byte) (err error) {
	key := make([]byte, len(ChildStorageKeyPrefix)+len(keyToChild))
	copy(key, ChildStorageKeyPrefix)
	copy(key[len(ChildStorageKeyPrefix):], keyToChild)

	err = t.Delete(key)
	if err != nil {
		return fmt.Errorf("deleting child trie located at key 0x%x: %w", keyToChild, err)
	}
	return nil
}

// ClearFromChild removes the child storage entry
func (t *InMemoryTrie) ClearFromChild(keyToChild, key []byte) error {
	child, err := t.getInternalChildTrie(keyToChild)
	if err != nil {
		return err
	}

	if child == nil {
		return fmt.Errorf("%w at key 0x%x%x", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}

	origChildHash, err := child.Hash()
	if err != nil {
		return err
	}

	err = child.Delete(key)
	if err != nil {
		return fmt.Errorf("deleting from child trie located at key 0x%x: %w", keyToChild, err)
	}

	delete(t.childTries, origChildHash)
	if child.root == nil {
		return t.DeleteChild(keyToChild)
	}

	return t.SetChild(keyToChild, child)
}
