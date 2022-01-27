// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"errors"
	"math/big"
	"sync"

	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

// leafMap provides quick lookup for existing leaves
type leafMap struct {
	currHighestLeaf *node
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

// highestLeaf searches the stored leaves to the find the one with the greatest number.
// If there are two leaves with the same number, choose the one with the earliest arrival time.
func (lm *leafMap) highestLeaf() *node {
	lm.RLock()
	defer lm.RUnlock()

	max := big.NewInt(-1)

	var deepest *node
	lm.smap.Range(func(h, n interface{}) bool {
		if n == nil {
			return true
		}

		node := n.(*node)

		if max.Cmp(node.number) < 0 {
			max = node.number
			deepest = node
		} else if max.Cmp(node.number) == 0 && node.arrivalTime.Before(deepest.arrivalTime) {
			deepest = node
		}

		return true
	})

	if lm.currHighestLeaf != nil {
		if lm.currHighestLeaf.hash == deepest.hash {
			return lm.currHighestLeaf
		}

		// update the current deepest leaf if the found deepest has a greater number or
		// if the current and the found deepest has the same number however the current
		// arrived later then the found deepest
		if deepest.number.Cmp(lm.currHighestLeaf.number) == 1 {
			lm.currHighestLeaf = deepest
		} else if deepest.number.Cmp(lm.currHighestLeaf.number) == 0 &&
			deepest.arrivalTime.Before(lm.currHighestLeaf.arrivalTime) {
			lm.currHighestLeaf = deepest
		}
	} else {
		lm.currHighestLeaf = deepest
	}

	return lm.currHighestLeaf
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

func (lm *leafMap) bestBlock() *node {
	lm.RLock()
	defer lm.RUnlock()

	// map of primary ancestor count -> *node
	counts := make(map[int][]*node)

	lm.smap.Range(func(_, nn interface{}) bool {
		n := nn.(*node)
		count := n.primaryAncestorCount(0)

		fmt.Println(n, count)
		nodesWithCount, has := counts[count]
		if !has {
			counts[count] = []*node{n}
		} else {
			counts[count] = append(nodesWithCount, n)
		}

		return true
	})

	// get highest count
	highest := 0
	for count := range counts {
		if count > highest {
			highest = count
		}
	}

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
