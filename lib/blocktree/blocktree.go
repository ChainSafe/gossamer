// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/disiqueira/gotree"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/exp/maps"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "blocktree"))
var (
	leavesGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_block",
		Name:      "leaves_total",
		Help:      "total number of blocktree leaves",
	})
	inMemoryRuntimesGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_blocktree",
		Name:      "runtimes_total",
		Help:      "total number of runtimes stored in the in-memory blocktree",
	})
	errAncestorOutOfBoundsCheck = errors.New("out of bounds ancestor check")
	ErrRuntimeNotFound          = errors.New("runtime not found")
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

// GetHashesAtNumber will return all blocks hashes that contains the number of the given hash plus one.
// To find all blocks at a number matching a certain block, pass in that block's parent hash
func (bt *BlockTree) GetHashesAtNumber(number uint) (hashes []common.Hash) {
	bt.RLock()
	defer bt.RUnlock()

	if number < bt.root.number {
		return []common.Hash{}
	}

	bestLeave := bt.leaves.bestBlock()
	if number > bestLeave.number {
		return []common.Hash{}
	}

	possibleNumOfBlocks := len(bt.leaves.nodes())
	hashes = make([]common.Hash, 0, possibleNumOfBlocks)
	return bt.root.hashesAtNumber(number, hashes)
}

var ErrStartGreaterThanEnd = errors.New("start greater than end")
var ErrNilBlockInRange = errors.New("nil block in range")

// Range will return all the blocks between the start and end hash inclusive.
// If the end hash does not exist in the blocktree then an error is returned.
// If the start hash does not exist in the blocktree then we will return all blocks
// between the end and the blocktree root inclusive
func (bt *BlockTree) Range(startHash common.Hash, endHash common.Hash) (hashes []common.Hash, err error) {
	bt.Lock()
	defer bt.Unlock()

	endNode := bt.getNode(endHash)
	if endNode == nil {
		return nil, fmt.Errorf("%w: %s", ErrEndNodeNotFound, endHash)
	}

	// if we don't find the start hash in the blocktree
	// that means it should be in the database, so we retrieve
	// as many nodes as we can, in other words we get all the
	// blocks from the end hash till the bt.root inclusive
	startNode := bt.getNode(startHash)
	if startNode == nil {
		startNode = bt.root
	}

	hashes, err = accumulateHashesInDescedingOrder(endNode, startNode)
	if err != nil {
		return nil, fmt.Errorf("getting blocks in range: %w", err)
	}

	return hashes, nil
}

// RangeInMemory returns the path from the node with Hash start to the node with Hash end.
// If the end hash does not exist in the blocktree then an error is returned.
// Different from blocktree.Range, if the start node is not found in the in memory blocktree
func (bt *BlockTree) RangeInMemory(startHash common.Hash, endHash common.Hash) (hashes []common.Hash, err error) {
	bt.Lock()
	defer bt.Unlock()

	endNode := bt.getNode(endHash)
	if endNode == nil {
		return nil, fmt.Errorf("%w: %s", ErrEndNodeNotFound, endHash)
	}

	startNode := bt.getNode(startHash)
	if startNode == nil {
		return nil, fmt.Errorf("%w: %s", ErrStartNodeNotFound, endHash)
	}

	if startNode.number > endNode.number {
		return nil, fmt.Errorf("%w", ErrStartGreaterThanEnd)
	}

	hashes, err = accumulateHashesInDescedingOrder(endNode, startNode)
	if err != nil {
		return nil, fmt.Errorf("getting blocks in range: %w", err)
	}

	return hashes, nil
}

func accumulateHashesInDescedingOrder(endNode, startNode *node) (
	hashes []common.Hash, err error) {

	if startNode.number > endNode.number {
		return nil, fmt.Errorf("%w", ErrStartGreaterThanEnd)
	}

	// blocksInRange is the difference between the end number to start number
	// but the difference don't includes the start item that is why we add 1
	blocksInRange := endNode.number - startNode.number + 1
	hashes = make([]common.Hash, blocksInRange)

	lastPosition := blocksInRange - 1
	hashes[0] = startNode.hash

	for position := lastPosition; position > 0; position-- {
		currentNodeHash := endNode.hash
		hashes[position] = currentNodeHash

		endNode = endNode.parent

		if endNode == nil {
			return nil, fmt.Errorf("%w: missing parent of %s",
				ErrNilBlockInRange, currentNodeHash)
		}
	}

	return hashes, nil
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

	// Cleanup in-memory runtimes from the canonical chain.
	// The runtime used in the newly finalised block is kept
	// instantiated in memory, and all other runtimes are
	// stopped and removed from memory. Note these are still
	// accessible through the storage as WASM blob.
	previousFinalisedBlock := bt.root
	newCanonicalChainBlocksCount := n.number - previousFinalisedBlock.number
	if previousFinalisedBlock.number == 0 { // include the genesis block
		newCanonicalChainBlocksCount++
	}
	canonicalChainBlock := n
	newCanonicalChainBlockHashes := make([]common.Hash, newCanonicalChainBlocksCount)
	for i := int(newCanonicalChainBlocksCount) - 1; i >= 0; i-- {
		newCanonicalChainBlockHashes[i] = canonicalChainBlock.hash
		canonicalChainBlock = canonicalChainBlock.parent
	}

	bt.runtimes.onFinalisation(newCanonicalChainBlockHashes)

	pruned = bt.root.prune(n, nil)
	bt.root = n
	bt.root.parent = nil

	leaves := n.getLeaves(nil)
	bt.leaves = newEmptyLeafMap()
	for _, leaf := range leaves {
		bt.leaves.store(leaf.hash, leaf)
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
	bt.leaves.smap.Range(func(hash, _ interface{}) bool {
		leaves = leaves + fmt.Sprintf("%s\n", hash.(Hash))
		return true
	})

	metadata := fmt.Sprintf("Leaves:\n %s", leaves)

	return fmt.Sprintf("%s\n%s\n", metadata, tree.Print())
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
// If parent and child are the same, we return true.
func (bt *BlockTree) IsDescendantOf(parent, child Hash) (bool, error) {
	if parent == child {
		return true, nil
	}

	bt.RLock()
	defer bt.RUnlock()

	pn := bt.getNode(parent)
	if pn == nil {
		return false, fmt.Errorf("%w: node hash %s", ErrStartNodeNotFound, parent)
	}
	cn := bt.getNode(child)
	if cn == nil {
		return false, fmt.Errorf("%w: node hash %s", ErrEndNodeNotFound, child)
	}
	return cn.isDescendantOf(pn), nil
}

// Leaves returns the leaves of the blocktree as an array
func (bt *BlockTree) Leaves() []Hash {
	bt.RLock()
	defer bt.RUnlock()

	lm := bt.leaves.toMap()
	return maps.Keys(lm)
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

// GetAllDescendants returns all block hashes that are descendants of the given block hash (including itself).
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
func (bt *BlockTree) StoreRuntime(hash common.Hash, instance runtime.Instance) {
	bt.runtimes.set(hash, instance)
}

// GetBlockRuntime returns the runtime corresponding to the given block hash. If there is no instance for
// the given block hash it will lookup an instance of an ancestor and return it.
func (bt *BlockTree) GetBlockRuntime(hash common.Hash) (runtime.Instance, error) {
	// if the current node contains a runtime entry in the runtime mapping
	// then we early return the instance, otherwise we will lookup for the
	// closest parent with a runtime instance entry in the mapping
	runtimeInstance := bt.runtimes.get(hash)
	if runtimeInstance != nil {
		return runtimeInstance, nil
	}

	bt.RLock()
	defer bt.RUnlock()

	currentNode := bt.getNode(hash)
	if currentNode == nil {
		return nil, fmt.Errorf("%w: for block hash %s", ErrNodeNotFound, hash)
	}

	currentNode = currentNode.parent
	for currentNode != nil {
		runtimeInstance := bt.runtimes.get(currentNode.hash)
		if runtimeInstance != nil {
			return runtimeInstance, nil
		}

		currentNode = currentNode.parent
	}

	return nil, nil
}
