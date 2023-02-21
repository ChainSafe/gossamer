// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"bytes"
	"fmt"
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
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
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

// highestLeaf searches the stored leaves to the find the one with the greatest number.
// If there are two leaves with the same number, choose the one with the earliest arrival time.
func (lm *leafMap) highestLeaf() *node {
	lm.RLock()
	defer lm.RUnlock()

	var max uint

	var deepest *node
	lm.smap.Range(func(h, n interface{}) bool {
		node := n.(*node)
		if node == nil {
			// this should never happen
			return true
		}

		if max < node.number {
			max = node.number
			deepest = node
		} else if max == node.number && node.arrivalTime.Before(deepest.arrivalTime) {
			deepest = node
		} else if max == node.number && node.arrivalTime.Equal(deepest.arrivalTime) {
			// there are two leaf nodes with the same number *and* arrival time, just pick the one
			// with the lower hash in lexicographical order.
			// practically, this is very unlikely to happen.
			if bytes.Compare(node.hash[:], deepest.hash[:]) < 0 {
				deepest = node
			}
		}

		return true
	})

	return deepest
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

	var nodes []*node

	lm.smap.Range(func(h, n interface{}) bool {
		node := n.(*node)
		nodes = append(nodes, node)
		return true
	})

	return nodes
}

func (lm *leafMap) bestBlock() *node {
	lm.RLock()
	defer lm.RUnlock()

	// map of primary ancestor count -> *node
	counts := make(map[int][]*node)
	highest := 0

	lm.smap.Range(func(_, nn interface{}) bool {
		n := nn.(*node)
		count := n.primaryAncestorCount(0)
		if count > highest {
			highest = count
		}

		nodesWithCount, has := counts[count]
		if !has {
			counts[count] = []*node{n}
		} else {
			counts[count] = append(nodesWithCount, n)
		}

		return true
	})

	// there's just one node with the highest amount of primary ancestors,
	// so let's return it
	if len(counts[highest]) == 1 {
		return counts[highest][0]
	}

	// there are multple with the highest count, run them through `highestLeaf`
	lm2 := newEmptyLeafMap()
	for _, node := range counts[highest] {
		lm2.store(node.hash, node)
	}

	return lm2.highestLeaf()
}
