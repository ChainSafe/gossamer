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

// SetChild inserts a child trie into the main trie at key :child_storage:[keyToChild]
// A child trie is added as a node (K, V) in the main trie. K is the child storage key
// associated to the child trie, and V is the root hash of the child trie.
func (t *Trie) SetChild(keyToChild []byte, child *Trie, version Version) error {
	childHash, err := child.Hash()
	if err != nil {
		return err
	}

	key := make([]byte, len(ChildStorageKeyPrefix)+len(keyToChild))
	copy(key, ChildStorageKeyPrefix)
	copy(key[len(ChildStorageKeyPrefix):], keyToChild)

	err = t.Put(key, childHash.ToBytes(), version)
	if err != nil {
		return fmt.Errorf("putting child trie root hash %s in trie: %w", childHash, err)
	}

	t.childTries[childHash] = child
	return nil
}

func (t *Trie) MustGetChild(keyToChild []byte) *Trie {
	t, err := t.GetChild(keyToChild)
	if err != nil {
		panic(err)
	}

	return t
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
func (t *Trie) PutIntoChild(keyToChild, key, value []byte, version Version) error {
	child, err := t.GetChild(keyToChild)
	if err != nil {
		if errors.Is(err, ErrChildTrieDoesNotExist) {
			child = NewEmptyTrie()
		} else {
			return fmt.Errorf("getting child: %w", err)
		}
	}

	origChildHash, err := child.Hash()
	if err != nil {
		return err
	}

	err = child.Put(key, value, version)
	if err != nil {
		return fmt.Errorf("putting into child trie located at key 0x%x: %w", keyToChild, err)
	}

	delete(t.childTries, origChildHash)
	return t.SetChild(keyToChild, child, version)
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
func (t *Trie) DeleteChild(keyToChild []byte) (err error) {
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
func (t *Trie) ClearFromChild(keyToChild, key []byte, version Version) error {
	child, err := t.GetChild(keyToChild)
	if err != nil {
		return err
	}

	if child == nil {
		return fmt.Errorf("%w at key 0x%x%x", ErrChildTrieDoesNotExist, ChildStorageKeyPrefix, keyToChild)
	}

	originalHash, err := child.Hash()
	if err != nil {
		return err
	}

	err = child.Delete(key)
	if err != nil {
		return fmt.Errorf("deleting from child trie located at key 0x%x: %w", keyToChild, err)
	}

	delete(t.childTries, originalHash)
	if child.root == nil {
		return t.DeleteChild(keyToChild)
	}

	return t.SetChild(keyToChild, child, version)
}
