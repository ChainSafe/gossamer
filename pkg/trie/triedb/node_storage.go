// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/gammazero/deque"
)

var EmptyNode = []byte{0}
var hashedNullNode = common.MustBlake2bHash(EmptyNode)

// StorageHandle is a pointer to a node contained in `NodeStorage`
type storageHandle int

// nodeHandle is an enum for the different types of nodes that can be stored in
// in our trieDB before a commit is applied
// This is useful to mantain the trie structure with nodes that could be loaded
// in memory or are a hash to a node that is stored in the backed db
type nodeHandle interface {
	isNodeHandle()
}

type (
	inMemory  storageHandle
	persisted common.Hash
)

func (inMemory) isNodeHandle()  {}
func (persisted) isNodeHandle() {}

func newFromEncodedMerkleValue(
	parentHash common.Hash,
	encodedNodeHandle codec.MerkleValue,
	storage nodeStorage,
) (nodeHandle, error) {
	switch encoded := encodedNodeHandle.(type) {
	case codec.HashedNode:
		return persisted(encoded), nil
	case codec.InlineNode:
		child, err := newNodeFromEncoded(parentHash, encoded, storage)
		if err != nil {
			return nil, err
		}
		return inMemory(storage.alloc(NewStoredNode{child})), nil
	default:
		panic("unreachable")
	}
}

// StoredNode is an enum for temporal nodes stored in the trieDB
// these nodes could be either new nodes or cached nodes
// New nodes are used to know that we need to add them in our backed db
// Cached nodes are loaded in memory and are used to keep the structure of the
// trie
type StoredNode interface {
	getNode() Node
}

type (
	NewStoredNode struct {
		node Node
	}
	CachedStoredNode struct {
		node Node
		hash common.Hash
	}
)

func (n NewStoredNode) getNode() Node {
	return n.node
}
func (n CachedStoredNode) getNode() Node {
	return n.node
}

// nodeStorage is a struct that contains all the temporal nodes that are stored
// in the trieDB before being written to the backed db
type nodeStorage struct {
	nodes       []StoredNode
	freeIndices *deque.Deque[int]
}

func newNodeStorage() nodeStorage {
	return nodeStorage{
		nodes:       make([]StoredNode, 0),
		freeIndices: deque.New[int](0),
	}
}

func (ns *nodeStorage) alloc(stored StoredNode) storageHandle {
	if ns.freeIndices.Len() > 0 {
		idx := ns.freeIndices.PopFront()
		ns.nodes[idx] = stored
		return storageHandle(idx)
	}

	ns.nodes = append(ns.nodes, stored)
	return storageHandle(len(ns.nodes) - 1)
}

func (ns *nodeStorage) destroy(handle storageHandle) StoredNode {
	idx := int(handle)
	ns.freeIndices.PushBack(idx)
	oldNode := ns.nodes[idx]
	ns.nodes[idx] = nil

	return oldNode
}

func (ns *nodeStorage) get(handle storageHandle) Node {
	switch n := ns.nodes[handle].(type) {
	case NewStoredNode:
		return n.node
	case CachedStoredNode:
		return n.node
	default:
		panic("unreachable")
	}
}
