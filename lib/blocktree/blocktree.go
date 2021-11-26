// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/disiqueira/gotree"
)

// Hash common.Hash
type Hash = common.Hash

// BlockTree represents the current state with all possible blocks
type BlockTree struct {
	root   *node
	leaves *leafMap
	sync.RWMutex
	nodeCache map[Hash]*node
	runtime   *sync.Map
}

// NewEmptyBlockTree creates a BlockTree with a nil head
func NewEmptyBlockTree() *BlockTree {
	return &BlockTree{
		root:      nil,
		leaves:    newEmptyLeafMap(),
		nodeCache: make(map[Hash]*node),
		runtime:   &sync.Map{}, // map[Hash]runtime.Instance
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
		root:      n,
		leaves:    newLeafMap(n),
		nodeCache: make(map[Hash]*node),
		runtime:   &sync.Map{},
	}
}

// GenesisHash returns the hash of the genesis block
func (bt *BlockTree) GenesisHash() Hash {
	bt.RLock()
	defer bt.RUnlock()
	return bt.root.hash
}

// AddBlock inserts the block as child of its parent node
// Note: Assumes block has no children
func (bt *BlockTree) AddBlock(header *types.Header, arrivalTime time.Time) error {
	bt.Lock()
	defer bt.Unlock()

	parent := bt.getNode(header.ParentHash)
	if parent == nil {
		return ErrParentNotFound
	}

	// Check if it already exists
	n := bt.getNode(header.Hash())
	if n != nil {
		return ErrBlockExists
	}

	number := big.NewInt(0)
	number.Add(parent.number, big.NewInt(1))

	if number.Cmp(header.Number) != 0 {
		return errUnexpectedNumber
	}

	n = &node{
		hash:        header.Hash(),
		parent:      parent,
		children:    []*node{},
		number:      number,
		arrivalTime: arrivalTime,
	}
	parent.addChild(n)
	bt.leaves.replace(parent, n)
	bt.setInCache(n)

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

	number := big.NewInt(0).Add(bt.getNode(hash).number, big.NewInt(1))

	if bt.root.number.Cmp(number) == 0 {
		hashes = append(hashes, bt.root.hash)
		return hashes
	}

	return bt.root.getNodesWithNumber(number, hashes)
}

func (bt *BlockTree) setInCache(b *node) {
	if b == nil {
		return
	}

	if _, has := bt.nodeCache[b.hash]; !has {
		bt.nodeCache[b.hash] = b
	}
}

// getNode finds and returns a node based on its Hash. Returns nil if not found.
func (bt *BlockTree) getNode(h Hash) (ret *node) {
	defer func() { bt.setInCache(ret) }()

	if b, ok := bt.nodeCache[h]; ok {
		return b
	}

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
	defer func() {
		for _, hash := range pruned {
			delete(bt.nodeCache, hash)
			bt.runtime.Delete(hash)
		}
	}()

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

// longestPath returns the path from the root to leftmost deepest leaf in BlockTree BT
func (bt *BlockTree) longestPath() []*node {
	dl := bt.deepestLeaf()
	var path []*node
	for curr := dl; ; curr = curr.parent {
		path = append([]*node{curr}, path...)
		if curr.parent == nil {
			return path
		}
	}
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

// deepestLeaf returns the leftmost deepest leaf in the block tree.
func (bt *BlockTree) deepestLeaf() *node {
	return bt.leaves.deepestLeaf()
}

// DeepestBlockHash returns the hash of the deepest block in the blocktree
// If there is multiple deepest blocks, it returns the one with the earliest arrival time.
func (bt *BlockTree) DeepestBlockHash() Hash {
	bt.RLock()
	defer bt.RUnlock()

	if bt.leaves == nil {
		return Hash{}
	}

	deepest := bt.leaves.deepestLeaf()
	if deepest == nil {
		return Hash{}
	}

	return deepest.hash
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

// HighestCommonAncestor returns the highest block that is a Ancestor to both a and b
func (bt *BlockTree) HighestCommonAncestor(a, b Hash) (Hash, error) {
	bt.RLock()
	defer bt.RUnlock()

	an := bt.getNode(a)
	if an == nil {
		return common.Hash{}, ErrNodeNotFound
	}

	bn := bt.getNode(b)
	if bn == nil {
		return common.Hash{}, ErrNodeNotFound
	}

	ancestor := an.highestCommonAncestor(bn)
	if ancestor == nil {
		// this case shouldn't happen - any two nodes in the blocktree must
		// have a common ancestor, the lowest of which is the root node
		return common.Hash{}, fmt.Errorf("%w: %s and %s", ErrNoCommonAncestor, a, b)
	}

	return ancestor.hash, nil
}

// GetAllBlocks returns all the blocks in the tree
func (bt *BlockTree) GetAllBlocks() []Hash {
	bt.RLock()
	defer bt.RUnlock()

	return bt.root.getAllDescendants(nil)
}

// GetHashByNumber returns the block hash with the given number that is on the best chain.
// If the number is lower or higher than the numbers in the blocktree, an error is returned.
func (bt *BlockTree) GetHashByNumber(num *big.Int) (common.Hash, error) {
	bt.RLock()
	defer bt.RUnlock()

	deepest := bt.leaves.deepestLeaf()
	if deepest.number.Cmp(num) == -1 {
		return common.Hash{}, ErrNumGreaterThanHighest
	}

	if deepest.number.Cmp(num) == 0 {
		return deepest.hash, nil
	}

	if bt.root.number.Cmp(num) == 1 {
		return common.Hash{}, ErrNumLowerThanRoot
	}

	if bt.root.number.Cmp(num) == 0 {
		return bt.root.hash, nil
	}

	curr := deepest.parent
	for {
		if curr == nil {
			return common.Hash{}, ErrNodeNotFound
		}

		if curr.number.Cmp(num) == 0 {
			return curr.hash, nil
		}

		curr = curr.parent
	}
}

// GetArrivalTime returns the arrival time of a block
func (bt *BlockTree) GetArrivalTime(hash common.Hash) (time.Time, error) {
	bt.RLock()
	defer bt.RUnlock()

	n, has := bt.nodeCache[hash]
	if !has {
		return time.Time{}, ErrNodeNotFound
	}

	return n.arrivalTime, nil
}

// DeepCopy returns a copy of the BlockTree
func (bt *BlockTree) DeepCopy() *BlockTree {
	bt.RLock()
	defer bt.RUnlock()

	btCopy := &BlockTree{
		nodeCache: make(map[Hash]*node),
	}

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

	for hash := range bt.nodeCache {
		btCopy.nodeCache[hash] = btCopy.getNode(hash)
	}

	return btCopy
}

// StoreRuntime stores the runtime for corresponding block hash.
func (bt *BlockTree) StoreRuntime(hash common.Hash, in runtime.Instance) {
	bt.runtime.Store(hash, in)
}

// GetBlockRuntime returns block runtime for corresponding block hash.
func (bt *BlockTree) GetBlockRuntime(hash common.Hash) (runtime.Instance, error) {
	ins, ok := bt.runtime.Load(hash)
	if !ok {
		return nil, ErrFailedToGetRuntime
	}
	return ins.(runtime.Instance), nil
}
