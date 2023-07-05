// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
<<<<<<< HEAD
	"math/big"
=======
>>>>>>> development
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBlockTreeFromNode(root *node) *BlockTree {
	return &BlockTree{
		root:   root,
		leaves: newLeafMap(root),
	}
}

func Test_NewBlockTreeFromNode(t *testing.T) {
	var bt *BlockTree
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(t, testHeader, 5)
		if len(branches) > 0 && len(bt.getNode(branches[0].hash).children) > 0 {
			break
		}
	}

	testNode := bt.getNode(branches[0].hash).children[0]
	leaves := testNode.getLeaves(nil)

	newBt := newBlockTreeFromNode(testNode)
	require.ElementsMatch(t, leaves, newBt.leaves.nodes())
}

func Test_BlockTree_GetBlock(t *testing.T) {
	bt, hashes := createFlatTree(t, 2)

	n := bt.getNode(hashes[2])
	if n == nil {
		t.Fatal("node is nil")
	}

	if !bytes.Equal(hashes[2][:], n.hash[:]) {
		t.Fatalf("Fail: got %x expected %x", n.hash, hashes[2])
	}

}

func Test_BlockTree_AddBlock(t *testing.T) {
	bt, hashes := createFlatTree(t, 1)

	header := &types.Header{
		ParentHash: hashes[1],
		Number:     2,
		Digest:     createPrimaryBABEDigest(t),
	}

	hash := header.Hash()
	err := bt.AddBlock(header, time.Unix(0, 0))
	require.NoError(t, err)

	node := bt.getNode(hash)

	if n, err := bt.leaves.load(node.hash); n == nil || err != nil {
		t.Errorf("expected %x to be a leaf", n.hash)
	}

	oldHash := common.Hash{0x01}

	if n, err := bt.leaves.load(oldHash); n != nil || err == nil {
		t.Errorf("expected %x to no longer be a leaf", oldHash)
	}
}

func Test_Node_isDecendantOf(t *testing.T) {
	// Create tree with number 4 (with 4 nodes)
	bt, hashes := createFlatTree(t, 4)

	// Check leaf is descendant of root
	leaf := bt.getNode(hashes[3])
	if !leaf.isDescendantOf(bt.root) {
		t.Error("failed to verify leaf is descendant of root")
	}

	// Verify the inverse relationship does not hold
	if bt.root.isDescendantOf(leaf) {
		t.Error("root should not be descendant of anything")
	}
}

func Test_BlockTree_Best_AllPrimary(t *testing.T) {
	arrivalTime := int64(256)
	var expected Hash

	bt, _ := createTestBlockTree(t, testHeader, 8)

	var deepest uint

	for leaf, node := range bt.leaves.toMap() {
		node.arrivalTime = time.Unix(arrivalTime, 0)
		arrivalTime--
		if node.number >= deepest {
			deepest = node.number
			expected = leaf
		}

		t.Logf("leaf=%s number=%d arrivalTime=%s", leaf, node.number, node.arrivalTime)
	}

	require.Equal(t, expected, bt.best().hash)
}

func Test_BlockTree_GetNode(t *testing.T) {
	bt, branches := createTestBlockTree(t, testHeader, 16)

	for _, branch := range branches {
		header := &types.Header{
			ParentHash: branch.hash,
			Number:     branch.number + 1,
			StateRoot:  Hash{0x2},
			Digest:     createPrimaryBABEDigest(t),
		}

		err := bt.AddBlock(header, time.Unix(0, 0))
		require.NoError(t, err)
	}

	block := bt.getNode(branches[0].hash)
	require.NotNil(t, block)
}

func Test_BlockTree_GetAllBlocksAtNumber(t *testing.T) {
	bt, _ := createTestBlockTree(t, testHeader, 8)
	hashes := bt.root.getNodesWithNumber(10, []common.Hash{})

	require.Empty(t, hashes)

	// create one-path tree
	const btNumber uint = 8
	const desiredNumber uint = 6
	bt, btHashes := createFlatTree(t, btNumber)

	expected := []common.Hash{btHashes[desiredNumber]}

	// add branch
	previousHash := btHashes[4]

	for i := uint(4); i <= btNumber; i++ {
		header := &types.Header{
			ParentHash: previousHash,
			StateRoot:  common.Hash{0x99},
			Number:     i + 1,
			Digest:     createPrimaryBABEDigest(t),
		}

		hash := header.Hash()
		err := bt.AddBlock(header, time.Unix(0, 0))
		require.NoError(t, err)
		previousHash = hash

		if i == desiredNumber-1 {
			expected = append(expected, hash)
		}
	}

	// add another branch
	previousHash = btHashes[2]

	for i := uint(2); i <= btNumber; i++ {
		header := &types.Header{
			ParentHash: previousHash,
			StateRoot:  common.Hash{0x88},
			Number:     i + 1,
			Digest:     createPrimaryBABEDigest(t),
		}

		hash := header.Hash()
		err := bt.AddBlock(header, time.Unix(0, 0))
		require.NoError(t, err)
		previousHash = hash

		if i == desiredNumber-1 {
			expected = append(expected, hash)
		}
	}

	hashes = bt.root.getNodesWithNumber(desiredNumber, []common.Hash{})
	require.Equal(t, expected, hashes)
}

func Test_BlockTree_GetAllDescendants(t *testing.T) {
	t.Parallel()

	// Create tree with number 4 (with 4 nodes)
	bt, hashes := createFlatTree(t, 4)

	descendants, err := bt.GetAllDescendants(bt.root.hash)
	require.NoError(t, err)
	require.Equal(t, hashes, descendants)
}

func Test_BlockTree_IsDecendantOf(t *testing.T) {
	// Create tree with number 4 (with 4 nodes)
	bt, hashes := createFlatTree(t, 4)

	isDescendant, err := bt.IsDescendantOf(bt.root.hash, hashes[3])
	require.NoError(t, err)
	require.True(t, isDescendant)

	isDescendant, err = bt.IsDescendantOf(hashes[3], bt.root.hash)
	require.NoError(t, err)
	require.False(t, isDescendant)
}

func Test_lowestCommonAncestor(t *testing.T) {
	t.Parallel()
	root := &node{
		hash:   common.Hash{0},
		number: 0,
	}

	children := []*node{
		{
			hash:      common.Hash{1},
			parent:    root,
			isPrimary: true,
			number:    1,
		},
		{
			hash:      common.Hash{2},
			parent:    root,
			isPrimary: false,
			number:    1,
		},
	}

	childrenChildren := []*node{
		{
			hash:      common.Hash{3},
			parent:    children[0],
			isPrimary: true,
			number:    2,
		},
		{
			hash:      common.Hash{4},
			parent:    children[1],
			isPrimary: false,
			number:    2,
		},
	}
	finalChild := []*node{
		{
			hash:      common.Hash{5},
			parent:    childrenChildren[1],
			isPrimary: true,
			number:    3,
		},
	}

	type args struct {
		nodeA *node
		nodeB *node
	}
	tests := []struct {
		name   string
		args   args
		expErr error
		expRes Hash
	}{
		{
			name: "child_and_root",
			args: args{
				nodeA: children[1],
				nodeB: root,
			},
			expRes: root.hash,
		},
		{
			name: "same_node",
			args: args{
				nodeA: children[1],
				nodeB: children[1],
			},
			expRes: children[1].hash,
		},
		{
			name: "siblings",
			args: args{
				nodeA: children[0],
				nodeB: children[1],
			},
			expRes: root.hash,
		},
		{
			name: "child_and_its_child",
			args: args{
				nodeA: children[0],
				nodeB: childrenChildren[0],
			},
			expRes: children[0].hash,
		},
		{
			name: "root_and_grandchild",
			args: args{
				nodeA: root,
				nodeB: childrenChildren[0],
			},
			expRes: root.hash,
		},
		{
			name: "grandchild_and_its_siblings_child",
			args: args{
				nodeA: finalChild[0],
				nodeB: childrenChildren[0],
			},
			expRes: root.hash,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ancestor := lowestCommonAncestor(tt.args.nodeA, tt.args.nodeB)
			require.Equal(t, tt.expRes, ancestor)
		})
	}
}

func Test_BlockTree_LowestCommonAncestor(t *testing.T) {
	var bt *BlockTree
	var leaves []common.Hash
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(t, testHeader, 8)
		leaves = bt.Leaves()
		if len(leaves) == 2 {
			break
		}
	}

	expected := branches[0].hash

	a := leaves[0]
	b := leaves[1]

	p, err := bt.LowestCommonAncestor(a, b)
	require.NoError(t, err)
	require.Equal(t, expected, p)
}

func Test_BlockTree_LowestCommonAncestor_SameNode(t *testing.T) {
	bt, _ := createTestBlockTree(t, testHeader, 8)
	leaves := bt.Leaves()

	a := leaves[0]

	p, err := bt.LowestCommonAncestor(a, a)
	require.NoError(t, err)
	require.Equal(t, a, p)
}

func Test_BlockTree_LowestCommonAncestor_SameChain(t *testing.T) {
	bt, _ := createTestBlockTree(t, testHeader, 8)
	leaves := bt.Leaves()

	a := leaves[0]
	b := bt.getNode(a).parent.hash

	// b is a's parent, so their highest common Ancestor is b.
	p, err := bt.LowestCommonAncestor(a, b)
	require.NoError(t, err)
	require.Equal(t, b, p)
}

func buildLinearBlockTree(t *testing.T, amount int) *BlockTree {
	t.Helper()

	blockTree := &BlockTree{
		leaves:   newEmptyLeafMap(),
		runtimes: newHashToRuntime(),
	}

	rootNode := &node{
		hash:   common.MustHexToHash("0x00"),
		number: 0,
	}

	blockTree.root = rootNode

	parentNode := rootNode
	for idx := 1; idx < amount; idx++ {
		newNode := &node{
			parent: parentNode,
			hash:   common.MustHexToHash(fmt.Sprintf("0x0%d", idx)),
			number: uint(idx),
		}

		parentNode.addChild(newNode)
		parentNode = newNode
	}

	// parentNode node here will be the latest block in the tree
	// that means it will be the leaf as well
	blockTree.leaves.store(parentNode.hash, parentNode)
	return blockTree
}

func appendRuntimeToHash(t *testing.T, blockTree *BlockTree,
	hash common.Hash, runtimeInstance runtime.Instance) {
	t.Helper()

	blockTree.runtimes.set(hash, runtimeInstance)
}

func appendForksAt(t *testing.T, blockTree *BlockTree, forkAt common.Hash, forkHashes ...common.Hash) {
	t.Helper()

	parentNode := blockTree.getNode(forkAt)
	require.NotNil(t, parentNode)

	for idx, hash := range forkHashes {
		newNode := &node{
			parent: parentNode,
			hash:   hash,
			number: uint(100 + idx),
		}
		parentNode.addChild(newNode)
		parentNode = newNode
	}

	// parentNode node here will be the latest block in the tree
	// that means it will be the leaf as well
	blockTree.leaves.store(parentNode.hash, parentNode)

}

func Test_BlockTree_GetBlockRuntime(t *testing.T) {
	// {0x00} -> {0x01} -> {0x02} -> {0x03}
	//                  -> {0x04} -> {0x05}
	//							  -> {0x06}
	blockTree := buildLinearBlockTree(t, 4)

	appendForksAt(t, blockTree, common.MustHexToHash("0x01"),
		common.MustHexToHash("0x04"),
		common.MustHexToHash("0x05"))

	appendForksAt(t, blockTree, common.MustHexToHash("0x04"),
		common.MustHexToHash("0x06"))

	rootRuntime := NewMockInstance(nil)
	lastCanonicalRuntime := NewMockInstance(nil)
	forkedRuntime := NewMockInstance(nil)

	appendRuntimeToHash(t, blockTree, common.MustHexToHash("0x00"), rootRuntime)
	appendRuntimeToHash(t, blockTree, common.MustHexToHash("0x03"), lastCanonicalRuntime)
	appendRuntimeToHash(t, blockTree, common.MustHexToHash("0x04"), forkedRuntime)
	appendRuntimeToHash(t, blockTree, common.MustHexToHash("0x05"), lastCanonicalRuntime)

	// Even though we have only 3 runtimes (rootRuntime, lastCanonicalRuntime and forkedRuntime)
	// the lastCanonicalRuntime happens in different forks, it is in the block `0x03` and in block
	// `0x05` and both blocks don't have any relashionship that justifies the usage of one instance
	const totalRuntimesInMemory = 4
	require.Equal(t, totalRuntimesInMemory, len(blockTree.runtimes.mapping))

	testCases := []struct {
		hashInput       common.Hash
		expectedRuntime runtime.Instance
	}{
		{common.MustHexToHash("0x06"), forkedRuntime},
		{common.MustHexToHash("0x05"), lastCanonicalRuntime},
		{common.MustHexToHash("0x04"), forkedRuntime},
		{common.MustHexToHash("0x03"), lastCanonicalRuntime},
		{common.MustHexToHash("0x02"), rootRuntime},
		{common.MustHexToHash("0x00"), rootRuntime},
	}

	for _, tt := range testCases {
		givenRuntime, err := blockTree.GetBlockRuntime(tt.hashInput)
		require.NoError(t, err)
		if tt.expectedRuntime != givenRuntime {
			t.Errorf("exepected %v. got %v", tt.expectedRuntime, givenRuntime)
			return
		}
	}
}

func Test_BlockTree_Prune(t *testing.T) {
	t.Parallel()

	t.Run("finalised_hash_is_root_hash", func(t *testing.T) {
		t.Parallel()

		blockTree := &BlockTree{
			root: &node{
				hash: Hash{1},
			},
		}

		pruned := blockTree.Prune(Hash{1})

		assert.Empty(t, pruned)
	})

	t.Run("node_not_found", func(t *testing.T) {
		t.Parallel()

		blockTree := &BlockTree{
			root:   &node{},
			leaves: newEmptyLeafMap(),
		}

		pruned := blockTree.Prune(Hash{1})

		assert.Empty(t, pruned)
	})

	t.Run("nothing_to_prune", func(t *testing.T) {
		t.Parallel()

		rootNode := &node{
			hash:   common.Hash{1},
			number: 0,
		}
		blockTree := &BlockTree{
			root:     rootNode,
			leaves:   newEmptyLeafMap(),
			runtimes: newHashToRuntime(),
		}

		ctrl := gomock.NewController(t)
		runtimeInstanceToBePrunned := NewMockInstance(ctrl)
		runtimeInstanceToBePrunned.EXPECT().Stop()
		blockTree.runtimes.set(common.Hash{1}, runtimeInstanceToBePrunned)

		// {1} -> {2}
		parent := rootNode
		newNode := &node{
			parent: parent,
			hash:   common.Hash{2},
			number: 1,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)

		expectedRuntimeInstance := NewMockInstance(nil)
		blockTree.runtimes.set(common.Hash{2}, expectedRuntimeInstance)

		pruned := blockTree.Prune(common.Hash{2})
		assert.Empty(t, pruned)

		expectedHashToRuntime := &hashToRuntime{
			mapping: map[common.Hash]runtime.Instance{
				{2}: expectedRuntimeInstance,
			},
		}
		assert.Equal(t, expectedHashToRuntime, blockTree.runtimes)
	})

	t.Run("prune_canonical_runtimes", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		rootNode := &node{
			hash:   common.Hash{1},
			number: 0,
		}
		rootRuntime := NewMockInstance(ctrl)
		blockTree := &BlockTree{
			root:     rootNode,
			leaves:   newEmptyLeafMap(),
			runtimes: newHashToRuntime(),
		}
		blockTree.runtimes.set(common.Hash{1}, rootRuntime)

		// {1} -> {2}
		parent := rootNode
		newNode := &node{
			parent: parent,
			hash:   common.Hash{2},
			number: 1,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)
		leafRuntime := NewMockInstance(ctrl)
		blockTree.runtimes.set(common.Hash{2}, leafRuntime)

		// Previous runtime is pruned
		rootRuntime.EXPECT().Stop()

		pruned := blockTree.Prune(common.Hash{2})
		assert.Empty(t, pruned)

		expectedHashToRuntime := &hashToRuntime{
			mapping: map[common.Hash]runtime.Instance{
				{2}: leafRuntime,
			},
		}
		assert.Equal(t, expectedHashToRuntime, blockTree.runtimes)
	})

	t.Run("prune_fork", func(t *testing.T) {
		t.Parallel()

		rootNode := &node{
			hash:   common.Hash{1},
			number: 0,
		}

		blockTree := &BlockTree{
			root:     rootNode,
			leaves:   newEmptyLeafMap(),
			runtimes: newHashToRuntime(),
		}

		rootRuntime := NewMockInstance(nil)
		blockTree.runtimes.set(common.Hash{1}, rootRuntime)

		// {1} -> {2}
		// we don't need to add a runtime to node number 2 since
		parent := rootNode
		newNode := &node{
			parent: parent,
			hash:   common.Hash{2},
			number: 1,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)

		// {1} -> {2}
		//     -> {3}
		parent = rootNode
		newNode = &node{
			parent: parent,
			hash:   common.Hash{3},
			number: 1,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)

		ctrl := gomock.NewController(t)
		runtimeToBePrunned := NewMockInstance(ctrl)
		runtimeToBePrunned.EXPECT().Stop()

		blockTree.runtimes.set(common.Hash{3}, runtimeToBePrunned)

		// expect that node number 3 to be prunned with its runtime
		pruned := blockTree.Prune(common.Hash{2})
		assert.Equal(t, []common.Hash{{3}}, pruned)
		expectedHashToRuntime := &hashToRuntime{
			mapping: map[common.Hash]runtime.Instance{
				{2}: rootRuntime,
			},
		}
		assert.Equal(t, expectedHashToRuntime, blockTree.runtimes)
	})

	t.Run("complex_example", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		rootNode := &node{
			hash:   common.Hash{1},
			number: 100,
		}
		rootRuntime := NewMockInstance(ctrl)
		rootRuntime.EXPECT().Stop()

		blockTree := &BlockTree{
			root:   rootNode,
			leaves: newEmptyLeafMap(),
			runtimes: &hashToRuntime{
				mapping: map[common.Hash]runtime.Instance{
					{1}: rootRuntime,
				},
			},
		}

		// {1} -> rootRuntime
		// {1} -> {2}
		parent := rootNode
		newNode := &node{
			parent: parent,
			hash:   common.Hash{2},
			number: 101,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)

		// {1} -> rootRuntime | {3} -> lastCanonicalRuntime
		// {1} -> {2} -> {3}
		parent = newNode
		newNode = &node{
			parent: parent,
			hash:   common.Hash{3},
			number: 102,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)
		lastCanonicalRuntime := NewMockInstance(ctrl)
		blockTree.runtimes.set(common.Hash{3}, lastCanonicalRuntime)

		// {1} -> rootRuntime | {3} -> lastCanonicalRuntime
		// {1} -> {2} -> {3} -> {4}
		parent = newNode
		newNode = &node{
			parent: parent,
			hash:   common.Hash{4},
			number: 103,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)

		// {1} -> rootRuntime | {3} -> lastCanonicalRuntime
		// {1} -> {2} -> {3} -> {4}
		//            -> {5}
		parent = blockTree.getNode(common.Hash{2})
		newNode = &node{
			parent: parent,
			hash:   common.Hash{5},
			number: 102,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)

		// {1} -> rootRuntime | {3} -> lastCanonicalRuntime
		// {1} -> rootRuntime | {6} -> lastCanonicalRuntime
		// {1} -> {2} -> {3} -> {4}
		//            -> {5} -> {6}
		parent = blockTree.getNode(common.Hash{5})
		newNode = &node{
			parent: parent,
			hash:   common.Hash{6},
			number: 103,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)
		blockTree.runtimes.set(common.Hash{6}, lastCanonicalRuntime)

		// {1} -> rootRuntime | {3} -> lastCanonicalRuntime
		// {1} -> rootRuntime | {6} -> lastCanonicalRuntime | {7} -> forkedRuntime
		// {1} -> {2} -> {3} -> {4}
		//            -> {5} -> {6}
		//						-> {7}
		parent = blockTree.getNode(common.Hash{5})
		newNode = &node{
			parent: parent,
			hash:   common.Hash{7},
			number: 102,
		}
		parent.addChild(newNode)
		blockTree.leaves.replace(parent, newNode)
		forkedRuntime := NewMockInstance(ctrl)
		forkedRuntime.EXPECT().Stop()
		blockTree.runtimes.set(common.Hash{7}, forkedRuntime)

		pruned := blockTree.Prune(common.Hash{4})
		assert.Equal(t, []common.Hash{{5}, {6}, {7}}, pruned)

		expectedHashToRuntime := &hashToRuntime{
			mapping: map[common.Hash]runtime.Instance{
				{4}: lastCanonicalRuntime,
			},
		}
		assert.Equal(t, expectedHashToRuntime, blockTree.runtimes)
	})
}

func Test_BlockTree_GetHashByNumber(t *testing.T) {
	bt, _ := createTestBlockTree(t, testHeader, 8)
	best := bt.BestBlockHash()
	bn := bt.getNode(best)

	for i := uint(0); i < bn.number; i++ {
		hash, err := bt.GetHashByNumber(i)
		require.NoError(t, err)
		require.Equal(t, i, bt.getNode(hash).number)
		desc, err := bt.IsDescendantOf(hash, best)
		require.NoError(t, err)
		require.True(t, desc, fmt.Sprintf("index %d failed, got hash=%s", i, hash))
	}

	_, err := bt.GetHashByNumber(bn.number + 1)
	require.Error(t, err)
}

func Test_BlockTree_BestBlockHash_AllChainsEqual(t *testing.T) {
	bt := NewBlockTreeFromRoot(testHeader)
	previousHash := testHeader.Hash()

	var branches []testBranch

	const fixedArrivalTime = 99
	const depth uint = 4

	// create a base tree with a fixed amount of blocks
	// and all block with the same arrival time

	/**
	base tree and nodes representation, all with the same arrival time and all
	the leaves has the same number (8) the numbers in the right represents the order
	the nodes are inserted into the blocktree.

	a -> b -> c -> d -> e -> f -> g -> h (1)
		|    |    |    |    |    |> h (7)
		|    |    |    |    |> g -> h (6)
		|    |    |    |> f -> g -> h (5)
		|    |    |> e -> f -> g -> h (4)
		|    |> d -> e -> f -> g -> h (3)
		|> c -> d -> e -> f -> g -> h (2)
	**/

	for i := uint(1); i <= depth; i++ {
		header := &types.Header{
			ParentHash: previousHash,
			Number:     i,
			Digest:     createPrimaryBABEDigest(t),
		}

		hash := header.Hash()

		err := bt.AddBlock(header, time.Unix(0, fixedArrivalTime))
		require.NoError(t, err)

		previousHash = hash

		// the last block on the base tree should not generates a branch
		if i < depth {
			branches = append(branches, testBranch{
				hash:   hash,
				number: bt.getNode(hash).number,
			})
		}
	}

	// create all the branch nodes with the same arrival time
	for _, branch := range branches {
		previousHash = branch.hash

		for i := branch.number; i < depth; i++ {
			header := &types.Header{
				ParentHash: previousHash,
				Number:     i + 1,
				StateRoot:  common.Hash{0x1},
				Digest:     createPrimaryBABEDigest(t),
			}

			hash := header.Hash()
			err := bt.AddBlock(header, time.Unix(0, fixedArrivalTime))
			require.NoError(t, err)

			previousHash = hash
		}
	}

	// check all leaves has the same number and timestamps
	leaves := bt.leaves.nodes()
	for idx := 0; idx < len(leaves)-2; idx++ {
		curr := leaves[idx]
		next := leaves[idx+1]

		require.Equal(t, curr.number, next.number)
		require.Equal(t, curr.arrivalTime, next.arrivalTime)
	}

	require.Len(t, leaves, int(depth))
	require.Contains(t, leaves, bt.best())

	// check that highest returned was one with lowest hash
	expected := leaves[0].hash
	for _, leaf := range leaves {
		if bytes.Compare(leaf.hash[:], expected[:]) < 0 {
			expected = leaf.hash
		}
	}

	require.Equal(t, bt.best().hash, expected)

	// adding a new node with a greater number should update the best block
	header := &types.Header{
		ParentHash: previousHash,
		Number:     bt.best().number + 1,
		StateRoot:  common.Hash{0x1},
		Digest:     createPrimaryBABEDigest(t),
	}

	hash := header.Hash()
	err := bt.AddBlock(header, time.Unix(0, fixedArrivalTime))
	require.NoError(t, err)
	require.Equal(t, hash, bt.best().hash)
}

func Test_BlockTree_DeepCopy(t *testing.T) {
	bt, _ := createFlatTree(t, 8)

	btCopy := bt.DeepCopy()
	equalNodeValue(t, bt.root, btCopy.root)
	equalLeaves(t, bt.leaves, btCopy.leaves)

	btCopy.root = &node{}
	require.NotEqual(t, bt.root, btCopy.root)
}

func equalNodeValue(t *testing.T, nd *node, ndCopy *node) {
	t.Helper()
	assert.Equal(t, nd.hash, ndCopy.hash, "hash not equal")
	assert.Equal(t, nd.number, ndCopy.number, "number not equal")
	assert.Equal(t, nd.arrivalTime, ndCopy.arrivalTime, "arrivalTime not equal")
	for i, child := range nd.children {
		equalNodeValue(t, child, ndCopy.children[i])
	}
	if nd.parent != nil {
		assert.Equal(t, nd.parent.hash, ndCopy.parent.hash, "parent hash not equal")
		assert.Equal(t, nd.parent.arrivalTime, ndCopy.parent.arrivalTime, "parent arrival time not equal")
		assert.Equal(t, nd.parent.number, ndCopy.parent.number, "parent number not equal")
	} else {
		assert.Nil(t, ndCopy.parent, "parent not nil")
	}
}

func equalLeaves(t *testing.T, lm *leafMap, lmCopy *leafMap) {
	lmm := lm.toMap()
	lmCopyM := lmCopy.toMap()
	for key, val := range lmm {
		lmCopyVal := lmCopyM[key]
		equalNodeValue(t, val, lmCopyVal)
	}
}

func Test_BlockTree_best(t *testing.T) {
	// test basic case where two chains have different amount of primaries
	bt := NewEmptyBlockTree()
	bt.root = &node{
		hash: common.Hash{0},
	}

	bt.root.children = []*node{
		{
			hash:      common.Hash{1},
			parent:    bt.root,
			isPrimary: true,
		},
		{
			hash:      common.Hash{2},
			parent:    bt.root,
			isPrimary: false,
		},
	}

	bt.leaves = newEmptyLeafMap()
	bt.leaves.store(bt.root.children[0].hash, bt.root.children[0])
	bt.leaves.store(bt.root.children[1].hash, bt.root.children[1])
	require.Equal(t, bt.root.children[0].hash, bt.BestBlockHash())

	// test case where two chains have the same amount of primaries
	// and the head numbers are also equal
	// should pick the chain with the lowest arrival time or block hash
	bt = NewEmptyBlockTree()
	bt.root = &node{
		hash: common.Hash{0},
	}

	bt.root.children = []*node{
		{
			hash:      common.Hash{1},
			parent:    bt.root,
			number:    1,
			isPrimary: true,
		},
		{
			hash:      common.Hash{2},
			parent:    bt.root,
			isPrimary: false,
		},
	}

	bt.root.children[1].children = []*node{
		{
			hash:      common.Hash{3},
			parent:    bt.root.children[1],
			number:    1,
			isPrimary: true,
		},
	}

	bt.leaves = newEmptyLeafMap()
	bt.leaves.store(bt.root.children[0].hash, bt.root.children[0])
	bt.leaves.store(bt.root.children[1].children[0].hash, bt.root.children[1].children[0])
	require.Equal(t, bt.root.children[0].hash, bt.BestBlockHash())

	// test case where three chains have the same amount of primaries
	// and the head numbers are also equal
	// should pick the chain with the lowest arrival time or block hash
	bt = NewEmptyBlockTree()
	bt.root = &node{
		hash: common.Hash{0},
	}

	bt.root.children = []*node{
		{
			hash:      common.Hash{3},
			parent:    bt.root,
			number:    1,
			isPrimary: true,
		},
		{
			hash:      common.Hash{2},
			parent:    bt.root,
			isPrimary: false,
		},
		{
			hash:      common.Hash{1},
			parent:    bt.root,
			number:    1,
			isPrimary: true,
		},
	}

	bt.leaves = newEmptyLeafMap()
	bt.leaves.store(bt.root.children[0].hash, bt.root.children[0])
	bt.leaves.store(bt.root.children[1].hash, bt.root.children[1])
	bt.leaves.store(bt.root.children[2].hash, bt.root.children[2])
	require.Equal(t, bt.root.children[2].hash, bt.BestBlockHash())
}

func BenchmarkBlockTreeSubBlockchain(b *testing.B) {
	testInputs := []struct {
		input int
	}{
		{input: 100},
		{input: 1000},
		{input: 10000},
	}

	for _, tt := range testInputs {
		bt, expectedHashes := createFlatTree(b, uint(tt.input))

		firstHash := expectedHashes[0]
		endHash := expectedHashes[len(expectedHashes)-1]

		b.Run(fmt.Sprintf("input_len_%d", tt.input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bt.RangeInMemory(firstHash, endHash)
			}
		})
	}

}
