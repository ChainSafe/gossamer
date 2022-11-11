// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/disiqueira/gotree"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	leavesGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_block",
		Name:      "leaves_total",
		Help:      "total number of blocktree leaves",
	})
	errAncestorOutOfBoundsCheck = errors.New("out of bounds ancestor check")
)

// Hash common.Hash
type Hash = common.Hash

// BlockTree represents the current state with all possible blocks
type BlockTree struct {
	root   *node
	leaves *leafMap
	sync.RWMutex
	runtimes *hashToRuntime
}

// NewEmptyBlockTree creates a BlockTree with a nil head
func NewEmptyBlockTree() *BlockTree {
	return &BlockTree{
		root:     nil,
		leaves:   newEmptyLeafMap(),
		runtimes: newHashToRuntime(),
	}
}

// NewBlockTreeFromRoot initialises a blocktree with a root block. The root block is always the most recently
// finalised block (ie the genesis block if the node is just starting.)
func NewBlockTreeFromRoot(root *types.Header) *BlockTree {
	n := &node{
		hash:        root.Hash(),
		parent:      nil,
		children:    []*node{},
		number:      root.Number,
		arrivalTime: time.Now(),
	}

	return &BlockTree{
		root:     n,
		leaves:   newLeafMap(n),
		runtimes: newHashToRuntime(),
	}
}

// AddBlock inserts the block as child of its parent node
// Note: Assumes block has no children
func (bt *BlockTree) AddBlock(header *types.Header, arrivalTime time.Time) (err error) {
	bt.Lock()
	defer bt.Unlock()

	parent := bt.getNode(header.ParentHash)
	if parent == nil {
		return ErrParentNotFound
	}

	// Check if it already exists
	if n := bt.getNode(header.Hash()); n != nil {
		return ErrBlockExists
	}

	number := parent.number + 1

	if number != header.Number {
		return errUnexpectedNumber
	}

	var isPrimary bool
	if header.Number != 0 {
		isPrimary, err = types.IsPrimary(header)
		if err != nil {
			return fmt.Errorf("failed to check if block was primary: %w", err)
		}
	}

	n := &node{
		hash:        header.Hash(),
		parent:      parent,
		children:    []*node{},
		number:      number,
		arrivalTime: arrivalTime,
		isPrimary:   isPrimary,
	}

	parent.addChild(n)
	bt.leaves.replace(parent, n)

	leavesGauge.Set(float64(len(bt.leaves.nodes())))
	return nil
}

// GetAllBlocksAtNumber will return all blocks hashes with the number of the given hash plus one.
// To find all blocks at a number matching a certain block, pass in that block's parent hash
func (bt *BlockTree) GetAllBlocksAtNumber(hash common.Hash) (hashes []common.Hash) {
	bt.RLock()
	defer bt.RUnlock()

	if bt.getNode(hash) == nil {
		return hashes
	}

	number := bt.getNode(hash).number + 1

	if bt.root.number == number {
		hashes = append(hashes, bt.root.hash)
		return hashes
	}

	return bt.root.getNodesWithNumber(number, hashes)
}

// getNode finds and returns a node based on its Hash. Returns nil if not found.
func (bt *BlockTree) getNode(h Hash) (ret *node) {
	if bt.root.hash == h {
		return bt.root
	}

	for _, leaf := range bt.leaves.nodes() {
		if leaf.hash == h {
			return leaf
		}
	}

	for _, child := range bt.root.children {
		if n := child.getNode(h); n != nil {
			return n
		}
	}

	return nil
}

// Prune sets the given hash as the new blocktree root,
// removing all nodes that are not the new root node or its descendant
// It returns an array of hashes that have been pruned
func (bt *BlockTree) Prune(finalised Hash) (pruned []Hash) {
	bt.Lock()
	defer bt.Unlock()

	if finalised == bt.root.hash {
		return pruned
	}

	n := bt.getNode(finalised)
	if n == nil {
		return pruned
	}

	pruned = bt.root.prune(n, nil)
	bt.root = n
	bt.root.parent = nil

	leaves := n.getLeaves(nil)
	bt.leaves = newEmptyLeafMap()
	for _, leaf := range leaves {
		bt.leaves.store(leaf.hash, leaf)
	}

	for _, hash := range pruned {
		bt.runtimes.delete(hash)
	}

	leavesGauge.Set(float64(len(bt.leaves.nodes())))
	return pruned
}

// String utilises github.com/disiqueira/gotree to create a printable tree
func (bt *BlockTree) String() string {
	bt.RLock()
	defer bt.RUnlock()

	// Construct tree
	tree := gotree.New(bt.root.string())

	for _, child := range bt.root.children {
		sub := tree.Add(child.string())
		child.createTree(sub)
	}

	// Format leaves
	var leaves string
	bt.leaves.smap.Range(func(hash, node interface{}) bool {
		leaves = leaves + fmt.Sprintf("%s\n", hash.(Hash))
		return true
	})

	metadata := fmt.Sprintf("Leaves:\n %s", leaves)

	return fmt.Sprintf("%s\n%s\n", metadata, tree.Print())
}

// subChain returns the path from the node with Hash start to the node with Hash end
func (bt *BlockTree) subChain(start, end Hash) ([]*node, error) {
	sn := bt.getNode(start)
	if sn == nil {
		return nil, ErrStartNodeNotFound
	}
	en := bt.getNode(end)
	if en == nil {
		return nil, ErrEndNodeNotFound
	}
	return sn.subChain(en)
}

// SubBlockchain returns the path from the node with Hash start to the node with Hash end
func (bt *BlockTree) SubBlockchain(start, end Hash) ([]Hash, error) {
	bt.RLock()
	defer bt.RUnlock()

	sc, err := bt.subChain(start, end)
	if err != nil {
		return nil, err
	}
	var bc []Hash
	for _, node := range sc {
		bc = append(bc, node.hash)
	}
	return bc, nil

}

// best returns the best node in the block tree using the fork choice rule.
func (bt *BlockTree) best() *node {
	return bt.leaves.bestBlock()
}

// BestBlockHash returns the hash of the block that is considered "best" based on the
// fork-choice rule. It returns the head of the chain with the most primary blocks.
// If there are multiple chains with the same number of primaries, it returns the one
// with the highest head number.
// If there are multiple chains with the same number of primaries and the same height,
// it returns the one with the head block that arrived the earliest.
func (bt *BlockTree) BestBlockHash() Hash {
	bt.RLock()
	defer bt.RUnlock()

	if bt.leaves == nil {
		// this shouldn't happen
		return Hash{}
	}

	if len(bt.root.children) == 0 {
		return bt.root.hash
	}

	return bt.best().hash
}

// IsDescendantOf returns true if the child is a descendant of parent, false otherwise.
// it returns an error if either the child or parent are not in the blocktree.
func (bt *BlockTree) IsDescendantOf(parent, child Hash) (bool, error) {
	bt.RLock()
	defer bt.RUnlock()

	pn := bt.getNode(parent)
	if pn == nil {
		return false, ErrStartNodeNotFound
	}
	cn := bt.getNode(child)
	if cn == nil {
		return false, ErrEndNodeNotFound
	}
	return cn.isDescendantOf(pn), nil
}

// Leaves returns the leaves of the blocktree as an array
func (bt *BlockTree) Leaves() []Hash {
	bt.RLock()
	defer bt.RUnlock()

	lm := bt.leaves.toMap()
	la := make([]common.Hash, len(lm))
	i := 0

	for k := range lm {
		la[i] = k
		i++
	}

	return la
}

// LowestCommonAncestor returns the lowest common ancestor block hash between two blocks in the tree.
func (bt *BlockTree) LowestCommonAncestor(a, b Hash) (Hash, error) {
	bt.RLock()
	defer bt.RUnlock()

	aNode := bt.getNode(a)
	if aNode == nil {
		return common.Hash{}, ErrNodeNotFound
	}

	bNode := bt.getNode(b)
	if bNode == nil {
		return common.Hash{}, ErrNodeNotFound
	}
	return lowestCommonAncestor(aNode, bNode), nil
}
func lowestCommonAncestor(aNode, bNode *node) Hash {
	higherNode := bNode
	lowerNode := aNode
	if aNode.number > bNode.number {
		higherNode = aNode
		lowerNode = bNode
	}

	higherNum := higherNode.number
	lowerNum := lowerNode.number
	diff := higherNum - lowerNum
	for diff > 0 {
		if higherNode.parent == nil {
			panic(fmt.Errorf("%w: for block number %v", errAncestorOutOfBoundsCheck, higherNum))
		}
		higherNode = higherNode.parent
		diff--
	}

	for {
		if higherNode.hash == lowerNode.hash {
			return higherNode.hash
		} else if higherNode.parent == nil || lowerNode.parent == nil {
			panic(fmt.Errorf("%w: for block number %v", errAncestorOutOfBoundsCheck, higherNum))
		}
		higherNode = higherNode.parent
		lowerNode = lowerNode.parent
	}
}

// GetAllBlocks returns all the blocks in the tree
func (bt *BlockTree) GetAllBlocks() []Hash {
	bt.RLock()
	defer bt.RUnlock()

	return bt.root.getAllDescendants(nil)
}

// GetAllDescendants returns all block hashes that are descendants of the given block hash.
func (bt *BlockTree) GetAllDescendants(hash common.Hash) ([]Hash, error) {
	bt.RLock()
	defer bt.RUnlock()

	node := bt.getNode(hash)
	if node == nil {
		return nil, fmt.Errorf("%w: for block hash %s", ErrNodeNotFound, hash)
	}

	return node.getAllDescendants(nil), nil
}

// GetHashByNumber returns the block hash with the given number that is on the best chain.
// If the number is lower or higher than the numbers in the blocktree, an error is returned.
func (bt *BlockTree) GetHashByNumber(num uint) (common.Hash, error) {
	bt.RLock()
	defer bt.RUnlock()

	best := bt.leaves.bestBlock()
	if best.number < num {
		return common.Hash{}, ErrNumGreaterThanHighest
	}

	if best.number == num {
		return best.hash, nil
	}

	if bt.root.number > num {
		return common.Hash{}, ErrNumLowerThanRoot
	}

	if bt.root.number == num {
		return bt.root.hash, nil
	}

	curr := best.parent
	for {
		if curr == nil {
			return common.Hash{}, ErrNodeNotFound
		}

		if curr.number == num {
			return curr.hash, nil
		}

		curr = curr.parent
	}
}

// GetArrivalTime returns the arrival time of a block
func (bt *BlockTree) GetArrivalTime(hash common.Hash) (time.Time, error) {
	bt.RLock()
	defer bt.RUnlock()

	n := bt.getNode(hash)
	if n == nil {
		return time.Time{}, ErrNodeNotFound
	}

	return n.arrivalTime, nil
}

// DeepCopy returns a copy of the BlockTree
func (bt *BlockTree) DeepCopy() *BlockTree {
	bt.RLock()
	defer bt.RUnlock()

	btCopy := &BlockTree{}

	if bt.root == nil {
		return btCopy
	}

	btCopy.root = bt.root.deepCopy(nil)

	if bt.leaves != nil {
		btCopy.leaves = newEmptyLeafMap()

		lMap := bt.leaves.toMap()
		for hash, val := range lMap {
			btCopy.leaves.store(hash, btCopy.getNode(val.hash))
		}
	}

	return btCopy
}

// StoreRuntime stores the runtime for corresponding block hash.
func (bt *BlockTree) StoreRuntime(hash common.Hash, in runtime.Instance) {
	bt.runtimes.set(hash, in)
}

// GetBlockRuntime returns block runtime for corresponding block hash.
func (bt *BlockTree) GetBlockRuntime(hash common.Hash) (runtime.Instance, error) {
	ins := bt.runtimes.get(hash)
	if ins == nil {
		return nil, ErrFailedToGetRuntime
	}
	return ins, nil
}
