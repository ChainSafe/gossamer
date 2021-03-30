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
	"io"
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
}

type mockDigestItem struct {
	i int
}

func newMockDigestItem(i int) *mockDigestItem {
	return &mockDigestItem{
		i: i,
	}
}

func (d *mockDigestItem) String() string {
	return ""
}

func (d *mockDigestItem) Type() byte {
	return byte(d.i)
}

func (d *mockDigestItem) Encode() ([]byte, error) {
	return []byte{byte(d.i)}, nil
}

func (d *mockDigestItem) Decode(_ io.Reader) error {
	return nil
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
		header := &types.Header{
			ParentHash: previousHash,
			Number:     big.NewInt(int64(i)),
			Digest:     types.Digest{newMockDigestItem(9)},
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
		header := &types.Header{
			ParentHash: previousHash,
			Number:     big.NewInt(int64(i)),
			Digest:     types.Digest{newMockDigestItem(7)},
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

	// pick some block to finalize
	finalized := bt.head.children[0].children[0].children[0]
	pruned := bt.Prune(finalized.hash)

	for _, prunedHash := range pruned {
		prunedNode := copy.getNode(prunedHash)
		if prunedNode.isDescendantOf(finalized) {
			t.Fatal("pruned node that's descendant of finalized node!!")
		}

		if finalized.isDescendantOf(prunedNode) {
			t.Fatal("pruned an ancestor of the finalized node!!")
		}
	}
}

func TestBlockTree_DeepCopy(t *testing.T) {
	bt, _ := createFlatTree(t, 8)

	btCopy := bt.DeepCopy()

	require.Equal(t, bt.db, btCopy.db)
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
