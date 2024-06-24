// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
	"github.com/ChainSafe/gossamer/pkg/trie/tracking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

var alwaysTrue = func([]byte) bool { return true }

func Test_EmptyHash(t *testing.T) {
	t.Parallel()

	expected := common.Hash{
		0x3, 0x17, 0xa, 0x2e, 0x75, 0x97, 0xb7, 0xb7,
		0xe3, 0xd8, 0x4c, 0x5, 0x39, 0x1d, 0x13, 0x9a,
		0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0,
		0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14,
	}
	assert.Equal(t, expected, trie.EmptyHash)
}

func Test_NewEmptyTrie(t *testing.T) {
	expectedTrie := &InMemoryTrie{
		childTries: make(map[common.Hash]*InMemoryTrie),
		deltas:     tracking.New(),
		db:         db.NewEmptyMemoryDB(),
	}
	trie := NewEmptyTrie()
	assert.Equal(t, expectedTrie, trie)
}

func Test_NewTrie(t *testing.T) {
	root := &node.Node{
		PartialKey:   []byte{0},
		StorageValue: []byte{17},
	}
	expectedTrie := &InMemoryTrie{
		root: &node.Node{
			PartialKey:   []byte{0},
			StorageValue: []byte{17},
		},
		childTries: make(map[common.Hash]*InMemoryTrie),
		deltas:     tracking.New(),
	}
	trie := NewTrie(root, nil)
	assert.Equal(t, expectedTrie, trie)
}

func Test_Trie_Snapshot(t *testing.T) {
	t.Parallel()

	emptyDeltas := newDeltas()
	setDeltas := newDeltas("0x01")

	trie := &InMemoryTrie{
		generation: 8,
		root:       &node.Node{PartialKey: []byte{8}, StorageValue: []byte{1}},
		childTries: map[common.Hash]*InMemoryTrie{
			{1}: {
				generation: 1,
				root:       &node.Node{PartialKey: []byte{1}, StorageValue: []byte{1}},
				deltas:     setDeltas,
			},
			{2}: {
				generation: 2,
				root:       &node.Node{PartialKey: []byte{2}, StorageValue: []byte{1}},
				deltas:     setDeltas,
			},
		},
		deltas: setDeltas,
	}

	expectedTrie := &InMemoryTrie{
		generation: 9,
		root:       &node.Node{PartialKey: []byte{8}, StorageValue: []byte{1}},
		childTries: map[common.Hash]*InMemoryTrie{
			{1}: {
				generation: 2,
				root:       &node.Node{PartialKey: []byte{1}, StorageValue: []byte{1}},
				deltas:     emptyDeltas,
			},
			{2}: {
				generation: 3,
				root:       &node.Node{PartialKey: []byte{2}, StorageValue: []byte{1}},
				deltas:     emptyDeltas,
			},
		},
		deltas: emptyDeltas,
	}

	newTrie := trie.Snapshot()

	assert.Equal(t, expectedTrie.childTries, newTrie.childTries)
}

func Test_Trie_handleTrackedDeltas(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie          InMemoryTrie
		success       bool
		pendingDeltas tracking.Getter
		expectedTrie  InMemoryTrie
	}{
		"no_success_and_generation_1": {
			trie: InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0x01"),
			},
			pendingDeltas: newDeltas("0x02"),
			expectedTrie: InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0x01"),
			},
		},
		"success_and_generation_0": {
			trie: InMemoryTrie{
				deltas: newDeltas("0x01"),
			},
			success:       true,
			pendingDeltas: newDeltas("0x02"),
			expectedTrie: InMemoryTrie{
				deltas: newDeltas("0x01"),
			},
		},
		"success_and_generation_1": {
			trie: InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0x01"),
			},
			success:       true,
			pendingDeltas: newDeltas("0x01", "0x02"),
			expectedTrie: InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0x01", "0x02"),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			trie.HandleTrackedDeltas(testCase.success, testCase.pendingDeltas)

			assert.Equal(t, testCase.expectedTrie, trie)
		})
	}
}

func Test_Trie_prepForMutation(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                  *InMemoryTrie
		currentNode           *node.Node
		copySettings          node.CopySettings
		pendingDeltas         tracking.DeltaRecorder
		newNode               *node.Node
		copied                bool
		errSentinel           error
		errMessage            string
		expectedPendingDeltas tracking.DeltaRecorder
	}{
		"no_update": {
			trie: &InMemoryTrie{
				generation: 1,
			},
			currentNode: &node.Node{
				Generation: 1,
				PartialKey: []byte{1},
			},
			copySettings: node.DefaultCopySettings,
			newNode: &node.Node{
				Generation: 1,
				PartialKey: []byte{1},
				Dirty:      true,
			},
		},
		"update_without_registering_deleted_merkle_value": {
			trie: &InMemoryTrie{
				generation: 2,
			},
			currentNode: &node.Node{
				Generation: 1,
				PartialKey: []byte{1},
			},
			copySettings: node.DefaultCopySettings,
			newNode: &node.Node{
				Generation: 2,
				PartialKey: []byte{1},
				Dirty:      true,
			},
			copied: true,
		},
		"update_and_register_deleted_Merkle_value": {
			trie: &InMemoryTrie{
				generation: 2,
			},
			pendingDeltas: newDeltas(),
			currentNode: &node.Node{
				Generation: 1,
				PartialKey: []byte{1},
				StorageValue: []byte{
					1, 2, 3, 4, 5, 6, 7, 8,
					9, 10, 11, 12, 13, 14, 15, 16,
					17, 18, 19, 20, 21, 22, 23, 24,
					25, 26, 27, 28, 29, 30, 31, 32},
			},
			copySettings: node.DefaultCopySettings,
			newNode: &node.Node{
				Generation: 2,
				PartialKey: []byte{1},
				StorageValue: []byte{
					1, 2, 3, 4, 5, 6, 7, 8,
					9, 10, 11, 12, 13, 14, 15, 16,
					17, 18, 19, 20, 21, 22, 23, 24,
					25, 26, 27, 28, 29, 30, 31, 32},
				Dirty: true,
			},
			copied:                true,
			expectedPendingDeltas: newDeltas("0x98fcd66ba312c29ef193052fd0c14c6e38b158bd5c0235064594cacc1ab5965d"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := testCase.trie.DeepCopy()

			newNode, err := trie.prepForMutation(testCase.currentNode, testCase.copySettings,
				testCase.pendingDeltas)

			require.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
			assert.Equal(t, expectedTrie, trie)

			// Check for deep copy
			if newNode != nil && testCase.copied {
				if newNode.Dirty {
					newNode.SetClean()
				} else {
					newNode.SetDirty()
				}
				assert.NotEqual(t, testCase.newNode, newNode)
			}
		})
	}
}

func Test_Trie_registerDeletedNodeHash(t *testing.T) {
	t.Parallel()

	someSmallNode := &node.Node{
		PartialKey:   []byte{1},
		StorageValue: []byte{2},
	}

	testCases := map[string]struct {
		trie                  InMemoryTrie
		node                  *node.Node
		pendingDeltas         *tracking.Deltas
		expectedPendingDeltas *tracking.Deltas
		expectedTrie          InMemoryTrie
	}{
		"dirty_node_not_registered": {
			node: &node.Node{Dirty: true},
		},
		"clean_root_node_registered": {
			node:                  someSmallNode,
			trie:                  InMemoryTrie{root: someSmallNode},
			pendingDeltas:         newDeltas(),
			expectedPendingDeltas: newDeltas("0x60516d0bb6e1bbfb1293f1b276ea9505e9f4a4e7d98f620d05115e0b85274ae1"),
			expectedTrie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1},
					StorageValue: []byte{2},
					MerkleValue: []byte{
						0x60, 0x51, 0x6d, 0x0b, 0xb6, 0xe1, 0xbb, 0xfb,
						0x12, 0x93, 0xf1, 0xb2, 0x76, 0xea, 0x95, 0x05,
						0xe9, 0xf4, 0xa4, 0xe7, 0xd9, 0x8f, 0x62, 0x0d,
						0x05, 0x11, 0x5e, 0x0b, 0x85, 0x27, 0x4a, 0xe1},
				},
			},
		},
		"clean_node_with_inlined_Merkle_value_not_registered": {
			node: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
			},
		},
		"clean_node_with_hash_Merkle_value_registered": {
			node: &node.Node{
				PartialKey: []byte{1},
				StorageValue: []byte{
					1, 2, 3, 4, 5, 6, 7, 8,
					9, 10, 11, 12, 13, 14, 15, 16,
					17, 18, 19, 20, 21, 22, 23, 24,
					25, 26, 27, 28, 29, 30, 31, 32},
			},
			pendingDeltas:         newDeltas(),
			expectedPendingDeltas: newDeltas("0x98fcd66ba312c29ef193052fd0c14c6e38b158bd5c0235064594cacc1ab5965d"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie

			err := trie.registerDeletedNodeHash(testCase.node,
				testCase.pendingDeltas)

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
			assert.Equal(t, testCase.expectedTrie, trie)
		})
	}
}

func getPointer(x interface{}) (pointer uintptr, ok bool) {
	func() {
		defer func() {
			ok = recover() == nil
		}()
		valueOfX := reflect.ValueOf(x)
		pointer = valueOfX.Pointer()
	}()
	return pointer, ok
}

func assertPointersNotEqual(t *testing.T, a, b interface{}) {
	t.Helper()
	pointerA, okA := getPointer(a)
	pointerB, okB := getPointer(b)
	require.Equal(t, okA, okB)

	switch {
	case pointerA == 0 && pointerB == 0: // nil and nil
	case okA:
		assert.NotEqual(t, pointerA, pointerB)
	default: // values like `int`
	}
}

// testTrieForDeepCopy verifies each pointer of the copied trie
// are different from the new copy trie.
func testTrieForDeepCopy(t *testing.T, original, copy *InMemoryTrie) {
	assertPointersNotEqual(t, original, copy)
	if original == nil {
		return
	}
	assertPointersNotEqual(t, original.generation, copy.generation)
	assertPointersNotEqual(t, original.deltas, copy.deltas)
	assertPointersNotEqual(t, original.childTries, copy.childTries)
	for hashKey, childTrie := range copy.childTries {
		originalChildTrie := original.childTries[hashKey]
		testTrieForDeepCopy(t, originalChildTrie, childTrie)
	}
	assertPointersNotEqual(t, original.root, copy.root)
}

func Test_Trie_DeepCopy(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trieOriginal *InMemoryTrie
		trieCopy     *InMemoryTrie
	}{
		"nil": {},
		"empty_trie": {
			trieOriginal: &InMemoryTrie{},
			trieCopy:     &InMemoryTrie{},
		},
		"filled_trie": {
			trieOriginal: &InMemoryTrie{
				generation: 1,
				root:       &node.Node{PartialKey: []byte{1, 2}, StorageValue: []byte{1}},
				childTries: map[common.Hash]*InMemoryTrie{
					{1, 2, 3}: {
						generation: 2,
						root:       &node.Node{PartialKey: []byte{1}, StorageValue: []byte{1}},
						deltas:     newDeltas("0x01", "0x02"),
					},
				},
				deltas: newDeltas("0x01", "0x02"),
			},
			trieCopy: &InMemoryTrie{
				generation: 1,
				root:       &node.Node{PartialKey: []byte{1, 2}, StorageValue: []byte{1}},
				childTries: map[common.Hash]*InMemoryTrie{
					{1, 2, 3}: {
						generation: 2,
						root:       &node.Node{PartialKey: []byte{1}, StorageValue: []byte{1}},
						deltas:     newDeltas("0x01", "0x02"),
					},
				},
				deltas: newDeltas("0x01", "0x02"),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trieCopy := testCase.trieOriginal.DeepCopy()

			assert.Equal(t, trieCopy, testCase.trieCopy)

			testTrieForDeepCopy(t, testCase.trieOriginal, trieCopy)
		})
	}
}

func Test_Trie_RootNode(t *testing.T) {
	t.Parallel()

	trie := InMemoryTrie{
		root: &node.Node{
			PartialKey:   []byte{1, 2, 3},
			StorageValue: []byte{1},
		},
	}
	expectedRoot := &node.Node{
		PartialKey:   []byte{1, 2, 3},
		StorageValue: []byte{1},
	}

	root := trie.RootNode()

	assert.Equal(t, expectedRoot, root)
}

func Test_Trie_MustHash(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		hash := trie.V0.MustHash(&InMemoryTrie{})

		expectedHash := common.Hash{
			0x3, 0x17, 0xa, 0x2e, 0x75, 0x97, 0xb7, 0xb7,
			0xe3, 0xd8, 0x4c, 0x5, 0x39, 0x1d, 0x13, 0x9a,
			0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0,
			0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14}
		assert.Equal(t, expectedHash, hash)
	})
}

func Test_Trie_Hash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         InMemoryTrie
		hash         common.Hash
		errWrapped   error
		errMessage   string
		expectedTrie InMemoryTrie
	}{
		"nil_root": {
			hash: common.Hash{
				0x3, 0x17, 0xa, 0x2e, 0x75, 0x97, 0xb7, 0xb7,
				0xe3, 0xd8, 0x4c, 0x5, 0x39, 0x1d, 0x13, 0x9a,
				0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0,
				0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14},
		},
		"leaf_root": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1, 2, 3},
					StorageValue: []byte{1},
				},
			},
			hash: common.Hash{
				0xa8, 0x13, 0x7c, 0xee, 0xb4, 0xad, 0xea, 0xac,
				0x9e, 0x5b, 0x37, 0xe2, 0x8e, 0x7d, 0x64, 0x78,
				0xac, 0xba, 0xb0, 0x6e, 0x90, 0x76, 0xe4, 0x67,
				0xa1, 0xd8, 0xa2, 0x29, 0x4e, 0x4a, 0xd9, 0xa3},
			expectedTrie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1, 2, 3},
					StorageValue: []byte{1},
					MerkleValue: []byte{
						0xa8, 0x13, 0x7c, 0xee, 0xb4, 0xad, 0xea, 0xac,
						0x9e, 0x5b, 0x37, 0xe2, 0x8e, 0x7d, 0x64, 0x78,
						0xac, 0xba, 0xb0, 0x6e, 0x90, 0x76, 0xe4, 0x67,
						0xa1, 0xd8, 0xa2, 0x29, 0x4e, 0x4a, 0xd9, 0xa3,
					},
				},
			},
		},
		"branch_root": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1, 2, 3},
					StorageValue: []byte("branch"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						{PartialKey: []byte{9}, StorageValue: []byte{1}},
					}),
				},
			},
			hash: common.Hash{
				0xaa, 0x7e, 0x57, 0x48, 0xb0, 0x27, 0x4d, 0x18,
				0xf5, 0x1c, 0xfd, 0x36, 0x4c, 0x4b, 0x56, 0x4a,
				0xf5, 0x37, 0x9d, 0xd7, 0xcb, 0xf5, 0x80, 0x15,
				0xf0, 0xe, 0xd3, 0x39, 0x48, 0x21, 0xe3, 0xdd},
			expectedTrie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1, 2, 3},
					StorageValue: []byte("branch"),
					MerkleValue: []byte{
						0xaa, 0x7e, 0x57, 0x48, 0xb0, 0x27, 0x4d, 0x18,
						0xf5, 0x1c, 0xfd, 0x36, 0x4c, 0x4b, 0x56, 0x4a,
						0xf5, 0x37, 0x9d, 0xd7, 0xcb, 0xf5, 0x80, 0x15,
						0xf0, 0x0e, 0xd3, 0x39, 0x48, 0x21, 0xe3, 0xdd,
					},
					Descendants: 1,
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{9},
							StorageValue: []byte{1},
							MerkleValue:  []byte{0x41, 0x09, 0x04, 0x01},
						},
					}),
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			hash, err := trie.V0.Hash(&testCase.trie)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.hash, hash)
			assert.Equal(t, testCase.expectedTrie, testCase.trie)
		})
	}
}

func entriesMatch(t *testing.T, expected, actual map[string][]byte) {
	t.Helper()

	for expectedKeyLEString, expectedValue := range expected {
		expectedKeyLE := []byte(expectedKeyLEString)
		actualValue, ok := actual[expectedKeyLEString]
		if !ok {
			t.Errorf("key 0x%x is missing from entries", expectedKeyLE)
			continue
		}

		if !bytes.Equal(expectedValue, actualValue) {
			t.Errorf("for key 0x%x, expected value 0x%x but got actual value 0x%x",
				expectedKeyLE, expectedValue, actualValue)
		}
	}

	for actualKeyLEString, actualValue := range actual {
		actualKeyLE := []byte(actualKeyLEString)
		_, ok := expected[actualKeyLEString]
		if !ok {
			t.Errorf("actual key 0x%x with value 0x%x was not expected",
				actualKeyLE, actualValue)
		}
	}
}

func Test_Trie_Entries(t *testing.T) {
	t.Parallel()

	t.Run("simple_root", func(t *testing.T) {
		t.Parallel()

		root := &node.Node{
			PartialKey:   []byte{0x0, 0xa},
			StorageValue: []byte("root"),
			Descendants:  2,
			Children: padRightChildren([]*node.Node{
				{ // index 0
					PartialKey:   []byte{0xb},
					StorageValue: []byte("leaf"),
				},
				nil,
				{ // index 2
					PartialKey:   []byte{0xb},
					StorageValue: []byte("leaf"),
				},
			}),
		}

		trie := NewTrie(root, nil)

		entries := trie.Entries()

		expectedEntries := map[string][]byte{
			string([]byte{0x0a}):       []byte("root"),
			string([]byte{0x0a, 0xb}):  []byte("leaf"),
			string([]byte{0x0a, 0x2b}): []byte("leaf"),
		}

		entriesMatch(t, expectedEntries, entries)
	})

	t.Run("custom_root", func(t *testing.T) {
		t.Parallel()

		root := &node.Node{
			PartialKey:   []byte{0xa, 0xb},
			StorageValue: []byte("root"),
			Descendants:  5,
			Children: padRightChildren([]*node.Node{
				nil, nil, nil,
				{ // branch with value at child index 3
					PartialKey:   []byte{0xb},
					StorageValue: []byte("branch 1"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						nil, nil, nil,
						{ // leaf at child index 3
							PartialKey:   []byte{0xc},
							StorageValue: []byte("bottom leaf"),
						},
					}),
				},
				nil, nil, nil,
				{ // leaf at child index 7
					PartialKey:   []byte{0xd},
					StorageValue: []byte("top leaf"),
				},
				nil,
				{ // branch without value at child index 9
					PartialKey:   []byte{0xe},
					StorageValue: []byte("branch 2"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						{ // leaf at child index 0
							PartialKey:   []byte{0xf},
							StorageValue: []byte("bottom leaf 2"),
						}, nil, nil,
					}),
				},
			}),
		}

		trie := NewTrie(root, nil)

		entries := trie.Entries()

		expectedEntries := map[string][]byte{
			string([]byte{0xab}):             []byte("root"),
			string([]byte{0xab, 0x7d}):       []byte("top leaf"),
			string([]byte{0xab, 0x3b}):       []byte("branch 1"),
			string([]byte{0xab, 0x3b, 0x3c}): []byte("bottom leaf"),
			string([]byte{0xab, 0x9e}):       []byte("branch 2"),
			string([]byte{0xab, 0x9e, 0x0f}): []byte("bottom leaf 2"),
		}

		entriesMatch(t, expectedEntries, entries)
	})

	t.Run("end_to_end_v0", func(t *testing.T) {
		t.Parallel()

		trie := InMemoryTrie{
			root:       nil,
			childTries: make(map[common.Hash]*InMemoryTrie),
			db:         db.NewEmptyMemoryDB(),
		}

		kv := map[string][]byte{
			"ab":  []byte("pen"),
			"abc": []byte("penguin"),
			"hy":  []byte("feather"),
			"cb":  []byte("noot"),
		}

		for k, v := range kv {
			trie.Put([]byte(k), v)
		}

		entries := trie.Entries()

		assert.Equal(t, kv, entries)
	})

	t.Run("end_to_end_v1", func(t *testing.T) {
		t.Parallel()

		trie := InMemoryTrie{
			root:       nil,
			childTries: make(map[common.Hash]*InMemoryTrie),
			db:         db.NewEmptyMemoryDB(),
		}

		kv := map[string][]byte{
			"ab":   []byte("pen"),
			"abc":  []byte("penguin"),
			"hy":   []byte("feather"),
			"long": []byte("newvaluewithmorethan32byteslength"),
		}

		for k, v := range kv {
			trie.Put([]byte(k), v)
		}

		entries := trie.Entries()

		assert.Equal(t, kv, entries)
	})
}

func Test_Trie_NextKey(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie    InMemoryTrie
		key     []byte
		nextKey []byte
	}{
		"nil_root_and_nil_key_returns_nil": {},
		"nil_root_returns_nil": {
			key: []byte{2},
		},
		"nil_key_returns_root_leaf": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte{1},
				},
			},
			nextKey: []byte{2},
		},
		"key_smaller_than_root_leaf_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte{1},
				},
			},
			key:     []byte{0x10}, // 10 => [1, 0] in nibbles
			nextKey: []byte{2},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nextKey := testCase.trie.NextKey(testCase.key, alwaysTrue)

			assert.Equal(t, testCase.nextKey, nextKey)
		})
	}
}

func Test_nextKey(t *testing.T) {
	// Note this test is basically testing trie.NextKey without
	// the headaches associated with converting nibbles and
	// LE keys back and forth
	t.Parallel()

	testCases := map[string]struct {
		trie    InMemoryTrie
		key     []byte
		nextKey []byte
	}{
		"nil_root_and_nil_key_returns_nil": {},
		"nil_root_returns_nil": {
			key: []byte{2},
		},
		"nil_key_returns_root_leaf": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte{1},
				},
			},
			nextKey: []byte{2},
		},
		"key_smaller_than_root_leaf_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte{1},
				},
			},
			key:     []byte{1},
			nextKey: []byte{2},
		},
		"key_equal_to_root_leaf_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte{1},
				},
			},
			key: []byte{2},
		},
		"key_greater_than_root_leaf_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte{1},
				},
			},
			key: []byte{3},
		},
		"key_smaller_than_root_branch_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte("branch"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{1},
							StorageValue: []byte{1},
						},
					}),
				},
			},
			key:     []byte{1},
			nextKey: []byte{2},
		},
		"key_equal_to_root_branch_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2},
					StorageValue: []byte("branch"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{1},
							StorageValue: []byte{1},
						},
					}),
				},
			},
			key: []byte{2, 0, 1},
		},
		"key_smaller_than_leaf_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1},
					StorageValue: []byte("branch"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							PartialKey:   []byte{3},
							StorageValue: []byte{1},
						},
					}),
				},
			},
			key:     []byte{1, 2, 2},
			nextKey: []byte{1, 2, 3},
		},
		"key_equal_to_leaf_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1},
					StorageValue: []byte("branch"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							PartialKey:   []byte{3},
							StorageValue: []byte{1},
						},
					}),
				},
			},
			key: []byte{1, 2, 3},
		},
		"key_greater_than_leaf_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1},
					StorageValue: []byte("branch"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							PartialKey:   []byte{3},
							StorageValue: []byte{1},
						},
					}),
				},
			},
			key: []byte{1, 2, 4},
		},
		"next_key_branch_with_value": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1},
					StorageValue: []byte("top branch"),
					Descendants:  2,
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							PartialKey:   []byte{3},
							StorageValue: []byte("branch 1"),
							Descendants:  1,
							Children: padRightChildren([]*node.Node{
								nil, nil, nil, nil,
								{
									// full key [1, 2, 3, 4, 5]
									PartialKey:   []byte{0x5},
									StorageValue: []byte("bottom leaf"),
								},
							}),
						},
					}),
				},
			},
			key:     []byte{1},
			nextKey: []byte{1, 2, 3},
		},
		"next_key_go_through_branch_without_value": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:  []byte{1},
					Descendants: 2,
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							PartialKey:  []byte{3},
							Descendants: 1,
							Children: padRightChildren([]*node.Node{
								nil, nil, nil, nil,
								{
									// full key [1, 2, 3, 4, 5]
									PartialKey:   []byte{0x5},
									StorageValue: []byte("bottom leaf"),
								},
							}),
						},
					}),
				},
			},
			key:     []byte{0},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"next_key_leaf_from_bottom_branch": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:  []byte{1},
					Descendants: 2,
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							PartialKey:   []byte{3},
							StorageValue: []byte("bottom branch"),
							Descendants:  1,
							Children: padRightChildren([]*node.Node{
								nil, nil, nil, nil,
								{
									// full key [1, 2, 3, 4, 5]
									PartialKey:   []byte{0x5},
									StorageValue: []byte("bottom leaf"),
								},
							}),
						},
					}),
				},
			},
			key:     []byte{1, 2, 3},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"next_key_greater_than_branch": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:  []byte{1},
					Descendants: 2,
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							PartialKey:   []byte{3},
							StorageValue: []byte("bottom branch"),
							Descendants:  1,
							Children: padRightChildren([]*node.Node{
								nil, nil, nil, nil,
								{
									// full key [1, 2, 3, 4, 5]
									PartialKey:   []byte{0x5},
									StorageValue: []byte("bottom leaf"),
								},
							}),
						},
					}),
				},
			},
			key:     []byte{1, 2, 3},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"key_smaller_length_and_greater_than_root_branch_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2, 0},
					StorageValue: []byte("branch"),
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						{PartialKey: []byte{1}, StorageValue: []byte{1}},
					}),
				},
			},
			key: []byte{3},
		},
		"key_smaller_length_and_greater_than_root_leaf_full_key": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{2, 0},
					StorageValue: []byte("leaf"),
				},
			},
			key: []byte{3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			originalTrie := testCase.trie.DeepCopy()

			nextKey := findNextKey(testCase.trie.root, nil, testCase.key, alwaysTrue)

			assert.Equal(t, testCase.nextKey, nextKey)
			assert.Equal(t, *originalTrie, testCase.trie) // ensure no mutation
		})
	}
}

func Test_Trie_Put(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         *InMemoryTrie
		key          []byte
		value        []byte
		expectedTrie *InMemoryTrie
	}{
		"trie_with_key_and_value": {
			trie: &InMemoryTrie{
				generation: 1,
				deltas:     newDeltas(),
				root: &node.Node{
					PartialKey:   []byte{1, 2, 0, 5},
					StorageValue: []byte{1},
				},
			},
			key:   []byte{0x12, 0x16},
			value: []byte{2},
			expectedTrie: &InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0xa195089c3e8f8b5b36978700ad954aed99e08413cfc1e2b4c00a5d064abe66a9"),
				root: &node.Node{
					PartialKey:  []byte{1, 2},
					Generation:  1,
					Dirty:       true,
					Descendants: 2,
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{5},
							StorageValue: []byte{1},
							Generation:   1,
							Dirty:        true,
						},
						{
							PartialKey:   []byte{6},
							StorageValue: []byte{2},
							Generation:   1,
							Dirty:        true,
						},
					}),
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			trie.Put(testCase.key, testCase.value)

			assert.Equal(t, testCase.expectedTrie, trie)
		})
	}
}

func Test_Trie_insert(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                  InMemoryTrie
		parent                *node.Node
		key                   []byte
		value                 []byte
		pendingDeltas         tracking.DeltaRecorder
		newNode               *node.Node
		mutated               bool
		nodesCreated          uint32
		expectedPendingDeltas tracking.DeltaRecorder
	}{
		"nil_parent": {
			trie: InMemoryTrie{
				generation: 1,
			},
			key:   []byte{1},
			value: []byte("leaf"),
			newNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("leaf"),
				Generation:   1,
				Dirty:        true,
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"branch_parent": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("branch"),
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					nil,
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{1, 0},
			value: []byte("leaf"),
			newNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("branch"),
				Generation:   1,
				Dirty:        true,
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{},
						StorageValue: []byte("leaf"),
						Generation:   1,
						Dirty:        true,
					},
					{
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
						MerkleValue:  []byte{0x41, 0x02, 0x04, 0x01},
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"override_leaf_parent": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("original leaf"),
			},
			key:   []byte{1},
			value: []byte("new leaf"),
			newNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("new leaf"),
				Generation:   1,
				Dirty:        true,
			},
			mutated: true,
		},
		"write_same_leaf_value_to_leaf_parent": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("same"),
			},
			key:   []byte{1},
			value: []byte("same"),
			newNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("same"),
			},
		},
		"write_leaf_as_child_to_parent_leaf": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("original leaf"),
			},
			key:   []byte{1, 0},
			value: []byte("leaf"),
			newNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("original leaf"),
				Dirty:        true,
				Generation:   1,
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{},
						StorageValue: []byte("leaf"),
						Generation:   1,
						Dirty:        true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"write_leaf_as_divergent_child_next_to_parent_leaf": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte("original leaf"),
			},
			key:   []byte{2, 3},
			value: []byte("leaf"),
			newNode: &node.Node{
				PartialKey:  []byte{},
				Dirty:       true,
				Generation:  1,
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					nil,
					{
						PartialKey:   []byte{2},
						StorageValue: []byte("original leaf"),
						Dirty:        true,
						Generation:   1,
					},
					{
						PartialKey:   []byte{3},
						StorageValue: []byte("leaf"),
						Generation:   1,
						Dirty:        true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 2,
		},
		"override_leaf_value": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			key:   []byte{1},
			value: []byte("leaf"),
			newNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("leaf"),
				Dirty:        true,
				Generation:   1,
			},
			mutated: true,
		},
		"write_leaf_as_child_to_leaf": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			key:   []byte{1},
			value: []byte("leaf"),
			newNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte("leaf"),
				Dirty:        true,
				Generation:   1,
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					nil, nil,
					{
						PartialKey:   []byte{},
						StorageValue: []byte{1},
						Dirty:        true,
						Generation:   1,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newNode, mutated, nodesCreated, err := trie.insert(
				testCase.parent, testCase.key, testCase.value,
				testCase.pendingDeltas)

			require.NoError(t, err)
			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.mutated, mutated)
			assert.Equal(t, testCase.nodesCreated, nodesCreated)
			assert.Equal(t, expectedTrie, trie)
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
		})
	}
}

func Test_Trie_insertInBranch(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		parent                *node.Node
		key                   []byte
		value                 []byte
		pendingDeltas         tracking.DeltaRecorder
		newNode               *node.Node
		mutated               bool
		nodesCreated          uint32
		errSentinel           error
		errMessage            string
		expectedPendingDeltas tracking.DeltaRecorder
	}{
		"insert_existing_value_to_branch": {
			parent: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte("same"),
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{2},
			value: []byte("same"),
			newNode: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte("same"),
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
		},
		"update_with_branch": {
			parent: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte("old"),
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{2},
			value: []byte("new"),
			newNode: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte("new"),
				Dirty:        true,
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			mutated: true,
		},
		"update_with_leaf": {
			parent: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte("old"),
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{2},
			value: []byte("new"),
			newNode: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte("new"),
				Dirty:        true,
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			mutated: true,
		},
		"add_leaf_as_direct_child": {
			parent: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte{5},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{2, 3, 4, 5},
			value: []byte{6},
			newNode: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte{5},
				Dirty:        true,
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					nil, nil,
					{
						PartialKey:   []byte{4, 5},
						StorageValue: []byte{6},
						Dirty:        true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"insert_same_leaf_as_existing_direct_child_leaf": {
			parent: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte{5},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{2, 0, 1},
			value: []byte{1},
			newNode: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte{5},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
		},
		"add_leaf_as_nested_child": {
			parent: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte{5},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					nil, nil, nil,
					{
						PartialKey:  []byte{4},
						Descendants: 1,
						Children: padRightChildren([]*node.Node{
							{PartialKey: []byte{1}, StorageValue: []byte{1}},
						}),
					},
				}),
			},
			key:   []byte{2, 3, 4, 5, 6},
			value: []byte{6},
			newNode: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte{5},
				Dirty:        true,
				Descendants:  3,
				Children: padRightChildren([]*node.Node{
					nil, nil, nil,
					{
						PartialKey:  []byte{4},
						Dirty:       true,
						Descendants: 2,
						Children: padRightChildren([]*node.Node{
							{PartialKey: []byte{1}, StorageValue: []byte{1}},
							nil, nil, nil, nil,
							{
								PartialKey:   []byte{6},
								StorageValue: []byte{6},
								Dirty:        true,
							},
						}),
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"split_branch_for_longer_key": {
			parent: &node.Node{
				PartialKey:   []byte{2, 3},
				StorageValue: []byte{5},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{2, 4, 5, 6},
			value: []byte{6},
			newNode: &node.Node{
				PartialKey:  []byte{2},
				Dirty:       true,
				Descendants: 3,
				Children: padRightChildren([]*node.Node{
					nil, nil, nil,
					{
						PartialKey:   []byte{},
						StorageValue: []byte{5},
						Dirty:        true,
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							{PartialKey: []byte{1}, StorageValue: []byte{1}},
						}),
					},
					{
						PartialKey:   []byte{5, 6},
						StorageValue: []byte{6},
						Dirty:        true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 2,
		},
		"split_root_branch": {
			parent: &node.Node{
				PartialKey:   []byte{2, 3},
				StorageValue: []byte{5},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{3},
			value: []byte{6},
			newNode: &node.Node{
				PartialKey:  []byte{},
				Dirty:       true,
				Descendants: 3,
				Children: padRightChildren([]*node.Node{
					nil, nil,
					{
						PartialKey:   []byte{3},
						StorageValue: []byte{5},
						Dirty:        true,
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							{PartialKey: []byte{1}, StorageValue: []byte{1}},
						}),
					},
					{
						PartialKey:   []byte{},
						StorageValue: []byte{6},
						Dirty:        true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 2,
		},
		"update_with_leaf_at_empty_key": {
			parent: &node.Node{
				PartialKey:   []byte{2},
				StorageValue: []byte{5},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{},
			value: []byte{6},
			newNode: &node.Node{
				PartialKey:   []byte{},
				StorageValue: []byte{6},
				Dirty:        true,
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					nil, nil,
					{
						PartialKey:   []byte{},
						StorageValue: []byte{5},
						Dirty:        true,
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							{PartialKey: []byte{1}, StorageValue: []byte{1}},
						}),
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := new(InMemoryTrie)

			newNode, mutated, nodesCreated, err := trie.insertInBranch(
				testCase.parent, testCase.key, testCase.value,
				testCase.pendingDeltas)

			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.mutated, mutated)
			assert.Equal(t, testCase.nodesCreated, nodesCreated)
			assert.Equal(t, new(InMemoryTrie), trie) // check no mutation
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
		})
	}
}

func Test_LoadFromMap(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		data         map[string]string
		expectedTrie *InMemoryTrie
		errWrapped   error
		errMessage   string
	}{
		"nil_data": {
			expectedTrie: &InMemoryTrie{
				childTries: map[common.Hash]*InMemoryTrie{},
				deltas:     newDeltas(),
				db:         db.NewEmptyMemoryDB(),
			},
		},
		"empty_data": {
			data: map[string]string{},
			expectedTrie: &InMemoryTrie{
				childTries: map[common.Hash]*InMemoryTrie{},
				deltas:     newDeltas(),
				db:         db.NewEmptyMemoryDB(),
			},
		},
		"bad_key": {
			data: map[string]string{
				"0xa": "0x01",
			},
			errWrapped: hex.ErrLength,
			errMessage: "cannot convert key hex to bytes: encoding/hex: odd length hex string: 0xa",
		},
		"bad_value": {
			data: map[string]string{
				"0x01": "0xa",
			},
			errWrapped: hex.ErrLength,
			errMessage: "cannot convert value hex to bytes: encoding/hex: odd length hex string: 0xa",
		},
		"load_large_key_value": {
			data: map[string]string{
				"0x01": "0x1234567812345678123456781234567812345678123456781234567812345678", // 32 bytes
			},
			expectedTrie: &InMemoryTrie{
				root: &node.Node{
					PartialKey: []byte{00, 01},
					StorageValue: []byte{
						0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
						0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
						0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
						0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
					},
					Dirty: true,
				},
				childTries: map[common.Hash]*InMemoryTrie{},
				deltas:     newDeltas(),
				db:         db.NewEmptyMemoryDB(),
			},
		},
		"load_key_values": {
			data: map[string]string{
				"0x01":   "0x06",
				"0x0120": "0x07",
				"0x0130": "0x08",
			},
			expectedTrie: &InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{00, 01},
					StorageValue: []byte{6},
					Dirty:        true,
					Descendants:  2,
					Children: padRightChildren([]*node.Node{
						nil, nil,
						{
							PartialKey:   []byte{0},
							StorageValue: []byte{7},
							Dirty:        true,
						},
						{
							PartialKey:   []byte{0},
							StorageValue: []byte{8},
							Dirty:        true,
						},
					}),
				},
				childTries: map[common.Hash]*InMemoryTrie{},
				deltas:     newDeltas(),
				db:         db.NewEmptyMemoryDB(),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie, err := LoadFromMap(testCase.data, trie.V0)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}

			assert.Equal(t, testCase.expectedTrie, trie)
		})
	}
}

func Test_Trie_GetKeysWithPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie   *InMemoryTrie
		prefix []byte
		keys   [][]byte
	}{
		"some_trie": {
			trie: &InMemoryTrie{
				root: &node.Node{
					PartialKey:  []byte{0, 1},
					Descendants: 4,
					Children: padRightChildren([]*node.Node{
						{ // full key 0, 1, 0, 3
							PartialKey:  []byte{3},
							Descendants: 2,
							Children: padRightChildren([]*node.Node{
								{ // full key 0, 1, 0, 0, 4
									PartialKey:   []byte{4},
									StorageValue: []byte{1},
								},
								{ // full key 0, 1, 0, 1, 5
									PartialKey:   []byte{5},
									StorageValue: []byte{1},
								},
							}),
						},
						{ // full key 0, 1, 1, 9
							PartialKey:   []byte{9},
							StorageValue: []byte{1},
						},
					}),
				},
			},
			prefix: []byte{1},
			keys: [][]byte{
				{1, 3, 4},
				{1, 3, 0x15},
				{1, 0x19},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keys := testCase.trie.GetKeysWithPrefix(testCase.prefix)

			assert.Equal(t, testCase.keys, keys)
		})
	}
}

func Test_getKeysWithPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		parent       *node.Node
		prefix       []byte
		key          []byte
		keys         [][]byte
		expectedKeys [][]byte
	}{
		"nil_parent_returns_keys_passed": {
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"common_prefix_for_parent_branch_and_search_key": {
			parent: &node.Node{
				PartialKey:  []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
					{PartialKey: []byte{5}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"parent_branch_and_empty_key": {
			parent: &node.Node{
				PartialKey:  []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
					{PartialKey: []byte{5}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"search_key_smaller_than_branch_key_with_no_full_common_prefix": {
			parent: &node.Node{
				PartialKey:  []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
					{PartialKey: []byte{5}, StorageValue: []byte{1}},
				}),
			},
			key:          []byte{1, 3},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"common_prefix_smaller_tan_search_key": {
			parent: &node.Node{
				PartialKey:  []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
					{PartialKey: []byte{5}, StorageValue: []byte{1}},
				}),
			},
			key:          []byte{1, 2, 3},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"recursive_call": {
			parent: &node.Node{
				PartialKey:  []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
					{PartialKey: []byte{5}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2, 3, 0},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04}},
		},
		"parent_leaf_with_search_key_equal_to_common_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2, 3},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent_leaf_with_empty_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent_leaf_with_too_deep_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
			},
			prefix:       []byte{9, 8, 7},
			key:          []byte{1, 2, 3, 4},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"parent_leaf_with_shorter_matching_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent_leaf_with_not_matching_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
			},
			prefix:       []byte{9, 8, 7},
			key:          []byte{1, 3, 3},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keys := getKeysWithPrefix(testCase.parent,
				testCase.prefix, testCase.key, testCase.keys)

			assert.Equal(t, testCase.expectedKeys, keys)
		})
	}
}

func Test_addAllKeys(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		parent       *node.Node
		prefix       []byte
		keys         [][]byte
		expectedKeys [][]byte
	}{
		"nil_parent_returns_keys_passed": {
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"leaf_parent": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
			},
			prefix: []byte{9, 8, 7},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent_branch_without_value": {
			parent: &node.Node{
				PartialKey:  []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
					{PartialKey: []byte{5}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"parent_branch_with_empty_value": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
					{PartialKey: []byte{5}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keys := addAllKeys(testCase.parent,
				testCase.prefix, testCase.keys)

			assert.Equal(t, testCase.expectedKeys, keys)
		})
	}
}

func Test_Trie_Get(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie  *InMemoryTrie
		key   []byte
		value []byte
	}{
		"some_trie": {
			trie: &InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{0, 1},
					StorageValue: []byte{1, 3},
					Descendants:  3,
					Children: padRightChildren([]*node.Node{
						{ // full key 0, 1, 0, 3
							PartialKey:   []byte{3},
							StorageValue: []byte{1, 2},
							Descendants:  1,
							Children: padRightChildren([]*node.Node{
								{PartialKey: []byte{1}, StorageValue: []byte{1}},
							}),
						},
						{ // full key 0, 1, 1, 9
							PartialKey:   []byte{9},
							StorageValue: []byte{1, 2, 3, 4, 5},
						},
					}),
				},
			},
			key:   []byte{0x01, 0x19},
			value: []byte{1, 2, 3, 4, 5},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			value := testCase.trie.Get(testCase.key)

			assert.Equal(t, testCase.value, value)
		})
	}
}

func Test_retrieve(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defaultDBGetterMock := NewMockDBGetter(ctrl)
	defaultDBGetterMock.EXPECT().Get(gomock.Any()).Times(0)

	hashedValue := []byte("hashedvalue")
	hashedValueResult := []byte("hashedvalueresult")

	testCases := map[string]struct {
		parent *node.Node
		key    []byte
		value  []byte
		db     db.DBGetter
	}{
		"nil_parent": {
			key: []byte{1},
			db:  defaultDBGetterMock,
		},
		"leaf_key_match": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
			},
			key:   []byte{1},
			value: []byte{2},
			db:    defaultDBGetterMock,
		},
		"leaf_key_mismatch": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{2},
			},
			key: []byte{1},
			db:  defaultDBGetterMock,
		},
		"branch_key_match": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key:   []byte{1},
			value: []byte{2},
			db:    defaultDBGetterMock,
		},
		"branch_key_with_empty_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			value: []byte{2},
			db:    defaultDBGetterMock,
		},
		"branch_key_mismatch_with_shorter_search_key": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{2},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			key: []byte{1},
			db:  defaultDBGetterMock,
		},
		"bottom_leaf_in_branch": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					nil, nil,
					{ // full key 1, 2, 3
						PartialKey:   []byte{3},
						StorageValue: []byte{2},
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							nil, nil, nil, nil,
							{ // full key 1, 2, 3, 4, 5
								PartialKey:   []byte{5},
								StorageValue: []byte{3},
							},
						}),
					},
				}),
			},
			key:   []byte{1, 2, 3, 4, 5},
			value: []byte{3},
			db:    defaultDBGetterMock,
		},
		"bottom_leaf_with_hashed_value_in_branch": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					nil, nil,
					{ // full key 1, 2, 3
						PartialKey:   []byte{3},
						StorageValue: []byte{2},
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							nil, nil, nil, nil,
							{ // full key 1, 2, 3, 4, 5
								PartialKey:    []byte{5},
								StorageValue:  hashedValue,
								IsHashedValue: true,
							},
						}),
					},
				}),
			},
			key:   []byte{1, 2, 3, 4, 5},
			value: hashedValueResult,
			db: func() db.DBGetter {
				defaultDBGetterMock := NewMockDBGetter(ctrl)
				defaultDBGetterMock.EXPECT().Get(gomock.Any()).Return(hashedValueResult, nil).Times(1)

				return defaultDBGetterMock
			}(),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Check no mutation was done
			copySettings := node.DeepCopySettings
			var expectedParent *node.Node
			if testCase.parent != nil {
				expectedParent = testCase.parent.Copy(copySettings)
			}

			value := retrieve(testCase.db, testCase.parent, testCase.key)

			assert.Equal(t, testCase.value, value)
			assert.Equal(t, expectedParent, testCase.parent)
		})
	}
}

func Test_Trie_ClearPrefixLimit(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         InMemoryTrie
		prefix       []byte
		limit        uint32
		deleted      uint32
		allDeleted   bool
		errSentinel  error
		errMessage   string
		expectedTrie InMemoryTrie
	}{
		"limit_is_zero": {},
		"clear_prefix_limit": {
			trie: InMemoryTrie{
				root: &node.Node{
					PartialKey:   []byte{1, 2},
					StorageValue: []byte{1},
					Descendants:  1,
					Children: padRightChildren([]*node.Node{
						nil, nil, nil,
						{
							PartialKey:   []byte{4},
							StorageValue: []byte{1},
						},
					}),
				},
			},
			prefix:     []byte{0x12},
			limit:      5,
			deleted:    2,
			allDeleted: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie

			deleted, allDeleted, err := trie.ClearPrefixLimit(testCase.prefix, testCase.limit)

			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.deleted, deleted)
			assert.Equal(t, testCase.allDeleted, allDeleted)
			assert.Equal(t, testCase.expectedTrie, trie)
		})
	}
}

func Test_Trie_clearPrefixLimitAtNode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                  InMemoryTrie
		parent                *node.Node
		prefix                []byte
		limit                 uint32
		pendingDeltas         tracking.DeltaRecorder
		newParent             *node.Node
		valuesDeleted         uint32
		nodesRemoved          uint32
		allDeleted            bool
		errSentinel           error
		errMessage            string
		expectedPendingDeltas tracking.DeltaRecorder
	}{
		"limit_is_zero": {
			allDeleted: true,
		},
		"nil_parent": {
			limit:      1,
			allDeleted: true,
		},
		"leaf_parent_with_common_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			prefix:        []byte{1},
			limit:         1,
			valuesDeleted: 1,
			nodesRemoved:  1,
			allDeleted:    true,
		},
		"leaf_parent_with_key_equal_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			prefix:        []byte{1},
			limit:         1,
			valuesDeleted: 1,
			nodesRemoved:  1,
			allDeleted:    true,
		},
		"leaf_parent_with_key_no_common_prefix": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			prefix: []byte{1, 3},
			limit:  1,
			newParent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			allDeleted: true,
		},
		"leaf_parent_with_key_smaller_than_prefix": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			prefix: []byte{1, 2},
			limit:  1,
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			allDeleted: true,
		},
		"branch_without_value_parent_with_common_prefix": {
			parent: &node.Node{
				PartialKey:  []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			prefix:        []byte{1},
			limit:         3,
			valuesDeleted: 2,
			nodesRemoved:  3,
			allDeleted:    true,
		},
		"branch_without_value_with_key_equal_prefix": {
			parent: &node.Node{
				PartialKey:  []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			prefix:        []byte{1, 2},
			limit:         3,
			valuesDeleted: 2,
			nodesRemoved:  3,
			allDeleted:    true,
		},
		"branch_without_value_with_no_common_prefix": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 3},
			limit:  1,
			newParent: &node.Node{
				PartialKey:  []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch_without_value_with_key_smaller_than_prefix_by_more_than_one": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 2, 3},
			limit:  1,
			newParent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch_without_value_with_key_smaller_than_prefix_by_one": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 2},
			limit:  1,
			newParent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch_with_value_with_common_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			prefix:        []byte{1},
			limit:         2,
			valuesDeleted: 2,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"branch_with_value_with_key_equal_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			prefix:        []byte{1, 2},
			limit:         2,
			valuesDeleted: 2,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"branch_with_value_with_no_common_prefix": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 3},
			limit:  1,
			newParent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch_with_value_with_key_smaller_than_prefix_by_more_than_one": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 2, 3},
			limit:  1,
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch_with_value_with_key_smaller_than_prefix_by_one": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 2},
			limit:  1,
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"delete_one_child_of_branch": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1},
			limit:  1,
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					nil,
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{1},
						MerkleValue:  []byte{0x41, 0x04, 0x04, 0x01},
					},
				}),
			},
			valuesDeleted: 1,
			nodesRemoved:  1,
		},
		"delete_only_child_of_branch": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			limit:  1,
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
			},
			valuesDeleted: 1,
			nodesRemoved:  1,
			allDeleted:    true,
		},
		"fully_delete_children_of_branch_with_value": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1},
			limit:  2,
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			valuesDeleted: 2,
			nodesRemoved:  2,
		},
		"fully_delete_children_of_branch_without_value": {
			parent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
				}),
			},
			prefix:        []byte{1},
			limit:         2,
			valuesDeleted: 2,
			nodesRemoved:  3,
			allDeleted:    true,
		},

		"partially_delete_child_of_branch": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  3,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 3
						PartialKey:   []byte{3},
						StorageValue: []byte{1},
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							{ // full key 1, 0, 3, 0, 5
								PartialKey:   []byte{5},
								StorageValue: []byte{1},
							},
						}),
					},
					{
						PartialKey:   []byte{6},
						StorageValue: []byte{1},
					},
				}),
			},
			prefix: []byte{1, 0},
			limit:  1,
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 3
						PartialKey:   []byte{3},
						StorageValue: []byte{1},
						Dirty:        true,
						Generation:   1,
					},
					{
						PartialKey:   []byte{6},
						StorageValue: []byte{1},
						// Not modified so same generation as before
						MerkleValue: []byte{0x41, 0x06, 0x04, 0x01},
					},
				}),
			},
			valuesDeleted: 1,
			nodesRemoved:  1,
		},
		"update_child_of_branch": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							{PartialKey: []byte{1}, StorageValue: []byte{1}},
						}),
					},
				}),
			},
			prefix: []byte{1, 0, 2},
			limit:  2,
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			valuesDeleted: 2,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"delete_one_of_two_children_of_branch_without_value": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0, 3},
			limit:  3,
			newParent: &node.Node{
				PartialKey:   []byte{1, 1, 4},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			valuesDeleted: 1,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"delete_one_of_two_children_of_branch": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			limit:  3,
			newParent: &node.Node{
				PartialKey:   []byte{1, 1, 4},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			valuesDeleted: 1,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"delete_child_of_branch_with_limit_reached": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
				}),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newParent, valuesDeleted, nodesRemoved, allDeleted, err :=
				trie.clearPrefixLimitAtNode(testCase.parent, testCase.prefix,
					testCase.limit, testCase.pendingDeltas)

			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.valuesDeleted, valuesDeleted)
			assert.Equal(t, testCase.nodesRemoved, nodesRemoved)
			assert.Equal(t, testCase.allDeleted, allDeleted)
			assert.Equal(t, expectedTrie, trie)
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
		})
	}
}

func Test_Trie_deleteNodesLimit(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                  InMemoryTrie
		parent                *node.Node
		limit                 uint32
		pendingDeltas         tracking.DeltaRecorder
		newNode               *node.Node
		valuesDeleted         uint32
		nodesRemoved          uint32
		errSentinel           error
		errMessage            string
		expectedPendingDeltas tracking.DeltaRecorder
	}{
		"zero_limit": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			newNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
		},
		"nil_parent": {
			limit: 1,
		},
		"delete_leaf": {
			parent: &node.Node{
				StorageValue: []byte{1},
			},
			limit:         2,
			valuesDeleted: 1,
			nodesRemoved:  1,
		},
		"delete_branch_without_value": {
			parent: &node.Node{
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{},
					{},
				}),
			},
			limit:         3,
			valuesDeleted: 2,
			nodesRemoved:  3,
		},
		"delete_branch_with_value": {
			parent: &node.Node{
				PartialKey:   []byte{3},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			limit:         3,
			valuesDeleted: 2,
			nodesRemoved:  2,
		},
		"delete_branch_and_all_children": {
			parent: &node.Node{
				PartialKey:  []byte{3},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			limit:         10,
			valuesDeleted: 2,
			nodesRemoved:  3,
		},
		"delete_branch_one_child_only": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{3},
				StorageValue: []byte{1, 2, 3},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			limit: 1,
			newNode: &node.Node{
				PartialKey:   []byte{3},
				StorageValue: []byte{1, 2, 3},
				Dirty:        true,
				Generation:   1,
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					nil,
					{
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
						MerkleValue:  []byte{0x41, 0x02, 0x04, 0x01},
					},
				}),
			},
			valuesDeleted: 1,
			nodesRemoved:  1,
		},
		"delete_branch_children_only": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{3},
				StorageValue: []byte{1, 2, 3},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			limit: 2,
			newNode: &node.Node{
				PartialKey:   []byte{3},
				StorageValue: []byte{1, 2, 3},
				Dirty:        true,
				Generation:   1,
			},
			valuesDeleted: 2,
			nodesRemoved:  2,
		},
		"delete_branch_all_children_except_one": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{3},
				Descendants: 3,
				Children: padRightChildren([]*node.Node{
					nil,
					{PartialKey: []byte{1}, StorageValue: []byte{1}},
					nil,
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
					nil,
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
				}),
			},
			limit: 2,
			newNode: &node.Node{
				PartialKey:   []byte{3, 5, 3},
				StorageValue: []byte{1},
				Generation:   1,
				Dirty:        true,
			},
			valuesDeleted: 2,
			nodesRemoved:  3,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newNode, valuesDeleted, nodesRemoved, err :=
				trie.deleteNodesLimit(testCase.parent,
					testCase.limit, testCase.pendingDeltas)

			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.valuesDeleted, valuesDeleted)
			assert.Equal(t, testCase.nodesRemoved, nodesRemoved)
			assert.Equal(t, expectedTrie, trie)
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
		})
	}
}

func Test_Trie_ClearPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         InMemoryTrie
		prefix       []byte
		expectedTrie InMemoryTrie
	}{
		"nil_prefix": {
			trie: InMemoryTrie{
				root:       &node.Node{StorageValue: []byte{1}},
				generation: 1,
				deltas:     newDeltas(),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0xf96a741522bcc14f0aea2f70604452241d59b5f2ddab9a6948fdb3fef5f98643"),
			},
		},
		"empty_prefix": {
			trie: InMemoryTrie{
				root:       &node.Node{StorageValue: []byte{1}},
				generation: 1,
				deltas:     newDeltas(),
			},
			prefix: []byte{},
			expectedTrie: InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0xf96a741522bcc14f0aea2f70604452241d59b5f2ddab9a6948fdb3fef5f98643"),
			},
		},
		"empty_trie": {
			prefix: []byte{0x12},
		},
		"clear_prefix": {
			trie: InMemoryTrie{
				generation: 1,
				root: &node.Node{
					PartialKey:  []byte{1, 2},
					Descendants: 3,
					Children: padRightChildren([]*node.Node{
						{ // full key in nibbles 1, 2, 0, 5
							PartialKey:   []byte{5},
							StorageValue: []byte{1},
						},
						{ // full key in nibbles 1, 2, 1, 6
							PartialKey:   []byte{6},
							StorageValue: []byte("bottom branch"),
							Children: padRightChildren([]*node.Node{
								{ // full key in nibbles 1, 2, 1, 6, 0, 7
									PartialKey:   []byte{7},
									StorageValue: []byte{1},
								},
							}),
						},
					}),
				},
				deltas: newDeltas(),
			},
			prefix: []byte{0x12, 0x16},
			expectedTrie: InMemoryTrie{
				generation: 1,
				root: &node.Node{
					PartialKey:   []byte{1, 2, 0, 5},
					StorageValue: []byte{1},
					Generation:   1,
					Dirty:        true,
				},
				deltas: newDeltas("0x5fe108c83d08329353d6918e0104dacc9d2187fd9dafa582d1c532e5fe7b2e50"),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Check for no mutation
			var expectedPrefix []byte
			if testCase.prefix != nil {
				expectedPrefix = make([]byte, len(testCase.prefix))
				copy(expectedPrefix, testCase.prefix)
			}

			testCase.trie.ClearPrefix(testCase.prefix)

			assert.Equal(t, testCase.expectedTrie, testCase.trie)
			assert.Equal(t, expectedPrefix, testCase.prefix)
		})
	}
}

func Test_Trie_clearPrefixAtNode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                  InMemoryTrie
		parent                *node.Node
		prefix                []byte
		pendingDeltas         tracking.DeltaRecorder
		newParent             *node.Node
		nodesRemoved          uint32
		expectedTrie          InMemoryTrie
		expectedPendingDeltas tracking.DeltaRecorder
	}{
		"delete_one_of_two_children_of_branch": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			newParent: &node.Node{
				PartialKey:   []byte{1, 1, 4},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			nodesRemoved: 2,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"nil_parent": {},
		"leaf_parent_with_common_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			prefix:       []byte{1},
			nodesRemoved: 1,
		},
		"leaf_parent_with_key_equal_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			prefix:       []byte{1},
			nodesRemoved: 1,
		},
		"leaf_parent_with_key_no_common_prefix": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			prefix: []byte{1, 3},
			newParent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"leaf_parent_with_key_smaller_than_prefix": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			prefix: []byte{1, 2},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_parent_with_common_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			prefix:       []byte{1},
			nodesRemoved: 2,
		},
		"branch_with_key_equal_prefix": {
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			prefix:       []byte{1, 2},
			nodesRemoved: 2,
		},
		"branch_with_no_common_prefix": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			prefix: []byte{1, 3},
			newParent: &node.Node{
				PartialKey:   []byte{1, 2},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_with_key_smaller_than_prefix_by_more_than_one": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			prefix: []byte{1, 2, 3},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_with_key_smaller_than_prefix_by_one": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			prefix: []byte{1, 2},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"delete_one_child_of_branch": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
					{PartialKey: []byte{4}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0, 3},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					nil,
					{
						PartialKey:   []byte{4},
						StorageValue: []byte{1},
						MerkleValue:  []byte{0x41, 0x04, 0x04, 0x01},
					},
				}),
			},
			nodesRemoved: 1,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"fully_delete_child_of_branch": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			nodesRemoved: 1,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"partially_delete_child_of_branch": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 3
						PartialKey:   []byte{3},
						StorageValue: []byte{1},
						Descendants:  1,
						Children: padRightChildren([]*node.Node{
							{ // full key 1, 0, 3, 0, 5
								PartialKey:   []byte{5},
								StorageValue: []byte{1},
							},
						}),
					},
				}),
			},
			prefix: []byte{1, 0, 3, 0},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 3
						PartialKey:   []byte{3},
						StorageValue: []byte{1},
						Dirty:        true,
						Generation:   1,
					},
				}),
			},
			nodesRemoved: 1,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"delete_one_of_two_children_of_branch_without_value": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{3}, StorageValue: []byte{1}}, // full key 1, 0, 3
					{PartialKey: []byte{4}, StorageValue: []byte{1}}, // full key 1, 1, 4
				}),
			},
			prefix: []byte{1, 0, 3},
			newParent: &node.Node{
				PartialKey:   []byte{1, 1, 4},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			nodesRemoved: 2,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie

			newParent, nodesRemoved, err := trie.clearPrefixAtNode(
				testCase.parent, testCase.prefix, testCase.pendingDeltas)

			require.NoError(t, err)
			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.nodesRemoved, nodesRemoved)
			assert.Equal(t, testCase.expectedTrie, trie)
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
		})
	}
}

func Test_Trie_Delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         InMemoryTrie
		key          []byte
		expectedTrie InMemoryTrie
	}{
		"nil_key": {
			trie: InMemoryTrie{
				root:       &node.Node{StorageValue: []byte{1}},
				generation: 1,
				deltas:     newDeltas(),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0xf96a741522bcc14f0aea2f70604452241d59b5f2ddab9a6948fdb3fef5f98643"),
			},
		},
		"empty_key": {
			trie: InMemoryTrie{
				root:       &node.Node{StorageValue: []byte{1}},
				generation: 1,
				deltas:     newDeltas(),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
				deltas:     newDeltas("0xf96a741522bcc14f0aea2f70604452241d59b5f2ddab9a6948fdb3fef5f98643"),
			},
		},
		"empty_trie": {
			key: []byte{0x12},
		},
		"delete_branch_node": {
			trie: InMemoryTrie{
				generation: 1,
				root: &node.Node{
					PartialKey:  []byte{1, 2},
					Descendants: 3,
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{5},
							StorageValue: []byte{97},
						},
						{ // full key in nibbles 1, 2, 1, 6
							PartialKey:   []byte{6},
							StorageValue: []byte{98},
							Descendants:  1,
							Children: padRightChildren([]*node.Node{
								{ // full key in nibbles 1, 2, 1, 6, 0, 7
									PartialKey:   []byte{7},
									StorageValue: []byte{99},
								},
							}),
						},
					}),
				},
				deltas: newDeltas(),
			},
			key: []byte{0x12, 0x16},
			expectedTrie: InMemoryTrie{
				generation: 1,
				root: &node.Node{
					PartialKey:  []byte{1, 2},
					Dirty:       true,
					Generation:  1,
					Descendants: 2,
					Children: padRightChildren([]*node.Node{
						{
							PartialKey:   []byte{5},
							StorageValue: []byte{97},
							MerkleValue:  []byte{0x41, 0x05, 0x04, 0x61},
						},
						{ // full key in nibbles 1, 2, 1, 6
							PartialKey:   []byte{6, 0, 7},
							StorageValue: []byte{99},
							Dirty:        true,
							Generation:   1,
						},
					}),
				},
				deltas: newDeltas("0x3d1b3d727ee404549a5d2531aab9fff0eeddc58bc30bfe2fe82b1a0cfe7e76d5"),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Check for no mutation
			var expectedKey []byte
			if testCase.key != nil {
				expectedKey = make([]byte, len(testCase.key))
				copy(expectedKey, testCase.key)
			}

			testCase.trie.Delete(testCase.key)

			assert.Equal(t, testCase.expectedTrie, testCase.trie)
			assert.Equal(t, expectedKey, testCase.key)
		})
	}
}

func Test_Trie_deleteAtNode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                  InMemoryTrie
		parent                *node.Node
		key                   []byte
		pendingDeltas         tracking.DeltaRecorder
		newParent             *node.Node
		updated               bool
		nodesRemoved          uint32
		errSentinel           error
		errMessage            string
		expectedTrie          InMemoryTrie
		expectedPendingDeltas tracking.DeltaRecorder
	}{
		"nil_parent": {
			key: []byte{1},
		},
		"leaf_parent_and_nil_key": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			updated:      true,
			nodesRemoved: 1,
		},
		"leaf_parent_and_empty_key": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			key:          []byte{},
			updated:      true,
			nodesRemoved: 1,
		},
		"leaf_parent_matches_key": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			key:          []byte{1},
			updated:      true,
			nodesRemoved: 1,
		},
		"leaf_parent_mismatches_key": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			key: []byte{2},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_parent_and_nil_key": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
					},
				}),
			},
			newParent: &node.Node{
				PartialKey:   []byte{1, 0, 2},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			updated:      true,
			nodesRemoved: 1,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_parent_and_empty_key": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			key: []byte{},
			newParent: &node.Node{
				PartialKey:   []byte{1, 0, 2},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			updated:      true,
			nodesRemoved: 1,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_parent_matches_key": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			key: []byte{1},
			newParent: &node.Node{
				PartialKey:   []byte{1, 0, 2},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			updated:      true,
			nodesRemoved: 1,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_parent_child_matches_key": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
					},
				}),
			},
			key: []byte{1, 0, 2},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Dirty:        true,
				Generation:   1,
			},
			updated:      true,
			nodesRemoved: 1,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_parent_mismatches_key": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			key: []byte{2},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{},
				}),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"branch_parent_child_mismatches_key": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
					},
				}),
			},
			key: []byte{1, 0, 3},
			newParent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  1,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
					},
				}),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"delete_branch_child_and_merge_branch_and_left_child": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
					},
					{ // full key 1, 1, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{2},
					},
				}),
			},
			key: []byte{1, 0, 2},
			newParent: &node.Node{
				PartialKey:   []byte{1, 1, 2},
				StorageValue: []byte{2},
				Dirty:        true,
				Generation:   1,
			},
			updated:      true,
			nodesRemoved: 2,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"delete_branch_and_keep_two_children": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{1},
				Descendants:  2,
				Children: padRightChildren([]*node.Node{
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
					{PartialKey: []byte{2}, StorageValue: []byte{1}},
				}),
			},
			key: []byte{1},
			newParent: &node.Node{
				PartialKey:  []byte{1},
				Generation:  1,
				Dirty:       true,
				Descendants: 2,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
						MerkleValue:  []byte{0x41, 0x02, 0x04, 0x01},
					},
					{
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
						MerkleValue:  []byte{0x41, 0x02, 0x04, 0x01},
					},
				}),
			},
			updated: true,
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
		"handle_nonexistent_key_(no_op)": {
			trie: InMemoryTrie{
				generation: 1,
			},
			parent: &node.Node{
				PartialKey:  []byte{1, 0, 2, 3},
				Descendants: 1,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
					},
					{ // full key 1, 1, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{2},
					},
				}),
			},
			key: []byte{1, 0, 2},
			newParent: &node.Node{
				PartialKey:  []byte{1, 0, 2, 3},
				Descendants: 1,
				Children: padRightChildren([]*node.Node{
					{ // full key 1, 0, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{1},
					},
					{ // full key 1, 1, 2
						PartialKey:   []byte{2},
						StorageValue: []byte{2},
					},
				}),
			},
			expectedTrie: InMemoryTrie{
				generation: 1,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Check for no mutation
			var expectedKey []byte
			if testCase.key != nil {
				expectedKey = make([]byte, len(testCase.key))
				copy(expectedKey, testCase.key)
			}

			newParent, updated, nodesRemoved, err := testCase.trie.deleteAtNode(
				testCase.parent, testCase.key, testCase.pendingDeltas)

			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.updated, updated)
			assert.Equal(t, testCase.nodesRemoved, nodesRemoved)
			assert.Equal(t, testCase.expectedTrie, testCase.trie)
			assert.Equal(t, expectedKey, testCase.key)
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
		})
	}
}

func Test_Trie_handleDeletion(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                  InMemoryTrie
		branch                *node.Node
		deletedKey            []byte
		pendingDeltas         tracking.DeltaRecorder
		newNode               *node.Node
		branchChildMerged     bool
		errSentinel           error
		errMessage            string
		expectedPendingDeltas tracking.DeltaRecorder
	}{
		"branch_with_value_and_without_children": {
			branch: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{5, 6, 7},
				Generation:   1,
			},
			deletedKey: []byte{1, 2, 3, 4},
			newNode: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{5, 6, 7},
				Generation:   1,
				Dirty:        true,
			},
		},
		// branch without value and without children cannot happen
		// since it would be turned into a leaf when it only has one child
		// remaining.
		"branch_with_value_and_a_single_child": {
			branch: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{5, 6, 7},
				Generation:   1,
				Children: padRightChildren([]*node.Node{
					nil,
					{PartialKey: []byte{9}, StorageValue: []byte{1}},
				}),
			},
			newNode: &node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{5, 6, 7},
				Generation:   1,
				Children: padRightChildren([]*node.Node{
					nil,
					{PartialKey: []byte{9}, StorageValue: []byte{1}},
				}),
			},
		},
		"branch_without_value_and_a_single_leaf_child": {
			branch: &node.Node{
				PartialKey: []byte{1, 2, 3},
				Generation: 1,
				Children: padRightChildren([]*node.Node{
					nil,
					{ // full key 1,2,3,1,9
						PartialKey:   []byte{9},
						StorageValue: []byte{10},
					},
				}),
			},
			deletedKey: []byte{1, 2, 3, 4},
			newNode: &node.Node{
				PartialKey:   []byte{1, 2, 3, 1, 9},
				StorageValue: []byte{10},
				Generation:   1,
				Dirty:        true,
			},
			branchChildMerged: true,
		},
		"branch_without_value_and_a_single_branch_child": {
			branch: &node.Node{
				PartialKey: []byte{1, 2, 3},
				Generation: 1,
				Children: padRightChildren([]*node.Node{
					nil,
					{
						PartialKey:   []byte{9},
						StorageValue: []byte{10},
						Children: padRightChildren([]*node.Node{
							{PartialKey: []byte{7}, StorageValue: []byte{1}},
							nil,
							{PartialKey: []byte{8}, StorageValue: []byte{1}},
						}),
					},
				}),
			},
			newNode: &node.Node{
				PartialKey:   []byte{1, 2, 3, 1, 9},
				StorageValue: []byte{10},
				Generation:   1,
				Dirty:        true,
				Children: padRightChildren([]*node.Node{
					{
						PartialKey:   []byte{7},
						StorageValue: []byte{1},
						MerkleValue:  []byte{0x41, 0x07, 0x04, 0x01},
					},
					nil,
					{
						PartialKey:   []byte{8},
						StorageValue: []byte{1},
						MerkleValue:  []byte{0x41, 0x08, 0x04, 0x01},
					},
				}),
			},
			branchChildMerged: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Check for no mutation
			var expectedKey []byte
			if testCase.deletedKey != nil {
				expectedKey = make([]byte, len(testCase.deletedKey))
				copy(expectedKey, testCase.deletedKey)
			}

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newNode, branchChildMerged, err := trie.handleDeletion(
				testCase.branch, testCase.deletedKey, testCase.pendingDeltas)

			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}

			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.branchChildMerged, branchChildMerged)
			assert.Equal(t, expectedKey, testCase.deletedKey)
			assert.Equal(t, testCase.expectedPendingDeltas, testCase.pendingDeltas)
			assert.Equal(t, expectedTrie, trie)
		})
	}
}

func Test_Trie_ensureMerkleValueIsCalculated(t *testing.T) {
	t.Parallel()

	n := &node.Node{
		PartialKey:   []byte{1},
		StorageValue: []byte{2},
	}

	nodeWithEncodingMerkleValue := &node.Node{
		PartialKey:   []byte{1},
		StorageValue: []byte{2},
		MerkleValue:  []byte{3},
	}

	nodeWithHashMerkleValue := &node.Node{
		PartialKey:   []byte{1},
		StorageValue: []byte{2},
		MerkleValue: []byte{
			1, 2, 3, 4, 5, 6, 7, 8,
			1, 2, 3, 4, 5, 6, 7, 8,
			1, 2, 3, 4, 5, 6, 7, 8,
			1, 2, 3, 4, 5, 6, 7, 8},
	}

	testCases := map[string]struct {
		trie         InMemoryTrie
		parent       *node.Node
		errSentinel  error
		errMessage   string
		expectedNode *node.Node
		expectedTrie InMemoryTrie
	}{
		"nil_parent": {},
		"root_node_without_Merkle_value": {
			trie: InMemoryTrie{
				root: n,
			},
			parent: n,
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				MerkleValue: []byte{
					0x60, 0x51, 0x6d, 0xb, 0xb6, 0xe1, 0xbb, 0xfb,
					0x12, 0x93, 0xf1, 0xb2, 0x76, 0xea, 0x95, 0x5,
					0xe9, 0xf4, 0xa4, 0xe7, 0xd9, 0x8f, 0x62, 0xd,
					0x5, 0x11, 0x5e, 0xb, 0x85, 0x27, 0x4a, 0xe1},
			},
			expectedTrie: InMemoryTrie{
				root: n,
			},
		},
		"root_node_with_inlined_Merkle_value": {
			trie: InMemoryTrie{
				root: nodeWithEncodingMerkleValue,
			},
			parent: nodeWithEncodingMerkleValue,
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				MerkleValue: []byte{
					0x60, 0x51, 0x6d, 0xb, 0xb6, 0xe1, 0xbb, 0xfb,
					0x12, 0x93, 0xf1, 0xb2, 0x76, 0xea, 0x95, 0x5,
					0xe9, 0xf4, 0xa4, 0xe7, 0xd9, 0x8f, 0x62, 0xd,
					0x5, 0x11, 0x5e, 0xb, 0x85, 0x27, 0x4a, 0xe1},
			},
			expectedTrie: InMemoryTrie{
				root: nodeWithEncodingMerkleValue,
			},
		},
		"root_node_with_hash_Merkle_value": {
			trie: InMemoryTrie{
				root: nodeWithHashMerkleValue,
			},
			parent: nodeWithHashMerkleValue,
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				MerkleValue: []byte{
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8,
					1, 2, 3, 4, 5, 6, 7, 8},
			},
			expectedTrie: InMemoryTrie{
				root: nodeWithHashMerkleValue,
			},
		},
		"non_root_node_without_Merkle_value": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				MerkleValue:  []byte{0x41, 0x1, 0x4, 0x2},
			},
		},
		"non_root_node_with_Merkle_value": {
			parent: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				MerkleValue:  []byte{3},
			},
			expectedNode: &node.Node{
				PartialKey:   []byte{1},
				StorageValue: []byte{2},
				MerkleValue:  []byte{3},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := testCase.trie.ensureMerkleValueIsCalculated(testCase.parent)

			checkMerkleValuesAreSet(t, testCase.parent)
			assert.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.expectedNode, testCase.parent)
			assert.Equal(t, testCase.expectedTrie, testCase.trie)
		})
	}
}

func Test_lenCommonPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		a      []byte
		b      []byte
		length int
	}{
		"nil_slices": {},
		"empty_slices": {
			a: []byte{},
			b: []byte{},
		},
		"fully_different": {
			a: []byte{1, 2, 3},
			b: []byte{4, 5, 6},
		},
		"fully_same": {
			a:      []byte{1, 2, 3},
			b:      []byte{1, 2, 3},
			length: 3,
		},
		"different_and_common_prefix": {
			a:      []byte{1, 2, 3, 4},
			b:      []byte{1, 2, 4, 4},
			length: 2,
		},
		"first_bigger_than_second": {
			a:      []byte{1, 2, 3},
			b:      []byte{1, 2},
			length: 2,
		},
		"first_smaller_than_second": {
			a:      []byte{1, 2},
			b:      []byte{1, 2, 3},
			length: 2,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			length := lenCommonPrefix(testCase.a, testCase.b)

			assert.Equal(t, testCase.length, length)
		})
	}
}

func Test_concatenateSlices(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		sliceOne     []byte
		sliceTwo     []byte
		otherSlices  [][]byte
		concatenated []byte
	}{
		"two_nil_slices": {},
		"four_nil_slices": {
			otherSlices: [][]byte{nil, nil},
		},
		"only_fourth_slice_not_nil": {
			otherSlices: [][]byte{
				nil,
				{1},
			},
			concatenated: []byte{1},
		},
		"two_empty_slices": {
			sliceOne:     []byte{},
			sliceTwo:     []byte{},
			concatenated: []byte{},
		},
		"three_empty_slices": {
			sliceOne:     []byte{},
			sliceTwo:     []byte{},
			otherSlices:  [][]byte{{}},
			concatenated: []byte{},
		},
		"concatenate_two_first_slices": {
			sliceOne:     []byte{1, 2},
			sliceTwo:     []byte{3, 4},
			concatenated: []byte{1, 2, 3, 4},
		},

		"concatenate_four_slices": {
			sliceOne: []byte{1, 2},
			sliceTwo: []byte{3, 4},
			otherSlices: [][]byte{
				{5, 6},
				{7, 8},
			},
			concatenated: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			concatenated := concatenateSlices(testCase.sliceOne,
				testCase.sliceTwo, testCase.otherSlices...)

			assert.Equal(t, testCase.concatenated, concatenated)
		})
	}
}

func Benchmark_concatSlices(b *testing.B) {
	const sliceSize = 100000 // 100KB
	slice1 := make([]byte, sliceSize)
	slice2 := make([]byte, sliceSize)

	// 16993 ns/op	  245760 B/op	       1 allocs/op
	b.Run("direct append", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			concatenated := append(slice1, slice2...) //skipcq: CRT-D0001
			concatenated[0] = 1
		}
	})

	// 16340 ns/op	  204800 B/op	       1 allocs/op
	b.Run("append with pre-allocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			concatenated := make([]byte, 0, len(slice1)+len(slice2))
			concatenated = append(concatenated, slice1...)
			concatenated = append(concatenated, slice2...)
			concatenated[0] = 1
		}
	})

	// 16453 ns/op	  204800 B/op	       1 allocs/op
	b.Run("concatenation helper function", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			concatenated := concatenateSlices(slice1, slice2)
			concatenated[0] = 1
		}
	})

	// 16453 ns/op	  204800 B/op	       1 allocs/op
	b.Run("bytes.Join", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			concatenated := bytes.Join([][]byte{slice1, slice2}, nil)
			concatenated[0] = 1
		}
	})
}

func TestTrieVersionAndMustHash(t *testing.T) {
	newTrie := NewEmptyTrie()

	// setting trie version to 0
	// no entry should be hashed (no matter its size)
	newTrie.SetVersion(0)

	type testStruct struct {
		key          []byte
		nibbles      []byte
		storageValue []byte
		mustBeHashed bool
	}

	testCases := []testStruct{
		{
			key:          []byte{1, 2, 3, 4},
			nibbles:      codec.KeyLEToNibbles([]byte{1, 2, 3, 4}),
			storageValue: make([]byte, 66),
			mustBeHashed: false,
		},
		{
			key:          []byte{2, 4, 5, 6},
			nibbles:      codec.KeyLEToNibbles([]byte{2, 4, 5, 6}),
			storageValue: make([]byte, 66),
			mustBeHashed: false,
		},
	}

	// inserting the key and values to the trie
	for _, tt := range testCases {
		require.NoError(
			t,
			newTrie.Put(tt.key, tt.storageValue),
		)
	}

	// asserting each trie node
	for _, tt := range testCases {
		node := findNode(t, newTrie.root, tt.nibbles)
		require.NotNil(t, node)
		require.Equal(t, node.MustBeHashed, tt.mustBeHashed)
	}

	// setting trie version to 1 a new inserted node
	// with storage value greater than 32 should be marked as MustBeHashed
	newTrie.SetVersion(1)

	nodeCKey := []byte{9, 8, 7, 5}
	nodeDKey := []byte{4, 4, 7, 2}
	nodeEKey := []byte{6, 7, 0xa, 0xb}

	require.NoError(
		t,
		newTrie.Put(nodeCKey, make([]byte, 66)),
	)

	require.NoError(
		t,
		newTrie.Put(nodeDKey, make([]byte, 66)),
	)

	require.NoError(
		t,
		newTrie.Put(nodeEKey, make([]byte, 10)),
	)

	testCases = append(testCases, testStruct{nibbles: codec.KeyLEToNibbles(nodeCKey), mustBeHashed: true})
	testCases = append(testCases, testStruct{nibbles: codec.KeyLEToNibbles(nodeDKey), mustBeHashed: true})
	testCases = append(testCases, testStruct{nibbles: codec.KeyLEToNibbles(nodeEKey), mustBeHashed: false})

	for _, tt := range testCases {
		node := findNode(t, newTrie.root, tt.nibbles)
		require.NotNil(t, node)
		require.Equal(t, tt.mustBeHashed, node.MustBeHashed)
	}
}

func findNode(t *testing.T, currNode *node.Node, nibbles []byte) *node.Node {
	t.Helper()

	if bytes.Equal(currNode.PartialKey, nibbles) {
		return currNode
	}

	if currNode.Kind() == node.Leaf {
		return nil
	}

	commonLen := lenCommonPrefix(currNode.PartialKey, nibbles)
	child := currNode.Children[nibbles[commonLen]]
	if child == nil {
		return nil
	}

	return findNode(t, child, nibbles[commonLen+1:])
}
