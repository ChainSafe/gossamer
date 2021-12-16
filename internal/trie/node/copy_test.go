package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testForSliceModif(t *testing.T, original, copied []byte) {
	t.Helper()
	require.Equal(t, len(original), len(copied))
	if len(copied) == 0 {
		// cannot test for modification
		return
	}
	original[0]++
	assert.NotEqual(t, copied, original)
}

func Test_Branch_Copy(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch         *Branch
		expectedBranch *Branch
	}{
		"empty branch": {
			branch:         &Branch{},
			expectedBranch: &Branch{},
		},
		"non empty branch": {
			branch: &Branch{
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Children: [16]Node{
					nil, nil, &Leaf{Key: []byte{9}},
				},
				dirty:      true,
				hashDigest: []byte{5},
				encoding:   []byte{6},
			},
			expectedBranch: &Branch{
				Key:   []byte{1, 2},
				Value: []byte{3, 4},
				Children: [16]Node{
					nil, nil, &Leaf{Key: []byte{9}},
				},
				dirty:      true,
				hashDigest: []byte{5},
				encoding:   []byte{6},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nodeCopy := testCase.branch.Copy()

			branchCopy, ok := nodeCopy.(*Branch)
			require.True(t, ok)

			assert.Equal(t, testCase.expectedBranch, branchCopy)
			testForSliceModif(t, testCase.branch.Key, branchCopy.Key)
			testForSliceModif(t, testCase.branch.Value, branchCopy.Value)
			testForSliceModif(t, testCase.branch.hashDigest, branchCopy.hashDigest)
			testForSliceModif(t, testCase.branch.encoding, branchCopy.encoding)

			testCase.branch.Children[15] = &Leaf{Key: []byte("modified")}
			assert.NotEqual(t, branchCopy.Children, testCase.branch.Children)
		})
	}
}

func Test_Leaf_Copy(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		leaf         *Leaf
		expectedLeaf *Leaf
	}{
		"empty leaf": {
			leaf:         &Leaf{},
			expectedLeaf: &Leaf{},
		},
		"non empty leaf": {
			leaf: &Leaf{
				Key:        []byte{1, 2},
				Value:      []byte{3, 4},
				dirty:      true,
				hashDigest: []byte{5},
				encoding:   []byte{6},
			},
			expectedLeaf: &Leaf{
				Key:        []byte{1, 2},
				Value:      []byte{3, 4},
				dirty:      true,
				hashDigest: []byte{5},
				encoding:   []byte{6},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nodeCopy := testCase.leaf.Copy()

			leafCopy, ok := nodeCopy.(*Leaf)
			require.True(t, ok)

			assert.Equal(t, testCase.expectedLeaf, leafCopy)
			testForSliceModif(t, testCase.leaf.Key, leafCopy.Key)
			testForSliceModif(t, testCase.leaf.Value, leafCopy.Value)
			testForSliceModif(t, testCase.leaf.hashDigest, leafCopy.hashDigest)
			testForSliceModif(t, testCase.leaf.encoding, leafCopy.encoding)
		})
	}
}
