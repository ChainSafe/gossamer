// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/gammazero/deque"
)

// StorageHandle is a pointer to a node contained in `NodeStorage`
type StorageHandle int

// NodeHandle is an enum for the different types of nodes that can be stored in
// in our trieDB before a commit is applied
// This is useful to mantain the trie structure with nodes that could be loaded
// in memory or are a hash to a node that is stored in the backed db
type NodeHandle interface {
	isNodeHandle()
}

type (
	InMemory struct {
		idx StorageHandle
	}
	Persisted struct {
		hash common.Hash
	}
)

func (InMemory) isNodeHandle()  {}
func (Persisted) isNodeHandle() {}

func newInMemoryNodeHandle(idx StorageHandle) NodeHandle {
	return InMemory{idx}
}

func newFromEncodedMerkleValue(
	parentHash common.Hash,
	encodedNodeHandle codec.MerkleValue,
	storage NodeStorage,
) (NodeHandle, error) {
	switch encoded := encodedNodeHandle.(type) {
	case codec.HashedNode:
		return Persisted{hash: common.NewHash(encoded.Data)}, nil
	case codec.InlineNode:
		child, err := newNodeFromEncoded(parentHash, encoded.Data, storage)
		if err != nil {
			return nil, err
		}
		return InMemory{storage.alloc(NewStoredNode{child})}, nil
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

func BuildNewStoredNode(node Node) NewStoredNode {
	return NewStoredNode{node}
}

// NodeStorage is a struct that contains all the temporal nodes that are stored
// in the trieDB before being written to the backed db
type NodeStorage struct {
	nodes       []StoredNode
	freeIndices *deque.Deque[int]
}

func NewNodeStorage() NodeStorage {
	return NodeStorage{
		nodes:       make([]StoredNode, 0),
		freeIndices: deque.New[int](0),
	}
}

func (ns *NodeStorage) alloc(stored StoredNode) StorageHandle {
	if ns.freeIndices.Len() > 0 {
		idx := ns.freeIndices.PopFront()
		ns.nodes[idx] = stored
		return StorageHandle(idx)
	}

	ns.nodes = append(ns.nodes, stored)
	return StorageHandle(len(ns.nodes) - 1)
}

func (ns *NodeStorage) destroy(handle StorageHandle) StoredNode {
	idx := int(handle)
	ns.freeIndices.PushBack(idx)
	oldNode := ns.nodes[idx]
	ns.nodes[idx] = nil

	return oldNode
}

func (ns *NodeStorage) get(handle StorageHandle) Node {
	switch n := ns.nodes[handle].(type) {
	case NewStoredNode:
		return n.node
	case CachedStoredNode:
		return n.node
	default:
		panic("unreachable")
	}
}
