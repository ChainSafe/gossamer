// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package forktree

import (
	"errors"
	"iter"
	"math"
	"slices"

	"golang.org/x/exp/constraints"
)

var (
	// Adding duplicate node to tree.
	ErrDuplicate = errors.New("hash already exists in tree")
	// Finalizing descendent of tree node without finalizing ancestor(s).
	ErrUnfinalizedAncestor = errors.New("finalized descendent of tree node without finalizing its ancestor(s) first")
	// Imported or finalized node that is an ancestor of previously finalized node.
	ErrRevert = errors.New("tried to import or finalize node that is an ancestor of a previously finalized node")
)

// Result of finalizing a node (that could be a part of the tree or not).
type FinalizationResult interface {
	isFinalizationResult()
}

// Filtering action.
type FilterAction uint

const (
	// Remove the node and its subtree.
	FilterActionRemove FilterAction = iota + 1
	// Maintain the node.
	FilterActionKeepNode
	// Maintain the node and its subtree.
	FilterActionKeepTree
)

// The tree has changed, optionally return the value associated with the finalized node.
type FinalizationResultChanged[V any] struct {
	Value *V
}

func (FinalizationResultChanged[V]) isFinalizationResult() {}

// The tree has not changed.
type FinalizationResultUnchanged struct{}

func (FinalizationResultUnchanged) isFinalizationResult() {}

type node[H comparable, N constraints.Integer, V any] struct {
	Hash     H
	Number   N
	Data     V
	Children []node[H, N, V]
}

type nodeHeight[H comparable, N constraints.Integer, V any] struct {
	node   node[H, N, V]
	height uint
}

type nodeParentIdx[H comparable, N constraints.Integer, V any] struct {
	node      node[H, N, V]
	parentIdx uint
}

// Finds the max depth among all branches descendent from this node.
func (n node[H, N, V]) MaxDepth() uint {
	var max uint = 0
	var stack []nodeHeight[H, N, V] = make([]nodeHeight[H, N, V], 0)
	stack = append(stack, nodeHeight[H, N, V]{node: n, height: 0})

	for len(stack) > 0 {
		popped := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := popped.node
		height := popped.height
		if height > max {
			max = height
		}

		for _, child := range node.Children {
			stack = append(stack, nodeHeight[H, N, V]{node: child, height: height + 1})
		}
	}
	return max
}

// A tree data structure that stores several nodes across multiple branches.
//
// Top-level branches are called roots. The tree has functionality for
// finalizing nodes, which means that node is traversed, and all competing
// branches are pruned. It also guarantees that nodes in the tree are finalized
// in order. Each node is uniquely identified by its hash but can be ordered by
// its number. In order to build the tree an external function must be provided
// when interacting with the tree to establish a node's ancestry.
type ForkTree[H comparable, N constraints.Integer, V any] struct {
	roots               []node[H, N, V]
	bestFinalizedNumber *N
}

// Create a new empty tree instance.
func NewForkTree[H comparable, N constraints.Integer, V any]() ForkTree[H, N, V] {
	return ForkTree[H, N, V]{
		roots:               nil,
		bestFinalizedNumber: nil,
	}
}

func (ft *ForkTree[H, N, V]) Clone() ForkTree[H, N, V] {
	var bfn N
	if ft.bestFinalizedNumber != nil {
		bfn = *ft.bestFinalizedNumber
		return ForkTree[H, N, V]{
			roots:               slices.Clone(ft.roots),
			bestFinalizedNumber: &bfn,
		}
	}
	return ForkTree[H, N, V]{
		roots:               slices.Clone(ft.roots),
		bestFinalizedNumber: nil,
	}
}

// Rebalance the tree.
//
// For each tree level sort child nodes by max branch depth (decreasing).
//
// Most operations in the tree are performed with depth-first search
// starting from the leftmost node at every level, since this tree is meant
// to be used in a blockchain context, a good heuristic is that the node
// we'll be looking for at any point will likely be in one of the deepest chains
// (i.e. the longest ones).
func (ft *ForkTree[H, N, V]) Rebalance() {
	slices.SortStableFunc(ft.roots, func(i, j node[H, N, V]) int {
		iMaxDepth := i.MaxDepth()
		jMaxDepth := j.MaxDepth()
		if iMaxDepth > jMaxDepth {
			return -1
		} else if iMaxDepth < jMaxDepth {
			return 1
		} else {
			return 0
		}
	})
	stack := make([]*node[H, N, V], len(ft.roots))
	for i := range ft.roots {
		stack[i] = &ft.roots[i]
	}

	for len(stack) > 0 {
		popped := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		slices.SortStableFunc(popped.Children, func(i, j node[H, N, V]) int {
			iMaxDepth := i.MaxDepth()
			jMaxDepth := j.MaxDepth()
			if iMaxDepth > jMaxDepth {
				return -1
			} else if iMaxDepth < jMaxDepth {
				return 1
			} else {
				return 0
			}
		})
		for _, child := range popped.Children {
			stack = append(stack, &child)
		}
	}
}

// Import a new node into the tree.
//
// The given function isDescendentOf should return true if the second
// hash (target) is a descendent of the first hash (base).
//
// This method assumes that nodes in the same branch are imported in order.
//
// Returns true if the imported node is a root.
// WARNING: some users of this method (i.e. consensus epoch changes tree) currently silently
// rely on a **post-order DFS** traversal. If we are using instead a top-down traversal method
// then the isDescendentOf closure, when used after a warp-sync, may end up querying the
// backend for a block (the one corresponding to the root) that is not present and thus will
// return a wrong result.
func (ft *ForkTree[H, N, V]) Import(
	hash H,
	number N,
	data V,
	isDescendentOf func(a, b H) (bool, error),
) (bool, error) {
	if ft.bestFinalizedNumber != nil {
		if number <= *ft.bestFinalizedNumber {
			return false, ErrRevert
		}
	}

	parent, err := ft.FindNodeWhereMut(hash, number, isDescendentOf, func(v V) bool { return true })
	if err != nil {
		return false, err
	}
	var children *[]node[H, N, V]
	var isRoot bool
	if parent != nil {
		children = &parent.Children
		isRoot = false
	} else {
		children = &ft.roots
		isRoot = true
	}

	for _, elem := range *children {
		if elem.Hash == hash {
			return false, ErrDuplicate
		}
	}

	*children = append(*children, node[H, N, V]{
		Hash:     hash,
		Number:   number,
		Data:     data,
		Children: make([]node[H, N, V], 0),
	})

	if len(*children) == 1 {
		// Rebalance may be required only if we've extended the branch depth.
		ft.Rebalance()
	}

	return isRoot, nil
}

type Node[H comparable, N constraints.Integer, V any] struct {
	Hash   H
	Number N
	Data   V
}

// Iterates over the existing roots in the tree.
func (ft *ForkTree[H, N, V]) Roots() []Node[H, N, V] {
	roots := make([]Node[H, N, V], len(ft.roots))
	for i, node := range ft.roots {
		roots[i] = Node[H, N, V]{
			Hash:   node.Hash,
			Number: node.Number,
			Data:   node.Data,
		}
	}
	return roots
}

func (ft *ForkTree[H, N, V]) nodeIter() iter.Seq[node[H, N, V]] {
	// we need to reverse the order of roots to maintain the expected
	// ordering since the iterator uses a stack to track state.
	stack := slices.Clone(ft.roots)
	slices.Reverse(stack)
	fti := forkTreeIterator[H, N, V]{
		stack: stack,
	}
	return fti.iter()
}

func (ft *ForkTree[H, N, V]) Iter() iter.Seq[Node[H, N, V]] {
	return func(yield func(Node[H, N, V]) bool) {
		for node := range ft.nodeIter() {
			if !yield(Node[H, N, V]{
				Hash:   node.Hash,
				Number: node.Number,
				Data:   node.Data,
			}) {
				return
			}
		}
	}
}

// Map fork tree into values of new types.
//
// Tree traversal technique (e.g. BFS vs DFS) is left as not specified and
// may be subject to change in the future. In other words, your predicates
// should not rely on the observed traversal technique currently in use.
func Map[H comparable, N constraints.Integer, V any, VT any](
	ft ForkTree[H, N, V], f func(H, N, V) VT,
) ForkTree[H, N, VT] {
	var (
		queue     []nodeParentIdx[H, N, V]  = make([]nodeParentIdx[H, N, V], 0, len(ft.roots))
		nextQueue []nodeParentIdx[H, N, V]  = make([]nodeParentIdx[H, N, V], 0)
		output    []nodeParentIdx[H, N, VT] = make([]nodeParentIdx[H, N, VT], 0)
	)

	for i := len(ft.roots) - 1; i >= 0; i-- {
		queue = append(queue, nodeParentIdx[H, N, V]{
			node:      ft.roots[i],
			parentIdx: math.MaxUint,
		})
	}

	for len(queue) > 0 {
		for len(queue) > 0 {
			popped := queue[0]
			queue = queue[1:]
			n := popped.node
			parentIndex := popped.parentIdx

			newData := f(n.Hash, n.Number, n.Data)
			newNode := node[H, N, VT]{
				Hash:     n.Hash,
				Number:   n.Number,
				Data:     newData,
				Children: make([]node[H, N, VT], 0, len(n.Children)),
			}

			nodeID := uint(len(output))
			output = append(output, nodeParentIdx[H, N, VT]{node: newNode, parentIdx: parentIndex})

			for i := len(n.Children) - 1; i >= 0; i-- {
				child := n.Children[i]
				nextQueue = append(nextQueue, nodeParentIdx[H, N, V]{node: child, parentIdx: nodeID})
			}
		}

		temp := queue
		queue = nextQueue
		nextQueue = temp
	}

	var roots []node[H, N, VT]
	for len(output) > 0 {
		elem := output[len(output)-1]
		output = output[:len(output)-1]
		parentIndex := elem.parentIdx
		newNode := elem.node
		if parentIndex == math.MaxUint {
			roots = append(roots, newNode)
		} else {
			output[parentIndex].node.Children = append(output[parentIndex].node.Children, newNode)
		}
	}

	return ForkTree[H, N, VT]{
		roots:               roots,
		bestFinalizedNumber: ft.bestFinalizedNumber,
	}
}

// Find a node in the tree that is the deepest ancestor of the given
// block hash and which passes the given predicate.
//
// The given function isDescendentOf should return true if the
// second hash (target) is a descendent of the first hash (base).
func (ft *ForkTree[H, N, V]) FindNodeWhere(
	hash H,
	number N,
	isDescendentOf func(a, b H) (bool, error),
	predicate func(V) bool,
) (*node[H, N, V], error) {
	maybePath, err := ft.FindNodeIndexWhere(hash, number, isDescendentOf, predicate)
	if err != nil {
		return nil, err
	}
	if maybePath == nil {
		return nil, nil
	}
	var children []node[H, N, V]
	curr := ft.roots
	for _, i := range maybePath[:len(maybePath)-1] {
		curr = curr[i].Children
	}
	children = curr
	n := children[maybePath[len(maybePath)-1]]
	return &n, nil
}

// Same as [ForkTree.FindNodeWhere], but returns mutable reference.
func (ft *ForkTree[H, N, V]) FindNodeWhereMut(
	hash H,
	number N,
	isDescendentOf func(a, b H) (bool, error),
	predicate func(V) bool,
) (*node[H, N, V], error) {
	maybePath, err := ft.FindNodeIndexWhere(hash, number, isDescendentOf, predicate)
	if err != nil {
		return nil, err
	}
	if maybePath == nil {
		return nil, nil
	}

	var children *[]node[H, N, V]
	curr := &ft.roots
	for _, i := range maybePath[:len(maybePath)-1] {
		curr = &(*curr)[i].Children
	}
	children = curr
	return &(*children)[maybePath[len(maybePath)-1]], nil
}

// Same as [ForkTree.FindNodeWhere], but returns indices.
//
// The returned indices represent the full path to reach the matching node starting
// from one of the roots, i.e. the earliest index in the traverse path goes first,
// and the final index in the traverse path goes last.
//
// If a node is found that matches the predicate the returned path should always
// contain at least one index, otherwise nil is returned.
//
// WARNING: some users of this method (i.e. consensus epoch changes tree) currently silently
// rely on a **post-order DFS** traversal. If we are using instead a top-down traversal method
// then the  isDescendentOf closure, when used after a warp-sync, will end up querying the
// backend for a block (the one corresponding to the root) that is not present and thus will
// return a wrong result.
func (ft *ForkTree[H, N, V]) FindNodeIndexWhere(
	hash H,
	number N,
	isDescendentOf func(a, b H) (bool, error),
	predicate func(V) bool,
) ([]uint, error) {
	var (
		stack        []nodeParentIdx[H, N, V] = make([]nodeParentIdx[H, N, V], 0)
		rootIdx      uint                     = 0
		found        bool                     = false
		isDescendent bool                     = false
	)

	for rootIdx < uint(len(ft.roots)) {
		if number <= ft.roots[rootIdx].Number {
			rootIdx++
			continue
		}
		// The second element in the stack tuple tracks what is the **next** children
		// index to search into. If we find an ancestor then we stop searching into
		// alternative branches and we focus on the current path up to the root.
		stack = append(stack, nodeParentIdx[H, N, V]{node: ft.roots[rootIdx], parentIdx: 0})
		for len(stack) > 0 {
			popped := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			node := popped.node
			i := popped.parentIdx

			if i < uint(len(node.Children)) && !isDescendent {
				stack = append(stack, nodeParentIdx[H, N, V]{node: node, parentIdx: i + 1})
				if node.Children[i].Number < number {
					stack = append(stack, nodeParentIdx[H, N, V]{node: node.Children[i], parentIdx: 0})
				}
			} else {
				if isDescendent {
					isDescendent = true
					if predicate(node.Data) {
						found = true
						break
					}
				} else {
					is, err := isDescendentOf(node.Hash, hash)
					if err != nil {
						return nil, err
					}
					if is {
						isDescendent = true
						if predicate(node.Data) {
							found = true
							break
						}
					}
				}
			}
		}

		// If the element we are looking for is a descendent of the current root
		// then we can stop the search.
		if isDescendent {
			break
		}
		rootIdx++
	}

	if found {
		// The path is the root index followed by the indices of all the children
		// we were processing when we found the element (remember the stack
		// contains the index of the **next** children to process).
		path := make([]uint, 0, len(stack)+1)
		path = append(path, rootIdx)
		for _, elem := range stack {
			path = append(path, elem.parentIdx-1)
		}
		return path, nil
	} else {
		return nil, nil
	}
}

// Prune the tree, removing all non-canonical nodes.
//
// We find the node in the tree that is the deepest ancestor of the given hash
// and that passes the given predicate. If such a node exists, we re-root the
// tree to this node. Otherwise the tree remains unchanged.
//
// The given function isDescendentOf should return true if the second
// hash (target) is a descendent of the first hash (base).
//
// Returns all pruned nodes data.
func (ft *ForkTree[H, N, V]) Prune(
	hash H,
	number N,
	isDescendentOf func(a, b H) (bool, error),
	predicate func(V) bool,
) (iter.Seq[Node[H, N, V]], error) {
	path, err := ft.FindNodeIndexWhere(hash, number, isDescendentOf, predicate)
	if err != nil {
		return nil, err
	}
	if path == nil {
		ri := removedIterator[H, N, V]{stack: nil}
		return ri.iter(), nil
	}
	newRootPath := path

	removed := ft.roots
	ft.roots = nil

	// Find and detach the new root from the removed nodes
	rootSiblings := &removed
	for _, idx := range newRootPath[:len(newRootPath)-1] {
		rootSiblings = &(*rootSiblings)[idx].Children
	}
	root := (*rootSiblings)[newRootPath[len(newRootPath)-1]]
	*rootSiblings = append(
		(*rootSiblings)[:newRootPath[len(newRootPath)-1]], (*rootSiblings)[newRootPath[len(newRootPath)-1]+1:]...)
	ft.roots = []node[H, N, V]{root}

	// If, because of the predicate, the new root is not the deepest ancestor
	// of hash then we can remove all the nodes that are descendants of the new
	// root but not ancestors of hash.
	curr := &ft.roots[0]
	for {
		var maybeAncestorIdx *int
		for idx, child := range curr.Children {
			if child.Number < number {
				is, err := isDescendentOf(child.Hash, hash)
				if err != nil {
					return nil, err
				}
				if is {
					maybeAncestorIdx = &idx
					break
				}
			}
		}
		if maybeAncestorIdx == nil {
			// Now we are positioned just above block identified by hash
			break
		}
		// Preserve only the ancestor node, the siblings are removed
		nextSiblings := curr.Children
		curr.Children = nil
		next := nextSiblings[*maybeAncestorIdx]
		nextSiblings = append(nextSiblings[:*maybeAncestorIdx], nextSiblings[*maybeAncestorIdx+1:]...)
		curr.Children = []node[H, N, V]{next}
		removed = append(removed, nextSiblings...)
		curr = &curr.Children[0]
	}

	// Curr now points to our direct ancestor, if necessary remove any node that is
	// not a descendant of hash.
	children := curr.Children
	curr.Children = nil
	for _, child := range children {
		if child.Number == number && child.Hash == hash {
			curr.Children = append(curr.Children, child)
		} else if number < child.Number {
			is, err := isDescendentOf(hash, child.Hash)
			if err != nil {
				return nil, err
			}
			if is {
				curr.Children = append(curr.Children, child)
			} else {
				removed = append(removed, child)
			}
		} else {
			removed = append(removed, child)
		}
	}

	ft.Rebalance()

	ri := removedIterator[H, N, V]{stack: removed}
	return ri.iter(), nil
}

// Finalize a root in the tree and return it, return nil in case no root
// with the given hash exists. All other roots are pruned, and the children
// of the finalized node become the new roots.
func (ft *ForkTree[H, N, V]) FinalizeRoot(hash H) *V {
	var pos int = -1
	for i, root := range ft.roots {
		if root.Hash == hash {
			pos = i
			break
		}
	}
	if pos < 0 {
		return nil
	}
	v := ft.FinalizeRootAt(pos)
	return &v
}

// Finalize root at given position. See FinalizeRoot comment for details.
func (ft *ForkTree[H, N, V]) FinalizeRootAt(position int) V {
	node := ft.roots[position]
	ft.roots = node.Children
	ft.bestFinalizedNumber = &node.Number
	return node.Data
}

// Finalize a node in the tree. This method will make sure that the node
// being finalized is either an existing root (and return its data), or a
// node from a competing branch (not in the tree), tree pruning is done
// accordingly. The given function isDescendentOf should return true
// if the second hash (target) is a descendent of the first hash (base).
func (ft *ForkTree[H, N, V]) Finalize(
	hash H,
	number N,
	isDescendentOf func(a, b H) (bool, error),
) (FinalizationResult, error) {
	if ft.bestFinalizedNumber != nil {
		if number <= *ft.bestFinalizedNumber {
			return nil, ErrRevert
		}
	}

	root := ft.FinalizeRoot(hash)
	if root != nil {
		return FinalizationResultChanged[V]{root}, nil
	}

	// make sure we're not finalizing a descendent of any root
	for _, root := range ft.roots {
		if number > root.Number {
			is, err := isDescendentOf(root.Hash, hash)
			if err != nil {
				return nil, err
			}
			if is {
				return nil, ErrUnfinalizedAncestor
			}
		}
	}

	// we finalized a block earlier than any existing root (or possibly
	// another fork not part of the tree). make sure to only keep roots that
	// are part of the finalized branch
	var changed bool
	roots := ft.roots
	ft.roots = nil

	for _, root := range roots {
		if root.Number > number {
			is, err := isDescendentOf(hash, root.Hash)
			if err != nil {
				return nil, err
			}
			if is {
				ft.roots = append(ft.roots, root)
			} else {
				changed = true
			}
		} else {
			changed = true
		}
	}

	ft.bestFinalizedNumber = &number

	if changed {
		return FinalizationResultChanged[V]{nil}, nil
	}
	return FinalizationResultUnchanged{}, nil
}

// Finalize a node in the tree and all its ancestors. The given function
// isDescendentOf should return true if the second hash (target) is
// a descendent of the first hash (base).
func (ft *ForkTree[H, N, V]) FinalizeWithAncestors(
	hash H,
	number N,
	isDescendentOf func(a, b H) (bool, error),
) (FinalizationResult, error) {
	if ft.bestFinalizedNumber != nil {
		if number <= *ft.bestFinalizedNumber {
			return nil, ErrRevert
		}
	}

	// check if one of the current roots is being finalized
	root := ft.FinalizeRoot(hash)
	if root != nil {
		return FinalizationResultChanged[V]{root}, nil
	}

	// we need to:
	// 1) remove all roots that are not ancestors AND not descendants of finalized block;
	// 2) if node is descendant - just leave it;
	// 3) if node is ancestor - 'open it'
	var (
		changed bool = false
		idx     int  = 0
	)
	for idx != len(ft.roots) {
		root := ft.roots[idx]
		isFinalized := root.Hash == hash
		var isDescendant bool = false
		if !isFinalized && root.Number > number {
			var err error
			isDescendant, err = isDescendentOf(hash, root.Hash)
			if err != nil {
				return nil, err
			}
		}
		var isAncestor bool = false
		if !isFinalized && !isDescendant && root.Number < number {
			var err error
			isAncestor, err = isDescendentOf(root.Hash, hash)
			if err != nil {
				return nil, err
			}
		}

		// if we have met finalized root - open it and return
		if isFinalized {
			v := ft.FinalizeRootAt(idx)
			return FinalizationResultChanged[V]{&v}, nil
		}

		// if node is descendant of finalized block - just leave it as is
		if isDescendant {
			idx++
			continue
		}

		// if node is ancestor of finalized block - remove it and continue with children
		if isAncestor {
			root := ft.roots[idx]
			ft.roots = append(ft.roots[:idx], ft.roots[idx+1:]...)
			ft.roots = append(ft.roots, root.Children...)
			changed = true
			continue
		}

		// if node is neither ancestor, nor descendant of the finalized block - remove it
		ft.roots = append(ft.roots[:idx], ft.roots[idx+1:]...)
		changed = true
	}

	ft.bestFinalizedNumber = &number

	if changed {
		return FinalizationResultChanged[V]{nil}, nil
	}
	return FinalizationResultUnchanged{}, nil
}

// Checks if any node in the tree is finalized by either finalizing the
// node itself or a node's descendent that's not in the tree, guaranteeing
// that the node being finalized isn't a descendent of (or equal to) any of
// the node's children. Returns &true if the node being finalized is
// a root, &false if the node being finalized is not a root, and
// nil if no node in the tree is finalized. The given predicate is
// checked on the prospective finalized root and must pass for finalization
// to occur. The given function isDescendentOf should return true if
// the second hash (target) is a descendent of the first hash (base).
func (ft *ForkTree[H, N, V]) FinalizesAnyWithDescendentIf(
	hash H,
	number N,
	isDescendentOf func(a, b H) (bool, error),
	predicate func(V) bool,
) (*bool, error) {
	if ft.bestFinalizedNumber != nil {
		if number <= *ft.bestFinalizedNumber {
			b := false
			return &b, ErrRevert
		}
	}

	// check if the given hash is equal or a descendent of any node in the
	// tree, if we find a valid node that passes the predicate then we must
	// ensure that we're not finalizing past any of its child nodes.
	for n := range ft.nodeIter() {
		if predicate(n.Data) {
			is, err := isDescendentOf(n.Hash, hash)
			if err != nil {
				var b bool = false
				return &b, err
			}
			if n.Hash == hash || is {
				for _, child := range n.Children {
					if child.Number <= number {
						if child.Hash == hash {
							return nil, ErrUnfinalizedAncestor
						}
						is, err := isDescendentOf(child.Hash, hash)
						if err != nil {
							return nil, err
						}
						if is {
							return nil, ErrUnfinalizedAncestor
						}
					}
				}

				b := slices.ContainsFunc(ft.roots, func(root node[H, N, V]) bool {
					return root.Hash == n.Hash
				})
				return &b, nil
			}
		}
	}

	return nil, nil
}

// Finalize a root in the tree by either finalizing the node itself or a
// node's descendent that's not in the tree, guaranteeing that the node
// being finalized isn't a descendent of (or equal to) any of the root's
// children. The given predicate is checked on the prospective finalized
// root and must pass for finalization to occur. The given function
// isDescendentOf should return true if the second hash (target) is a
// descendent of the first hash (base).
func (ft *ForkTree[H, N, V]) FinalizeWithDescendentIf(
	hash H,
	number N,
	isDescendentOf func(a, b H) (bool, error),
	predicate func(V) bool,
) (FinalizationResult, error) {
	if ft.bestFinalizedNumber != nil {
		if number <= *ft.bestFinalizedNumber {
			return nil, ErrRevert
		}
	}

	// check if the given hash is equal or a a descendent of any root, if we
	// find a valid root that passes the predicate then we must ensure that
	// we're not finalizing past any children node.
	var position int = -1
	for i, root := range ft.roots {
		if predicate(root.Data) {
			is, err := isDescendentOf(root.Hash, hash)
			if err != nil {
				return nil, err
			}
			if root.Hash == hash || is {
				for _, child := range root.Children {
					if child.Number <= number {
						if child.Hash == hash {
							return nil, ErrUnfinalizedAncestor
						}
						is, err := isDescendentOf(child.Hash, hash)
						if err != nil {
							return nil, err
						}
						if is {
							return nil, ErrUnfinalizedAncestor
						}
					}
				}

				position = i
				break
			}
		}
	}

	var nodeData *V
	if position >= 0 {
		node := ft.roots[position]
		ft.roots = node.Children
		ft.bestFinalizedNumber = &node.Number
		nodeData = &node.Data
	}

	// Retain only roots that are descendants of the finalized block (this
	// happens if the node has been properly finalized) or that are
	// ancestors (or equal) to the finalized block (in this case the node
	// wasn't finalized earlier presumably because the predicate didn't
	// pass).
	var changed bool = false
	roots := ft.roots
	ft.roots = nil

	for _, root := range roots {
		var retain bool
		if root.Number > number {
			var err error
			retain, err = isDescendentOf(hash, root.Hash)
			if err != nil {
				return nil, err
			}
		} else if root.Number == number && root.Hash == hash {
			retain = true
		} else {
			var err error
			retain, err = isDescendentOf(root.Hash, hash)
			if err != nil {
				return nil, err
			}
		}

		if retain {
			ft.roots = append(ft.roots, root)
		} else {
			changed = true
		}
	}

	ft.bestFinalizedNumber = &number

	switch {
	case nodeData != nil:
		return FinalizationResultChanged[V]{nodeData}, nil
	case changed:
		return FinalizationResultChanged[V]{nil}, nil
	case !changed:
		return FinalizationResultUnchanged{}, nil
	default:
		panic("unreachable")
	}
}

// Remove from the tree some nodes (and their subtrees) using a filter predicate.
//
// The filter is called over tree nodes and returns a filter action:
// - Remove if the node and its subtree should be removed;
// - KeepNode if we should maintain the node and keep processing the tree.
// - KeepTree if we should maintain the node and its entire subtree.
//
// An iterator over all the pruned nodes is returned.
func (ft *ForkTree[H, N, V]) DrainFilter(
	filter func(H, N, V) FilterAction,
) iter.Seq[Node[H, N, V]] {
	// 	let mut removed = Vec::new();
	// 	let mut retained = Vec::new();
	removed := make([]node[H, N, V], 0)
	retained := make([]nodeParentIdx[H, N, V], 0)

	roots := ft.roots
	ft.roots = nil
	queue := make([]nodeParentIdx[H, N, V], 0, len(roots))
	for i := len(roots) - 1; i >= 0; i-- {
		queue = append(queue, nodeParentIdx[H, N, V]{parentIdx: math.MaxUint, node: roots[i]})
	}
	var nextQueue []nodeParentIdx[H, N, V]

	for len(queue) > 0 {
		for len(queue) > 0 {
			popped := queue[0]
			queue = queue[1:]
			parentIdx := popped.parentIdx
			n := popped.node
			filterAction := filter(n.Hash, n.Number, n.Data)
			switch filterAction {
			case FilterActionKeepNode:
				nodeIdx := len(retained)
				children := n.Children
				n.Children = nil
				retained = append(retained, nodeParentIdx[H, N, V]{parentIdx: parentIdx, node: n})
				for i := len(children) - 1; i >= 0; i-- {
					nextQueue = append(nextQueue, nodeParentIdx[H, N, V]{parentIdx: uint(nodeIdx), node: children[i]})
				}
			case FilterActionKeepTree:
				retained = append(retained, nodeParentIdx[H, N, V]{parentIdx: parentIdx, node: n})
			case FilterActionRemove:
				removed = append(removed, n)
			default:
				panic("unreachable")
			}
		}

		temp := queue
		queue = nextQueue
		nextQueue = temp
	}

	for len(retained) > 0 {
		popped := retained[len(retained)-1]
		retained = retained[:len(retained)-1]
		parentIdx := popped.parentIdx
		n := popped.node
		if parentIdx == math.MaxUint {
			ft.roots = append(ft.roots, n)
		} else {
			retained[parentIdx].node.Children = append(retained[parentIdx].node.Children, n)
		}
	}

	if len(removed) > 0 {
		ft.Rebalance()
	}

	ri := removedIterator[H, N, V]{stack: removed}
	return ri.iter()
}

type forkTreeIterator[H comparable, N constraints.Integer, V any] struct {
	stack []node[H, N, V]
}

func (fti *forkTreeIterator[H, N, V]) next() *node[H, N, V] {
	if len(fti.stack) == 0 {
		return nil
	}
	node := fti.stack[len(fti.stack)-1]
	fti.stack = fti.stack[:len(fti.stack)-1]
	// child nodes are stored ordered by max branch height (decreasing),
	// we want to keep this ordering while iterating but since we're
	// using a stack for iterator state we need to reverse it.
	children := slices.Clone(node.Children)
	slices.Reverse(children)
	fti.stack = append(fti.stack, children...)
	return &node
}

func (fti *forkTreeIterator[H, N, V]) iter() iter.Seq[node[H, N, V]] {
	return func(yield func(node[H, N, V]) bool) {
		for {
			item := fti.next()
			if item == nil {
				return
			}
			if !yield(*item) {
				return
			}
		}
	}
}

type removedIterator[H comparable, N constraints.Integer, V any] struct {
	// stack: Vec<Node<H, N, V>>,
	stack []node[H, N, V]
}

func (ri *removedIterator[H, N, V]) next() *Node[H, N, V] {
	if len(ri.stack) == 0 {
		return nil
	}
	node := ri.stack[len(ri.stack)-1]
	ri.stack = ri.stack[:len(ri.stack)-1]
	// child nodes are stored ordered by max branch height (decreasing),
	// we want to keep this ordering while iterating but since we're
	// using a stack for iterator state we need to reverse it.
	children := node.Children
	node.Children = nil

	slices.Reverse(children)
	ri.stack = append(ri.stack, children...)
	return &Node[H, N, V]{
		Hash:   node.Hash,
		Number: node.Number,
		Data:   node.Data,
	}
}

func (ri *removedIterator[H, N, V]) iter() iter.Seq[Node[H, N, V]] {
	return func(yield func(Node[H, N, V]) bool) {
		for {
			item := ri.next()
			if item == nil {
				return
			}
			if !yield(*item) {
				return
			}
		}
	}
}
