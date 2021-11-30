// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
)

// leafMap provides quick lookup for existing leaves
type leafMap struct {
	sync.RWMutex
	smap *sync.Map // map[common.Hash]*node
}

func newEmptyLeafMap() *leafMap {
	return &leafMap{
		smap: &sync.Map{},
	}
}

func newLeafMap(n *node) *leafMap {
	smap := &sync.Map{}
	for _, leaf := range n.getLeaves(nil) {
		smap.Store(leaf.hash, leaf)
	}

	return &leafMap{
		smap: smap,
	}
}

func (lm *leafMap) store(key Hash, value *node) {
	lm.smap.Store(key, value)
}

func (lm *leafMap) load(key Hash) (*node, error) {
	v, ok := lm.smap.Load(key)
	if !ok {
		return nil, errors.New("key not found")
	}

	return v.(*node), nil
}

// Replace deletes the old node from the map and inserts the new one
func (lm *leafMap) replace(oldNode, newNode *node) {
	lm.Lock()
	defer lm.Unlock()
	lm.smap.Delete(oldNode.hash)
	lm.store(newNode.hash, newNode)
}

// DeepestLeaf searches the stored leaves to the find the one with the greatest number.
// If there are two leaves with the same number, choose the one with the earliest arrival time.
func (lm *leafMap) deepestLeaf() *node {
	lm.RLock()
	defer lm.RUnlock()

	max := big.NewInt(-1)

	var dLeaf *node
	lm.smap.Range(func(h, n interface{}) bool {
		if n == nil {
			return true
		}

		node := n.(*node)

		if max.Cmp(node.number) < 0 {
			max = node.number
			dLeaf = node
		} else if max.Cmp(node.number) == 0 && node.arrivalTime.Before(dLeaf.arrivalTime) {
			dLeaf = node
		}

		return true
	})

	return dLeaf
}

func (lm *leafMap) toMap() map[common.Hash]*node {
	lm.RLock()
	defer lm.RUnlock()

	mmap := make(map[common.Hash]*node)

	lm.smap.Range(func(h, n interface{}) bool {
		hash := h.(Hash)
		node := n.(*node)
		mmap[hash] = node
		return true
	})

	return mmap
}

func (lm *leafMap) nodes() []*node {
	lm.RLock()
	defer lm.RUnlock()

	nodes := []*node{}

	lm.smap.Range(func(h, n interface{}) bool {
		node := n.(*node)
		nodes = append(nodes, node)
		return true
	})

	return nodes
}
