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
