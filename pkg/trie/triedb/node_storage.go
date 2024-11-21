// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/gammazero/deque"
)

var EmptyNode = []byte{0}

// StorageHandle is a pointer to a node contained in `NodeStorage`
type storageHandle int

// NodeHandle is an enum for the different types of nodes that can be stored in
// in our trieDB before a commit is applied
// This is useful to mantain the trie structure with nodes that could be loaded
// in memory or are a hash to a node that is stored in the backed db
type NodeHandle interface {
	isNodeHandle()
}

type (
	inMemory               storageHandle
	persisted[H hash.Hash] struct{ hash H }
)

func (inMemory) isNodeHandle()     {}
func (persisted[H]) isNodeHandle() {}

func newNodeHandleFromMerkleValue[H hash.Hash](
	parentHash H,
	encodedNodeHandle codec.MerkleValue,
	storage *nodeStorage[H],
) (NodeHandle, error) {
	switch encoded := encodedNodeHandle.(type) {
	case codec.HashedNode[H]:
		return persisted[H]{encoded.Hash}, nil
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

func newNodeHandleFromCachedNodeHandle[H hash.Hash](
	child CachedNodeHandle,
	storage *nodeStorage[H],
) NodeHandle {
	switch child := child.(type) {
	case HashCachedNodeHandle[H]:
		return persisted[H]{child.Hash}
	case InlineCachedNodeHandle[H]:
		ch := newNodeFromCachedNode(child.CachedNode, storage)
		return inMemory(storage.alloc(NewStoredNode{node: ch}))
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
	CachedStoredNode[H hash.Hash] struct {
		node Node
		hash H
	}
)

func (n NewStoredNode) getNode() Node {
	return n.node
}
func (n CachedStoredNode[H]) getNode() Node {
	return n.node
}

// nodeStorage is a struct that contains all the temporal nodes that are stored
// in the trieDB before being written to the backed db
type nodeStorage[H hash.Hash] struct {
	nodes       []StoredNode
	freeIndices *deque.Deque[int]
}

func newNodeStorage[H hash.Hash]() nodeStorage[H] {
	return nodeStorage[H]{
		nodes:       make([]StoredNode, 0),
		freeIndices: new(deque.Deque[int]),
	}
}

func (ns *nodeStorage[H]) alloc(stored StoredNode) storageHandle {
	if ns.freeIndices.Len() > 0 {
		idx := ns.freeIndices.PopFront()
		ns.nodes[idx] = stored
		return storageHandle(idx)
	}

	ns.nodes = append(ns.nodes, stored)
	return storageHandle(len(ns.nodes) - 1)
}

func (ns *nodeStorage[H]) destroy(handle storageHandle) StoredNode {
	idx := int(handle)
	ns.freeIndices.PushBack(idx)
	oldNode := ns.nodes[idx]
	ns.nodes[idx] = nil

	return oldNode
}

func (ns *nodeStorage[H]) get(handle storageHandle) Node {
	switch n := ns.nodes[handle].(type) {
	case NewStoredNode:
		return n.node
	case CachedStoredNode[H]:
		return n.node
	default:
		panic("unreachable")
	}
}
