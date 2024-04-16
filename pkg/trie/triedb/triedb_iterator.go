// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"

	nibbles "github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
)

type iteratorState struct {
	parentFullKey []byte     // key of the parent node of the actual node
	node          codec.Node // actual node
}

// fullKeyNibbles return the full key of the node contained in this state
// child is the child where the node is stored in the parent node
func (s *iteratorState) fullKeyNibbles(child *int) []byte {
	fullKey := bytes.Join([][]byte{s.parentFullKey, s.node.GetPartialKey()}, nil)
	if child != nil {
		return bytes.Join([][]byte{fullKey, {byte(*child)}}, nil)
	}

	return nibbles.NibblesToKeyLE(fullKey)
}

type TrieDBIterator struct {
	db        *TrieDB          // trie to iterate over
	nodeStack []*iteratorState // Pending nodes to visit
}

func NewTrieDBIterator(trie *TrieDB) *TrieDBIterator {
	rootNode, err := trie.getRootNode()
	if err != nil {
		panic("trying to create trie iterator with incomplete trie DB")
	}
	return &TrieDBIterator{
		db: trie,
		nodeStack: []*iteratorState{
			{
				node: rootNode,
			},
		},
	}
}

// nextToVisit sets the next node to visit in the iterator
func (i *TrieDBIterator) nextToVisit(parentKey []byte, node codec.Node) {
	i.nodeStack = append(i.nodeStack, &iteratorState{
		parentFullKey: parentKey,
		node:          node,
	})
}

// nextState pops the next node to visit from the stack
func (i *TrieDBIterator) nextState() *iteratorState {
	currentState := i.nodeStack[len(i.nodeStack)-1]
	i.nodeStack = i.nodeStack[:len(i.nodeStack)-1]
	return currentState
}

func (i *TrieDBIterator) NextEntry() *Entry {
	for len(i.nodeStack) > 0 {
		currentState := i.nextState()
		currentNode := currentState.node

		switch n := currentNode.(type) {
		case codec.Leaf:
			key := currentState.fullKeyNibbles(nil)
			value, err := i.db.loadValue(n.PartialKey, n.GetValue())
			if err != nil {
				panic("Error loading value")
			}
			return &Entry{key: key, value: value}
		case codec.Branch:
			// Reverse iterate over children because we are using a LIFO stack
			// and we want to visit the leftmost child first
			for idx := len(n.Children) - 1; idx >= 0; idx-- {
				child := n.Children[idx]
				if child != nil {
					childNode, err := i.db.getNode(child)
					if err != nil {
						panic(err)
					}
					i.nextToVisit(currentState.fullKeyNibbles(&idx), childNode)
				}
			}
			if n.GetValue() != nil {
				key := currentState.fullKeyNibbles(nil)
				value, err := i.db.loadValue(n.PartialKey, n.GetValue())
				if err != nil {
					panic("Error loading value")
				}
				return &Entry{key: key, value: value}
			}
		}
	}

	return nil
}

// NextKey performs a depth-first search on the trie and returns the next key
// based on the current state of the iterator.
func (i *TrieDBIterator) NextKey() []byte {
	entry := i.NextEntry()
	if entry != nil {
		return entry.key
	}
	return nil
}

// Seek moves the iterator to the first key that is greater than the target key.
func (i *TrieDBIterator) Seek(targetKey []byte) {
	for key := i.NextKey(); bytes.Compare(key, targetKey) < 0; key = i.NextKey() {
	}
}
