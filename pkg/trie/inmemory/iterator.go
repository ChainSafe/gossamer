// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
)

type IterOpts func(*InMemoryTrieIterator)

var WithTrie = func(tt *InMemoryTrie) IterOpts {
	return func(imti *InMemoryTrieIterator) {
		imti.trie = tt
	}
}

var WithCursorAt = func(cursor []byte) IterOpts {
	return func(imti *InMemoryTrieIterator) {
		imti.cursorAtKey = cursor
	}
}

var _ trie.TrieIterator = (*InMemoryTrieIterator)(nil)

type InMemoryTrieIterator struct {
	trie        *InMemoryTrie
	cursorAtKey []byte
}

func NewInMemoryTrieIterator(opts ...IterOpts) *InMemoryTrieIterator {
	iter := &InMemoryTrieIterator{
		trie:        NewEmptyTrie(),
		cursorAtKey: nil,
	}

	for _, opt := range opts {
		opt(iter)
	}

	return iter
}

func (t *InMemoryTrieIterator) NextEntry() *trie.Entry {
	found := findNextNode(t.trie.root, []byte(nil), t.cursorAtKey)
	if found != nil {
		t.cursorAtKey = found.Key
	}
	return found
}

func (t *InMemoryTrieIterator) NextKey() []byte {
	entry := t.NextEntry()
	if entry != nil {
		return codec.NibblesToKeyLE(entry.Key)
	}
	return nil
}

func (t *InMemoryTrieIterator) NextKeyFunc(predicate func(nextKey []byte) bool) (nextKey []byte) {
	for entry := t.NextEntry(); entry != nil; entry = t.NextEntry() {
		key := codec.NibblesToKeyLE(entry.Key)
		if predicate(key) {
			return key
		}
	}
	return nil
}

func (i *InMemoryTrieIterator) Seek(targetKey []byte) {
	for key := i.NextKey(); bytes.Compare(key, targetKey) < 0; key = i.NextKey() {
	}
}

// Entries returns all the key-value pairs in the trie as a map of keys to values
// where the keys are encoded in Little Endian.
func (t *InMemoryTrie) Entries() (keyValueMap map[string][]byte) {
	keyValueMap = make(map[string][]byte)
	t.buildEntriesMap(t.root, nil, keyValueMap)
	return keyValueMap
}

// NextKey returns the next key in the trie in lexicographic order.
// It returns nil if no next key is found.
func (t *InMemoryTrie) NextKey(keyLE []byte) (nextKeyLE []byte) {
	key := codec.KeyLEToNibbles(keyLE)

	iter := NewInMemoryTrieIterator(WithTrie(t), WithCursorAt(key))
	return iter.NextKey()
}

func findNextNode(currentNode *node.Node, prefix, searchKey []byte) *trie.Entry {
	if currentNode == nil {
		return nil
	}

	currentFullKey := bytes.Join([][]byte{prefix, currentNode.PartialKey}, nil)

	// if the keys are lexicographically equal then we will proceed
	// in order to find the one that is lexicographically greater
	// if the current node is a leaf then there is no other path
	// if the current node is a branch then we can iterate over its children
	switch currentNode.Kind() {
	case node.Leaf:
		// if search key lexicographically lower than the current full key
		// then we should return the full key if it is not in the deletedKeys
		if bytes.Compare(searchKey, currentFullKey) == -1 {
			return &trie.Entry{Key: currentFullKey, Value: currentNode.StorageValue}
		}
	case node.Branch:
		cmp := bytes.Compare(searchKey, currentFullKey)

		// if searchKey is lexicographically lower (-1) and the branch has a storage value then
		// we found the next key, otherwise go over the children from the start
		if cmp == -1 {
			if currentNode.StorageValue != nil {
				return &trie.Entry{Key: currentFullKey, Value: currentNode.StorageValue}
			}

			return findNextKeyOnChildren(
				currentNode,
				currentFullKey,
				searchKey,
				0,
			)
		}

		// if searchKey is lexicographically equal (0) we should go over children from the start
		if cmp == 0 {
			return findNextKeyOnChildren(
				currentNode,
				currentFullKey,
				searchKey,
				0,
			)
		}

		// if searchKey is lexicographically greater (1) we should go over  children starting from
		// the last match between `searchKey` and `currentFullKey`
		if cmp == 1 {
			// search key is exhausted then return nil
			if len(searchKey) <= len(currentFullKey) {
				return nil
			}

			return findNextKeyOnChildren(
				currentNode,
				currentFullKey,
				searchKey,
				searchKey[len(currentFullKey)],
			)
		}
	default:
		panic(fmt.Sprintf("node type not supported: %s", currentNode.Kind().String()))
	}

	return nil
}

func findNextKeyOnChildren(currentNode *node.Node, prefix, searchKey []byte, startingAt byte) *trie.Entry {
	for i := startingAt; i < node.ChildrenCapacity; i++ {
		child := currentNode.Children[i]
		if child == nil {
			continue
		}

		next := findNextNode(child,
			bytes.Join([][]byte{prefix, {i}}, nil),
			searchKey,
		)

		if next != nil {
			return next
		}
	}

	return nil
}
