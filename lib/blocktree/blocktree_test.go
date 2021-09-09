// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package blocktree

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	database "github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

var zeroHash, _ = common.HexToHash("0x00")
var testHeader = &types.Header{
	ParentHash: zeroHash,
	Number:     big.NewInt(0),
	Digest:     types.NewDigest(),
}

func newBlockTreeFromNode(head *node, db database.Database) *BlockTree {
	return &BlockTree{
		head:   head,
		leaves: newLeafMap(head),
		db:     db,
	}
}

func createFlatTree(t *testing.T, depth int) (*BlockTree, []common.Hash) {
	bt := NewBlockTreeFromRoot(testHeader, nil)
	require.NotNil(t, bt)

	previousHash := bt.head.hash

	hashes := []common.Hash{bt.head.hash}
	for i := 1; i <= depth; i++ {
		header := &types.Header{
			ParentHash: previousHash,
			Number:     big.NewInt(int64(i)),
			Digest:     types.NewDigest(),
		}

		hash := header.Hash()
		hashes = append(hashes, hash)

		err := bt.AddBlock(header, 0)
		require.Nil(t, err)
		previousHash = hash
	}

	return bt, hashes
}

func TestNewBlockTreeFromNode(t *testing.T) {
	var bt *BlockTree
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(testHeader, 5, nil)
		if len(branches) > 0 && len(bt.getNode(branches[0].hash).children) > 0 {
			break
		}
	}

	testNode := bt.getNode(branches[0].hash).children[0]
	leaves := testNode.getLeaves(nil)

	newBt := newBlockTreeFromNode(testNode, nil)
	require.ElementsMatch(t, leaves, newBt.leaves.nodes())
}

func TestBlockTree_GetBlock(t *testing.T) {
	bt, hashes := createFlatTree(t, 2)

	n := bt.getNode(hashes[2])
	if n == nil {
		t.Fatal("node is nil")
	}

	if !bytes.Equal(hashes[2][:], n.hash[:]) {
		t.Fatalf("Fail: got %x expected %x", n.hash, hashes[2])
	}

}

func TestBlockTree_AddBlock(t *testing.T) {
	bt, hashes := createFlatTree(t, 1)

	header := &types.Header{
		ParentHash: hashes[1],
		Number:     big.NewInt(1),
	}

	hash := header.Hash()
	err := bt.AddBlock(header, 0)
	require.Nil(t, err)

	node := bt.getNode(hash)

	if n, err := bt.leaves.load(node.hash); n == nil || err != nil {
		t.Errorf("expected %x to be a leaf", n.hash)
	}

	oldHash := common.Hash{0x01}

	if n, err := bt.leaves.load(oldHash); n != nil || err == nil {
		t.Errorf("expected %x to no longer be a leaf", oldHash)
	}
}

func TestNode_isDecendantOf(t *testing.T) {
	// Create tree with depth 4 (with 4 nodes)
	bt, hashes := createFlatTree(t, 4)

	// Check leaf is descendant of root
	leaf := bt.getNode(hashes[3])
	if !leaf.isDescendantOf(bt.head) {
		t.Error("failed to verify leaf is descendant of root")
	}

	// Verify the inverse relationship does not hold
	if bt.head.isDescendantOf(leaf) {
		t.Error("root should not be descendant of anything")
	}
}

func TestBlockTree_LongestPath(t *testing.T) {
	bt, hashes := createFlatTree(t, 3)

	// Insert a block to create a competing path
	header := &types.Header{
		ParentHash: hashes[0],
		Number:     big.NewInt(1),
	}

	header.Hash()
	err := bt.AddBlock(header, 0)
	require.NotNil(t, err)

	longestPath := bt.longestPath()

	for i, n := range longestPath {
		if n.hash != hashes[i] {
			t.Errorf("expected Hash: 0x%X got: 0x%X\n", hashes[i], n.hash)
		}
	}
}

func TestBlockTree_Subchain(t *testing.T) {
	bt, hashes := createFlatTree(t, 4)
	expectedPath := hashes[1:]

	// Insert a block to create a competing path
	extraBlock := &types.Header{
		ParentHash: hashes[0],
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
	}

	extraBlock.Hash()
	err := bt.AddBlock(extraBlock, 0)
	require.NotNil(t, err)

	subChain, err := bt.subChain(hashes[1], hashes[3])
	if err != nil {
		t.Fatal(err)
	}

	for i, n := range subChain {
		if n.hash != expectedPath[i] {
			t.Errorf("expected Hash: 0x%X got: 0x%X\n", expectedPath[i], n.hash)
		}
	}
}

func TestBlockTree_DeepestLeaf(t *testing.T) {
	arrivalTime := uint64(256)
	var expected Hash

	bt, _ := createTestBlockTree(testHeader, 8, nil)

	deepest := big.NewInt(0)

	for leaf, node := range bt.leaves.toMap() {
		node.arrivalTime = arrivalTime
		arrivalTime--
		if node.depth.Cmp(deepest) >= 0 {
			deepest = node.depth
			expected = leaf
		}

		t.Logf("leaf=%s depth=%d arrivalTime=%d", leaf, node.depth, node.arrivalTime)
	}

	deepestLeaf := bt.deepestLeaf()
	if deepestLeaf.hash != expected {
		t.Fatalf("Fail: got %s expected %s", deepestLeaf.hash, expected)
	}
}

func TestBlockTree_GetNode(t *testing.T) {
	bt, branches := createTestBlockTree(testHeader, 16, nil)

	for _, branch := range branches {
		header := &types.Header{
			ParentHash: branch.hash,
			Number:     branch.depth,
			StateRoot:  Hash{0x1},
		}

		err := bt.AddBlock(header, 0)
		require.Nil(t, err)
	}
}

func TestBlockTree_GetNodeCache(t *testing.T) {
	bt, branches := createTestBlockTree(testHeader, 16, nil)

	for _, branch := range branches {
		header := &types.Header{
			ParentHash: branch.hash,
			Number:     branch.depth,
			StateRoot:  Hash{0x1},
		}

		err := bt.AddBlock(header, 0)
		require.Nil(t, err)
	}

	block := bt.getNode(branches[0].hash)

	cachedBlock, ok := bt.nodeCache[block.hash]

	require.True(t, len(bt.nodeCache) > 0)
	require.True(t, ok)
	require.NotNil(t, cachedBlock)
	require.Equal(t, cachedBlock, block)

}

func TestBlockTree_GetAllBlocksAtDepth(t *testing.T) {
	bt, _ := createTestBlockTree(testHeader, 8, nil)
	hashes := bt.head.getNodesWithDepth(big.NewInt(10), []common.Hash{})

	expected := []common.Hash{}

	if !reflect.DeepEqual(hashes, expected) {
		t.Fatalf("Fail: expected empty array")
	}

	// create one-path tree
	btDepth := 8
	desiredDepth := 6
	bt, btHashes := createFlatTree(t, btDepth)

	expected = []common.Hash{btHashes[desiredDepth]}

	// add branch
	previousHash := btHashes[4]


	for i := 4; i <= btDepth; i++ {
		digest := types.NewDigest()
		digest.Add(types.ConsensusDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		})
		header := &types.Header{
			ParentHash: previousHash,
			Number:     big.NewInt(int64(i)),
			Digest:     digest,
		}

		hash := header.Hash()
		bt.AddBlock(header, 0)
		previousHash = hash

		if i == desiredDepth-1 {
			expected = append(expected, hash)
		}
	}

	// add another branch
	previousHash = btHashes[2]

	for i := 2; i <= btDepth; i++ {
		digest := types.NewDigest()
		digest.Add(types.SealDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		})
		header := &types.Header{
			ParentHash: previousHash,
			Number:     big.NewInt(int64(i)),
			Digest:     digest,
		}

		hash := header.Hash()
		bt.AddBlock(header, 0)
		previousHash = hash

		if i == desiredDepth-1 {
			expected = append(expected, hash)
		}
	}

	hashes = bt.head.getNodesWithDepth(big.NewInt(int64(desiredDepth)), []common.Hash{})

	if !reflect.DeepEqual(hashes, expected) {
		t.Fatalf("Fail: did not get all expected hashes got %v expected %v", hashes, expected)
	}
}

func TestBlockTree_IsDecendantOf(t *testing.T) {
	// Create tree with depth 4 (with 4 nodes)
	bt, hashes := createFlatTree(t, 4)

	isDescendant, err := bt.IsDescendantOf(bt.head.hash, hashes[3])
	require.NoError(t, err)
	require.True(t, isDescendant)

	isDescendant, err = bt.IsDescendantOf(hashes[3], bt.head.hash)
	require.NoError(t, err)
	require.False(t, isDescendant)
}

func TestBlockTree_HighestCommonAncestor(t *testing.T) {
	var bt *BlockTree
	var leaves []common.Hash
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(testHeader, 8, nil)
		leaves = bt.Leaves()
		if len(leaves) == 2 {
			break
		}
	}

	expected := branches[0].hash

	a := leaves[0]
	b := leaves[1]

	p, err := bt.HighestCommonAncestor(a, b)
	require.NoError(t, err)
	require.Equal(t, expected, p)
}

func TestBlockTree_HighestCommonAncestor_SameNode(t *testing.T) {
	bt, _ := createTestBlockTree(testHeader, 8, nil)
	leaves := bt.Leaves()

	a := leaves[0]

	p, err := bt.HighestCommonAncestor(a, a)
	require.NoError(t, err)
	require.Equal(t, a, p)
}

func TestBlockTree_HighestCommonAncestor_SameChain(t *testing.T) {
	bt, _ := createTestBlockTree(testHeader, 8, nil)
	leaves := bt.Leaves()

	a := leaves[0]
	b := bt.getNode(a).parent.hash

	// b is a's parent, so their highest common Ancestor is b.
	p, err := bt.HighestCommonAncestor(a, b)
	require.NoError(t, err)
	require.Equal(t, b, p)
}

func TestBlockTree_Prune(t *testing.T) {
	var bt *BlockTree
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(testHeader, 5, nil)
		if len(branches) > 0 && len(bt.getNode(branches[0].hash).children) > 1 {
			break
		}
	}

	copy := bt.DeepCopy()

	// pick some block to finalise
	finalised := bt.head.children[0].children[0].children[0]
	pruned := bt.Prune(finalised.hash)

	for _, prunedHash := range pruned {
		prunedNode := copy.getNode(prunedHash)
		if prunedNode.isDescendantOf(finalised) {
			t.Fatal("pruned node that's descendant of finalised node!!")
		}

		if finalised.isDescendantOf(prunedNode) {
			t.Fatal("pruned an ancestor of the finalised node!!")
		}
	}

	require.NotEqual(t, 0, len(bt.leaves.nodes()))
	for _, leaf := range bt.leaves.nodes() {
		require.NotEqual(t, leaf.hash, finalised.hash)
		require.True(t, leaf.isDescendantOf(finalised))
	}
}

func TestBlockTree_PruneCache(t *testing.T) {
	var bt *BlockTree
	var branches []testBranch

	for {
		bt, branches = createTestBlockTree(testHeader, 5, nil)
		if len(branches) > 0 && len(bt.getNode(branches[0].hash).children) > 1 {
			break
		}
	}

	// pick some block to finalise
	finalised := bt.head.children[0].children[0].children[0]
	pruned := bt.Prune(finalised.hash)

	for _, prunedHash := range pruned {
		block, ok := bt.nodeCache[prunedHash]

		require.False(t, ok)
		require.Nil(t, block)
	}

}

func TestBlockTree_DeepCopy(t *testing.T) {
	bt, _ := createFlatTree(t, 8)

	btCopy := bt.DeepCopy()

	require.Equal(t, bt.db, btCopy.db)
	for hash := range bt.nodeCache {
		b, ok := btCopy.nodeCache[hash]
		b2 := bt.nodeCache[hash]

		require.True(t, ok)
		require.True(t, b != b2)

		require.True(t, equalNodeValue(b, b2))

	}
	require.True(t, equalNodeValue(bt.head, btCopy.head), "BlockTree heads not equal")
	require.True(t, equalLeave(bt.leaves, btCopy.leaves), "BlockTree leaves not equal")

	btCopy.head = &node{}
	require.NotEqual(t, bt.head, btCopy.head)
}

func equalNodeValue(nd *node, ndCopy *node) bool {
	if nd.hash != ndCopy.hash {
		return false
	}
	if nd.depth.Cmp(ndCopy.depth) != 0 {
		return false
	}
	if nd.arrivalTime != ndCopy.arrivalTime {
		return false
	}
	for i, child := range nd.children {
		return equalNodeValue(child, ndCopy.children[i])
	}
	if nd.parent.hash != ndCopy.parent.hash {
		return false
	}
	if nd.parent.arrivalTime != ndCopy.parent.arrivalTime {
		return false
	}
	if nd.parent.depth.Cmp(ndCopy.parent.depth) != 0 {
		return false
	}
	return true
}

func equalLeave(lm *leafMap, lmCopy *leafMap) bool {
	lmm := lm.toMap()
	lmCopyM := lmCopy.toMap()
	for key, val := range lmm {
		lmCopyVal := lmCopyM[key]
		return equalNodeValue(val, lmCopyVal)
	}
	return true
}

func TestBlockTree_Rewind(t *testing.T) {
	var bt *BlockTree
	var branches []testBranch

	rewind := 6

	for {
		bt, branches = createTestBlockTree(testHeader, 12, nil)
		if len(branches) > 0 && len(bt.getNode(branches[0].hash).children) > 1 {
			break
		}
	}

	start := bt.leaves.deepestLeaf()

	bt.Rewind(rewind)
	deepest := bt.leaves.deepestLeaf()
	require.Equal(t, start.depth.Int64()-int64(rewind), deepest.depth.Int64())
}
