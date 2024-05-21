// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/gammazero/deque"
)

type StoredNode interface {
	isStoredNode()
	getNode() Node
}

type (
	New struct {
		node Node
	}
	Cached struct {
		node Node
		hash common.Hash
	}
)

func (New) isStoredNode() {}
func (n New) getNode() Node {
	return n.node
}
func (Cached) isStoredNode() {}
func (n Cached) getNode() Node {
	return n.node
}

func NewNewNode(node Node) New {
	return New{node}
}

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
		return StorageHandle{idx}
	}

	ns.nodes = append(ns.nodes, stored)
	return StorageHandle{len(ns.nodes) - 1}
}

func (ns *NodeStorage) destroy(handle StorageHandle) StoredNode {
	idx := handle.int
	ns.freeIndices.PushBack(idx)
	oldNode := ns.nodes[idx]
	ns.nodes[idx] = nil

	return oldNode
}

func (ns *NodeStorage) get(handle StorageHandle) Node {
	switch n := ns.nodes[handle.int].(type) {
	case New:
		return n.node
	case Cached:
		return n.node
	default:
		panic("unreachable")
	}
}
