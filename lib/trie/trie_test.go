// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewEmptyTrie(t *testing.T) {
	expectedTrie := &Trie{
		childTries:          make(map[common.Hash]*Trie),
		deletedMerkleValues: map[string]struct{}{},
	}
	trie := NewEmptyTrie()
	assert.Equal(t, expectedTrie, trie)
}

func Test_NewTrie(t *testing.T) {
	root := &Node{
		Key:      []byte{0},
		SubValue: []byte{17},
	}
	expectedTrie := &Trie{
		root: &Node{
			Key:      []byte{0},
			SubValue: []byte{17},
		},
		childTries:          make(map[common.Hash]*Trie),
		deletedMerkleValues: map[string]struct{}{},
	}
	trie := NewTrie(root)
	assert.Equal(t, expectedTrie, trie)
}

func Test_Trie_Snapshot(t *testing.T) {
	t.Parallel()

	trie := &Trie{
		generation: 8,
		root:       &Node{Key: []byte{8}, SubValue: []byte{1}},
		childTries: map[common.Hash]*Trie{
			{1}: {
				generation: 1,
				root:       &Node{Key: []byte{1}, SubValue: []byte{1}},
				deletedMerkleValues: map[string]struct{}{
					"a": {},
				},
			},
			{2}: {
				generation: 2,
				root:       &Node{Key: []byte{2}, SubValue: []byte{1}},
				deletedMerkleValues: map[string]struct{}{
					"b": {},
				},
			},
		},
		deletedMerkleValues: map[string]struct{}{
			"a": {},
			"b": {},
		},
	}

	expectedTrie := &Trie{
		generation: 9,
		root:       &Node{Key: []byte{8}, SubValue: []byte{1}},
		childTries: map[common.Hash]*Trie{
			{1}: {
				generation:          2,
				root:                &Node{Key: []byte{1}, SubValue: []byte{1}},
				deletedMerkleValues: map[string]struct{}{},
			},
			{2}: {
				generation:          3,
				root:                &Node{Key: []byte{2}, SubValue: []byte{1}},
				deletedMerkleValues: map[string]struct{}{},
			},
		},
		deletedMerkleValues: map[string]struct{}{},
	}

	newTrie := trie.Snapshot()

	assert.Equal(t, expectedTrie.childTries, newTrie.childTries)
}

func Test_Trie_updateGeneration(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trieGeneration              uint64
		node                        *Node
		copySettings                node.CopySettings
		newNode                     *Node
		copied                      bool
		expectedDeletedMerkleValues map[string]struct{}
	}{
		"trie generation higher and empty hash": {
			trieGeneration: 2,
			node: &Node{
				Generation: 1,
				Key:        []byte{1},
			},
			copySettings: node.DefaultCopySettings,
			newNode: &Node{
				Generation: 2,
				Key:        []byte{1},
			},
			copied:                      true,
			expectedDeletedMerkleValues: map[string]struct{}{},
		},
		"trie generation higher and hash": {
			trieGeneration: 2,
			node: &Node{
				Generation:  1,
				Key:         []byte{1},
				MerkleValue: []byte{1, 2, 3},
			},
			copySettings: node.DefaultCopySettings,
			newNode: &Node{
				Generation: 2,
				Key:        []byte{1},
			},
			copied: true,
			expectedDeletedMerkleValues: map[string]struct{}{
				string([]byte{1, 2, 3}): {},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deletedMerkleValues := make(map[string]struct{})

			newNode := updateGeneration(testCase.node, testCase.trieGeneration,
				deletedMerkleValues, testCase.copySettings)

			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.expectedDeletedMerkleValues, deletedMerkleValues)

			// Check for deep copy
			if newNode != nil && testCase.copied {
				if newNode.Dirty {
					newNode.SetClean()
				} else {
					newNode.SetDirty()
				}
				assert.NotEqual(t, testCase.node, newNode)
			}
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
func testTrieForDeepCopy(t *testing.T, original, copy *Trie) {
	assertPointersNotEqual(t, original, copy)
	if original == nil {
		return
	}
	assertPointersNotEqual(t, original.generation, copy.generation)
	assertPointersNotEqual(t, original.deletedMerkleValues, copy.deletedMerkleValues)
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
		trieOriginal *Trie
		trieCopy     *Trie
	}{
		"nil": {},
		"empty trie": {
			trieOriginal: &Trie{},
			trieCopy:     &Trie{},
		},
		"filled trie": {
			trieOriginal: &Trie{
				generation: 1,
				root:       &Node{Key: []byte{1, 2}, SubValue: []byte{1}},
				childTries: map[common.Hash]*Trie{
					{1, 2, 3}: {
						generation: 2,
						root:       &Node{Key: []byte{1}, SubValue: []byte{1}},
						deletedMerkleValues: map[string]struct{}{
							"a": {},
							"b": {},
						},
					},
				},
				deletedMerkleValues: map[string]struct{}{
					"a": {},
					"b": {},
				},
			},
			trieCopy: &Trie{
				generation: 1,
				root:       &Node{Key: []byte{1, 2}, SubValue: []byte{1}},
				childTries: map[common.Hash]*Trie{
					{1, 2, 3}: {
						generation: 2,
						root:       &Node{Key: []byte{1}, SubValue: []byte{1}},
						deletedMerkleValues: map[string]struct{}{
							"a": {},
							"b": {},
						},
					},
				},
				deletedMerkleValues: map[string]struct{}{
					"a": {},
					"b": {},
				},
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

	trie := Trie{
		root: &Node{
			Key:      []byte{1, 2, 3},
			SubValue: []byte{1},
		},
	}
	expectedRoot := &Node{
		Key:      []byte{1, 2, 3},
		SubValue: []byte{1},
	}

	root := trie.RootNode()

	assert.Equal(t, expectedRoot, root)
}

//go:generate mockgen -destination=buffer_mock_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/internal/trie/node Buffer

func Test_encodeRoot(t *testing.T) {
	t.Parallel()

	type bufferCalls struct {
		writeCalls  []writeCall
		lenCall     bool
		lenReturn   int
		bytesCall   bool
		bytesReturn []byte
	}

	testCases := map[string]struct {
		root         *Node
		bufferCalls  bufferCalls
		errWrapped   error
		errMessage   string
		expectedRoot *Node
	}{
		"nil root and no error": {
			bufferCalls: bufferCalls{
				writeCalls: []writeCall{
					{written: []byte{0}},
				},
			},
		},
		"nil root and write error": {
			bufferCalls: bufferCalls{
				writeCalls: []writeCall{
					{
						written: []byte{0},
						err:     errTest,
					},
				},
			},
			errWrapped: errTest,
			errMessage: "cannot write nil root node to buffer: test error",
		},
		"root encoding error": {
			root: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
			bufferCalls: bufferCalls{
				writeCalls: []writeCall{
					{
						written: []byte{66},
						err:     errTest,
					},
				},
			},
			errWrapped: errTest,
			errMessage: "cannot encode header: test error",
			expectedRoot: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
		},
		"root encoding success": {
			root: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
			bufferCalls: bufferCalls{
				writeCalls: []writeCall{
					{written: []byte{66}},
					{written: []byte{18}},
					{written: []byte{4, 1}},
				},
				lenCall:     true,
				lenReturn:   3,
				bytesCall:   true,
				bytesReturn: []byte{66, 18, 4, 1},
			},
			expectedRoot: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
				Encoding: []byte{66, 18, 4},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			buffer := NewMockBuffer(ctrl)

			var previousCall *gomock.Call
			for _, write := range testCase.bufferCalls.writeCalls {
				call := buffer.EXPECT().
					Write(write.written).
					Return(write.n, write.err)

				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}
			if testCase.bufferCalls.lenCall {
				buffer.EXPECT().Len().
					Return(testCase.bufferCalls.lenReturn)
			}
			if testCase.bufferCalls.bytesCall {
				buffer.EXPECT().Bytes().
					Return(testCase.bufferCalls.bytesReturn)
			}

			err := encodeRoot(testCase.root, buffer)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.expectedRoot, testCase.root)
		})
	}
}

func Test_Trie_MustHash(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var trie Trie

		hash := trie.MustHash()

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
		trie         Trie
		hash         common.Hash
		errWrapped   error
		errMessage   string
		expectedTrie Trie
	}{
		"nil root": {
			hash: common.Hash{
				0x3, 0x17, 0xa, 0x2e, 0x75, 0x97, 0xb7, 0xb7,
				0xe3, 0xd8, 0x4c, 0x5, 0x39, 0x1d, 0x13, 0x9a,
				0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0,
				0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14},
		},
		"leaf root": {
			trie: Trie{
				root: &Node{
					Key:      []byte{1, 2, 3},
					SubValue: []byte{1},
				},
			},
			hash: common.Hash{
				0xa8, 0x13, 0x7c, 0xee, 0xb4, 0xad, 0xea, 0xac,
				0x9e, 0x5b, 0x37, 0xe2, 0x8e, 0x7d, 0x64, 0x78,
				0xac, 0xba, 0xb0, 0x6e, 0x90, 0x76, 0xe4, 0x67,
				0xa1, 0xd8, 0xa2, 0x29, 0x4e, 0x4a, 0xd9, 0xa3},
			expectedTrie: Trie{
				root: &Node{
					Key:      []byte{1, 2, 3},
					SubValue: []byte{1},
					Encoding: []byte{0x43, 0x01, 0x23, 0x04, 0x01},
				},
			},
		},
		"branch root": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1, 2, 3},
					SubValue:    []byte("branch"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						{Key: []byte{9}, SubValue: []byte{1}},
					}),
				},
			},
			hash: common.Hash{
				0xaa, 0x7e, 0x57, 0x48, 0xb0, 0x27, 0x4d, 0x18,
				0xf5, 0x1c, 0xfd, 0x36, 0x4c, 0x4b, 0x56, 0x4a,
				0xf5, 0x37, 0x9d, 0xd7, 0xcb, 0xf5, 0x80, 0x15,
				0xf0, 0xe, 0xd3, 0x39, 0x48, 0x21, 0xe3, 0xdd},
			expectedTrie: Trie{
				root: &Node{
					Key:         []byte{1, 2, 3},
					SubValue:    []byte("branch"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						{
							Key:         []byte{9},
							SubValue:    []byte{1},
							Encoding:    []byte{0x41, 0x09, 0x04, 0x01},
							MerkleValue: []byte{0x41, 0x09, 0x04, 0x01},
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

			hash, err := testCase.trie.Hash()

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

	t.Run("simple root", func(t *testing.T) {
		t.Parallel()

		root := &Node{
			Key:         []byte{0xa},
			SubValue:    []byte("root"),
			Descendants: 2,
			Children: padRightChildren([]*Node{
				{ // index 0
					Key:      []byte{2, 0xb},
					SubValue: []byte("leaf"),
				},
				nil,
				{ // index 2
					Key:      []byte{0xb},
					SubValue: []byte("leaf"),
				},
			}),
		}

		trie := NewTrie(root)

		entries := trie.Entries()

		expectedEntries := map[string][]byte{
			string([]byte{0x0a}):       []byte("root"),
			string([]byte{0xa0, 0x2b}): []byte("leaf"),
			string([]byte{0x0a, 0x2b}): []byte("leaf"),
		}

		entriesMatch(t, expectedEntries, entries)
	})

	t.Run("custom root", func(t *testing.T) {
		t.Parallel()

		root := &Node{
			Key:         []byte{0xa, 0xb},
			SubValue:    []byte("root"),
			Descendants: 5,
			Children: padRightChildren([]*Node{
				nil, nil, nil,
				{ // branch with value at child index 3
					Key:         []byte{0xb},
					SubValue:    []byte("branch 1"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						nil, nil, nil,
						{ // leaf at child index 3
							Key:      []byte{0xc},
							SubValue: []byte("bottom leaf"),
						},
					}),
				},
				nil, nil, nil,
				{ // leaf at child index 7
					Key:      []byte{0xd},
					SubValue: []byte("top leaf"),
				},
				nil,
				{ // branch without value at child index 9
					Key:         []byte{0xe},
					SubValue:    []byte("branch 2"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						{ // leaf at child index 0
							Key:      []byte{0xf},
							SubValue: []byte("bottom leaf 2"),
						}, nil, nil,
					}),
				},
			}),
		}

		trie := NewTrie(root)

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

	t.Run("end to end", func(t *testing.T) {
		t.Parallel()

		trie := Trie{
			root:       nil,
			childTries: make(map[common.Hash]*Trie),
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
}

func Test_Trie_NextKey(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie    Trie
		key     []byte
		nextKey []byte
	}{
		"nil root and nil key returns nil": {},
		"nil root returns nil": {
			key: []byte{2},
		},
		"nil key returns root leaf": {
			trie: Trie{
				root: &Node{
					Key:      []byte{2},
					SubValue: []byte{1},
				},
			},
			nextKey: []byte{2},
		},
		"key smaller than root leaf full key": {
			trie: Trie{
				root: &Node{
					Key:      []byte{2},
					SubValue: []byte{1},
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

			nextKey := testCase.trie.NextKey(testCase.key)

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
		trie    Trie
		key     []byte
		nextKey []byte
	}{
		"nil root and nil key returns nil": {},
		"nil root returns nil": {
			key: []byte{2},
		},
		"nil key returns root leaf": {
			trie: Trie{
				root: &Node{
					Key:      []byte{2},
					SubValue: []byte{1},
				},
			},
			nextKey: []byte{2},
		},
		"key smaller than root leaf full key": {
			trie: Trie{
				root: &Node{
					Key:      []byte{2},
					SubValue: []byte{1},
				},
			},
			key:     []byte{1},
			nextKey: []byte{2},
		},
		"key equal to root leaf full key": {
			trie: Trie{
				root: &Node{
					Key:      []byte{2},
					SubValue: []byte{1},
				},
			},
			key: []byte{2},
		},
		"key greater than root leaf full key": {
			trie: Trie{
				root: &Node{
					Key:      []byte{2},
					SubValue: []byte{1},
				},
			},
			key: []byte{3},
		},
		"key smaller than root branch full key": {
			trie: Trie{
				root: &Node{
					Key:         []byte{2},
					SubValue:    []byte("branch"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						{
							Key:      []byte{1},
							SubValue: []byte{1},
						},
					}),
				},
			},
			key:     []byte{1},
			nextKey: []byte{2},
		},
		"key equal to root branch full key": {
			trie: Trie{
				root: &Node{
					Key:         []byte{2},
					SubValue:    []byte("branch"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						{
							Key:      []byte{1},
							SubValue: []byte{1},
						},
					}),
				},
			},
			key: []byte{2, 0, 1},
		},
		"key smaller than leaf full key": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1},
					SubValue:    []byte("branch"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							Key:      []byte{3},
							SubValue: []byte{1},
						},
					}),
				},
			},
			key:     []byte{1, 2, 2},
			nextKey: []byte{1, 2, 3},
		},
		"key equal to leaf full key": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1},
					SubValue:    []byte("branch"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							Key:      []byte{3},
							SubValue: []byte{1},
						},
					}),
				},
			},
			key: []byte{1, 2, 3},
		},
		"key greater than leaf full key": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1},
					SubValue:    []byte("branch"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							Key:      []byte{3},
							SubValue: []byte{1},
						},
					}),
				},
			},
			key: []byte{1, 2, 4},
		},
		"next key branch with value": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1},
					SubValue:    []byte("top branch"),
					Descendants: 2,
					Children: padRightChildren([]*Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							Key:         []byte{3},
							SubValue:    []byte("branch 1"),
							Descendants: 1,
							Children: padRightChildren([]*Node{
								nil, nil, nil, nil,
								{
									// full key [1, 2, 3, 4, 5]
									Key:      []byte{0x5},
									SubValue: []byte("bottom leaf"),
								},
							}),
						},
					}),
				},
			},
			key:     []byte{1},
			nextKey: []byte{1, 2, 3},
		},
		"next key go through branch without value": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1},
					Descendants: 2,
					Children: padRightChildren([]*Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							Key:         []byte{3},
							Descendants: 1,
							Children: padRightChildren([]*Node{
								nil, nil, nil, nil,
								{
									// full key [1, 2, 3, 4, 5]
									Key:      []byte{0x5},
									SubValue: []byte("bottom leaf"),
								},
							}),
						},
					}),
				},
			},
			key:     []byte{0},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"next key leaf from bottom branch": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1},
					Descendants: 2,
					Children: padRightChildren([]*Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							Key:         []byte{3},
							SubValue:    []byte("bottom branch"),
							Descendants: 1,
							Children: padRightChildren([]*Node{
								nil, nil, nil, nil,
								{
									// full key [1, 2, 3, 4, 5]
									Key:      []byte{0x5},
									SubValue: []byte("bottom leaf"),
								},
							}),
						},
					}),
				},
			},
			key:     []byte{1, 2, 3},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"next key greater than branch": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1},
					Descendants: 2,
					Children: padRightChildren([]*Node{
						nil, nil,
						{
							// full key [1, 2, 3]
							Key:         []byte{3},
							SubValue:    []byte("bottom branch"),
							Descendants: 1,
							Children: padRightChildren([]*Node{
								nil, nil, nil, nil,
								{
									// full key [1, 2, 3, 4, 5]
									Key:      []byte{0x5},
									SubValue: []byte("bottom leaf"),
								},
							}),
						},
					}),
				},
			},
			key:     []byte{1, 2, 3},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"key smaller length and greater than root branch full key": {
			trie: Trie{
				root: &Node{
					Key:         []byte{2, 0},
					SubValue:    []byte("branch"),
					Descendants: 1,
					Children: padRightChildren([]*Node{
						{Key: []byte{1}, SubValue: []byte{1}},
					}),
				},
			},
			key: []byte{3},
		},
		"key smaller length and greater than root leaf full key": {
			trie: Trie{
				root: &Node{
					Key:      []byte{2, 0},
					SubValue: []byte("leaf"),
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

			nextKey := findNextKey(testCase.trie.root, nil, testCase.key)

			assert.Equal(t, testCase.nextKey, nextKey)
			assert.Equal(t, *originalTrie, testCase.trie) // ensure no mutation
		})
	}
}

func Test_Trie_Put(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         Trie
		key          []byte
		value        []byte
		expectedTrie Trie
	}{
		"trie with key and value": {
			trie: Trie{
				generation: 1,
				root: &Node{
					Key:      []byte{1, 2, 0, 5},
					SubValue: []byte{1},
				},
			},
			key:   []byte{0x12, 0x16},
			value: []byte{2},
			expectedTrie: Trie{
				generation: 1,
				root: &Node{
					Key:         []byte{1, 2},
					Generation:  1,
					Dirty:       true,
					Descendants: 2,
					Children: padRightChildren([]*Node{
						{
							Key:        []byte{5},
							SubValue:   []byte{1},
							Generation: 1,
							Dirty:      true,
						},
						{
							Key:        []byte{6},
							SubValue:   []byte{2},
							Generation: 1,
							Dirty:      true,
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
		trie                Trie
		parent              *Node
		key                 []byte
		value               []byte
		deletedMerkleValues map[string]struct{}
		newNode             *Node
		mutated             bool
		nodesCreated        uint32
	}{
		"nil parent": {
			trie: Trie{
				generation: 1,
			},
			key:   []byte{1},
			value: []byte("leaf"),
			newNode: &Node{
				Key:        []byte{1},
				SubValue:   []byte("leaf"),
				Generation: 1,
				Dirty:      true,
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"branch parent": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte("branch"),
				Descendants: 1,
				Children: padRightChildren([]*Node{
					nil,
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{1, 0},
			value: []byte("leaf"),
			newNode: &Node{
				Key:         []byte{1},
				SubValue:    []byte("branch"),
				Generation:  1,
				Dirty:       true,
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{
						Key:        []byte{},
						SubValue:   []byte("leaf"),
						Generation: 1,
						Dirty:      true,
					},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"override leaf parent": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte("original leaf"),
			},
			key:   []byte{1},
			value: []byte("new leaf"),
			newNode: &Node{
				Key:        []byte{1},
				SubValue:   []byte("new leaf"),
				Generation: 1,
				Dirty:      true,
			},
			mutated: true,
		},
		"write same leaf value to leaf parent": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte("same"),
			},
			key:   []byte{1},
			value: []byte("same"),
			newNode: &Node{
				Key:      []byte{1},
				SubValue: []byte("same"),
			},
		},
		"write leaf as child to parent leaf": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte("original leaf"),
			},
			key:   []byte{1, 0},
			value: []byte("leaf"),
			newNode: &Node{
				Key:         []byte{1},
				SubValue:    []byte("original leaf"),
				Dirty:       true,
				Generation:  1,
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{
						Key:        []byte{},
						SubValue:   []byte("leaf"),
						Generation: 1,
						Dirty:      true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"write leaf as divergent child next to parent leaf": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte("original leaf"),
			},
			key:   []byte{2, 3},
			value: []byte("leaf"),
			newNode: &Node{
				Key:         []byte{},
				Dirty:       true,
				Generation:  1,
				Descendants: 2,
				Children: padRightChildren([]*Node{
					nil,
					{
						Key:        []byte{2},
						SubValue:   []byte("original leaf"),
						Dirty:      true,
						Generation: 1,
					},
					{
						Key:        []byte{3},
						SubValue:   []byte("leaf"),
						Generation: 1,
						Dirty:      true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 2,
		},
		"override leaf value": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			key:   []byte{1},
			value: []byte("leaf"),
			newNode: &Node{
				Key:        []byte{1},
				SubValue:   []byte("leaf"),
				Dirty:      true,
				Generation: 1,
			},
			mutated: true,
		},
		"write leaf as child to leaf": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
			key:   []byte{1},
			value: []byte("leaf"),
			newNode: &Node{
				Key:         []byte{1},
				SubValue:    []byte("leaf"),
				Dirty:       true,
				Generation:  1,
				Descendants: 1,
				Children: padRightChildren([]*Node{
					nil, nil,
					{
						Key:        []byte{},
						SubValue:   []byte{1},
						Dirty:      true,
						Generation: 1,
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

			newNode, mutated, nodesCreated := trie.insert(
				testCase.parent, testCase.key, testCase.value,
				testCase.deletedMerkleValues)

			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.mutated, mutated)
			assert.Equal(t, testCase.nodesCreated, nodesCreated)
			assert.Equal(t, expectedTrie, trie)
		})
	}
}

func Test_Trie_insertInBranch(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		parent              *Node
		key                 []byte
		value               []byte
		deletedMerkleValues map[string]struct{}
		newNode             *Node
		mutated             bool
		nodesCreated        uint32
	}{
		"insert existing value to branch": {
			parent: &Node{
				Key:         []byte{2},
				SubValue:    []byte("same"),
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{2},
			value: []byte("same"),
			newNode: &Node{
				Key:         []byte{2},
				SubValue:    []byte("same"),
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
		},
		"update with branch": {
			parent: &Node{
				Key:         []byte{2},
				SubValue:    []byte("old"),
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{2},
			value: []byte("new"),
			newNode: &Node{
				Key:         []byte{2},
				SubValue:    []byte("new"),
				Dirty:       true,
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			mutated: true,
		},
		"update with leaf": {
			parent: &Node{
				Key:         []byte{2},
				SubValue:    []byte("old"),
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{2},
			value: []byte("new"),
			newNode: &Node{
				Key:         []byte{2},
				SubValue:    []byte("new"),
				Dirty:       true,
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			mutated: true,
		},
		"add leaf as direct child": {
			parent: &Node{
				Key:         []byte{2},
				SubValue:    []byte{5},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{2, 3, 4, 5},
			value: []byte{6},
			newNode: &Node{
				Key:         []byte{2},
				SubValue:    []byte{5},
				Dirty:       true,
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					nil, nil,
					{
						Key:      []byte{4, 5},
						SubValue: []byte{6},
						Dirty:    true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"insert same leaf as existing direct child leaf": {
			parent: &Node{
				Key:         []byte{2},
				SubValue:    []byte{5},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{2, 0, 1},
			value: []byte{1},
			newNode: &Node{
				Key:         []byte{2},
				SubValue:    []byte{5},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
		},
		"add leaf as nested child": {
			parent: &Node{
				Key:         []byte{2},
				SubValue:    []byte{5},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					nil, nil, nil,
					{
						Key:         []byte{4},
						Descendants: 1,
						Children: padRightChildren([]*Node{
							{Key: []byte{1}, SubValue: []byte{1}},
						}),
					},
				}),
			},
			key:   []byte{2, 3, 4, 5, 6},
			value: []byte{6},
			newNode: &Node{
				Key:         []byte{2},
				SubValue:    []byte{5},
				Dirty:       true,
				Descendants: 3,
				Children: padRightChildren([]*Node{
					nil, nil, nil,
					{
						Key:         []byte{4},
						Dirty:       true,
						Descendants: 2,
						Children: padRightChildren([]*Node{
							{Key: []byte{1}, SubValue: []byte{1}},
							nil, nil, nil, nil,
							{
								Key:      []byte{6},
								SubValue: []byte{6},
								Dirty:    true,
							},
						}),
					},
				}),
			},
			mutated:      true,
			nodesCreated: 1,
		},
		"split branch for longer key": {
			parent: &Node{
				Key:         []byte{2, 3},
				SubValue:    []byte{5},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{2, 4, 5, 6},
			value: []byte{6},
			newNode: &Node{
				Key:         []byte{2},
				Dirty:       true,
				Descendants: 3,
				Children: padRightChildren([]*Node{
					nil, nil, nil,
					{
						Key:         []byte{},
						SubValue:    []byte{5},
						Dirty:       true,
						Descendants: 1,
						Children: padRightChildren([]*Node{
							{Key: []byte{1}, SubValue: []byte{1}},
						}),
					},
					{
						Key:      []byte{5, 6},
						SubValue: []byte{6},
						Dirty:    true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 2,
		},
		"split root branch": {
			parent: &Node{
				Key:         []byte{2, 3},
				SubValue:    []byte{5},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{3},
			value: []byte{6},
			newNode: &Node{
				Key:         []byte{},
				Dirty:       true,
				Descendants: 3,
				Children: padRightChildren([]*Node{
					nil, nil,
					{
						Key:         []byte{3},
						SubValue:    []byte{5},
						Dirty:       true,
						Descendants: 1,
						Children: padRightChildren([]*Node{
							{Key: []byte{1}, SubValue: []byte{1}},
						}),
					},
					{
						Key:      []byte{},
						SubValue: []byte{6},
						Dirty:    true,
					},
				}),
			},
			mutated:      true,
			nodesCreated: 2,
		},
		"update with leaf at empty key": {
			parent: &Node{
				Key:         []byte{2},
				SubValue:    []byte{5},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{},
			value: []byte{6},
			newNode: &Node{
				Key:         []byte{},
				SubValue:    []byte{6},
				Dirty:       true,
				Descendants: 2,
				Children: padRightChildren([]*Node{
					nil, nil,
					{
						Key:         []byte{},
						SubValue:    []byte{5},
						Dirty:       true,
						Descendants: 1,
						Children: padRightChildren([]*Node{
							{Key: []byte{1}, SubValue: []byte{1}},
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

			trie := new(Trie)

			newNode, mutated, nodesCreated := trie.insertInBranch(
				testCase.parent, testCase.key, testCase.value,
				testCase.deletedMerkleValues)

			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.mutated, mutated)
			assert.Equal(t, testCase.nodesCreated, nodesCreated)
			assert.Equal(t, new(Trie), trie) // check no mutation
		})
	}
}

func Test_LoadFromMap(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		data         map[string]string
		expectedTrie Trie
		errWrapped   error
		errMessage   string
	}{
		"nil data": {
			expectedTrie: Trie{
				childTries:          map[common.Hash]*Trie{},
				deletedMerkleValues: map[string]struct{}{},
			},
		},
		"empty data": {
			data: map[string]string{},
			expectedTrie: Trie{
				childTries:          map[common.Hash]*Trie{},
				deletedMerkleValues: map[string]struct{}{},
			},
		},
		"bad key": {
			data: map[string]string{
				"0xa": "0x01",
			},
			errWrapped: hex.ErrLength,
			errMessage: "cannot convert key hex to bytes: encoding/hex: odd length hex string: 0xa",
		},
		"bad value": {
			data: map[string]string{
				"0x01": "0xa",
			},
			errWrapped: hex.ErrLength,
			errMessage: "cannot convert value hex to bytes: encoding/hex: odd length hex string: 0xa",
		},
		"load large key value": {
			data: map[string]string{
				"0x01": "0x1234567812345678123456781234567812345678123456781234567812345678", // 32 bytes
			},
			expectedTrie: Trie{
				root: &Node{
					Key: []byte{00, 01},
					SubValue: []byte{
						0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
						0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
						0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
						0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
					},
					Dirty: true,
				},
				childTries:          map[common.Hash]*Trie{},
				deletedMerkleValues: map[string]struct{}{},
			},
		},
		"load key values": {
			data: map[string]string{
				"0x01":   "0x06",
				"0x0120": "0x07",
				"0x0130": "0x08",
			},
			expectedTrie: Trie{
				root: &Node{
					Key:         []byte{00, 01},
					SubValue:    []byte{6},
					Dirty:       true,
					Descendants: 2,
					Children: padRightChildren([]*Node{
						nil, nil,
						{
							Key:      []byte{0},
							SubValue: []byte{7},
							Dirty:    true,
						},
						{
							Key:      []byte{0},
							SubValue: []byte{8},
							Dirty:    true,
						},
					}),
				},
				childTries:          map[common.Hash]*Trie{},
				deletedMerkleValues: map[string]struct{}{},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie, err := LoadFromMap(testCase.data)

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
		trie   Trie
		prefix []byte
		keys   [][]byte
	}{
		"some trie": {
			trie: Trie{
				root: &Node{
					Key:         []byte{0, 1},
					Descendants: 4,
					Children: padRightChildren([]*Node{
						{ // full key 0, 1, 0, 3
							Key:         []byte{3},
							Descendants: 2,
							Children: padRightChildren([]*Node{
								{ // full key 0, 1, 0, 0, 4
									Key:      []byte{4},
									SubValue: []byte{1},
								},
								{ // full key 0, 1, 0, 1, 5
									Key:      []byte{5},
									SubValue: []byte{1},
								},
							}),
						},
						{ // full key 0, 1, 1, 9
							Key:      []byte{9},
							SubValue: []byte{1},
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
		parent       *Node
		prefix       []byte
		key          []byte
		keys         [][]byte
		expectedKeys [][]byte
	}{
		"nil parent returns keys passed": {
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"common prefix for parent branch and search key": {
			parent: &Node{
				Key:         []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{4}, SubValue: []byte{1}},
					{Key: []byte{5}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"parent branch and empty key": {
			parent: &Node{
				Key:         []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{4}, SubValue: []byte{1}},
					{Key: []byte{5}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"search key smaller than branch key with no full common prefix": {
			parent: &Node{
				Key:         []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{4}, SubValue: []byte{1}},
					{Key: []byte{5}, SubValue: []byte{1}},
				}),
			},
			key:          []byte{1, 3},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"common prefix smaller tan search key": {
			parent: &Node{
				Key:         []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{4}, SubValue: []byte{1}},
					{Key: []byte{5}, SubValue: []byte{1}},
				}),
			},
			key:          []byte{1, 2, 3},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"recursive call": {
			parent: &Node{
				Key:         []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{4}, SubValue: []byte{1}},
					{Key: []byte{5}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2, 3, 0},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04}},
		},
		"parent leaf with search key equal to common prefix": {
			parent: &Node{
				Key:      []byte{1, 2, 3},
				SubValue: []byte{1},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2, 3},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent leaf with empty search key": {
			parent: &Node{
				Key:      []byte{1, 2, 3},
				SubValue: []byte{1},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent leaf with too deep search key": {
			parent: &Node{
				Key:      []byte{1, 2, 3},
				SubValue: []byte{1},
			},
			prefix:       []byte{9, 8, 7},
			key:          []byte{1, 2, 3, 4},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"parent leaf with shorter matching search key": {
			parent: &Node{
				Key:      []byte{1, 2, 3},
				SubValue: []byte{1},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent leaf with not matching search key": {
			parent: &Node{
				Key:      []byte{1, 2, 3},
				SubValue: []byte{1},
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
		parent       *Node
		prefix       []byte
		keys         [][]byte
		expectedKeys [][]byte
	}{
		"nil parent returns keys passed": {
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"leaf parent": {
			parent: &Node{
				Key:      []byte{1, 2, 3},
				SubValue: []byte{1},
			},
			prefix: []byte{9, 8, 7},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent branch without value": {
			parent: &Node{
				Key:         []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{4}, SubValue: []byte{1}},
					{Key: []byte{5}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{9, 8, 7},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"parent branch with empty value": {
			parent: &Node{
				Key:         []byte{1, 2, 3},
				SubValue:    []byte{},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{4}, SubValue: []byte{1}},
					{Key: []byte{5}, SubValue: []byte{1}},
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
		trie  Trie
		key   []byte
		value []byte
	}{
		"some trie": {
			trie: Trie{
				root: &Node{
					Key:         []byte{0, 1},
					SubValue:    []byte{1, 3},
					Descendants: 3,
					Children: padRightChildren([]*Node{
						{ // full key 0, 1, 0, 3
							Key:         []byte{3},
							SubValue:    []byte{1, 2},
							Descendants: 1,
							Children: padRightChildren([]*Node{
								{Key: []byte{1}, SubValue: []byte{1}},
							}),
						},
						{ // full key 0, 1, 1, 9
							Key:      []byte{9},
							SubValue: []byte{1, 2, 3, 4, 5},
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

	testCases := map[string]struct {
		parent *Node
		key    []byte
		value  []byte
	}{
		"nil parent": {
			key: []byte{1},
		},
		"leaf key match": {
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{2},
			},
			key:   []byte{1},
			value: []byte{2},
		},
		"leaf key mismatch": {
			parent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{2},
			},
			key: []byte{1},
		},
		"branch key match": {
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{2},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key:   []byte{1},
			value: []byte{2},
		},
		"branch key with empty search key": {
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{2},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			value: []byte{2},
		},
		"branch key mismatch with shorter search key": {
			parent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{2},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			key: []byte{1},
		},
		"bottom leaf in branch": {
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					nil, nil,
					{ // full key 1, 2, 3
						Key:         []byte{3},
						SubValue:    []byte{2},
						Descendants: 1,
						Children: padRightChildren([]*Node{
							nil, nil, nil, nil,
							{ // full key 1, 2, 3, 4, 5
								Key:      []byte{5},
								SubValue: []byte{3},
							},
						}),
					},
				}),
			},
			key:   []byte{1, 2, 3, 4, 5},
			value: []byte{3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Check no mutation was done
			copySettings := node.DeepCopySettings
			var expectedParent *Node
			if testCase.parent != nil {
				expectedParent = testCase.parent.Copy(copySettings)
			}

			value := retrieve(testCase.parent, testCase.key)

			assert.Equal(t, testCase.value, value)
			assert.Equal(t, expectedParent, testCase.parent)
		})
	}
}

func Test_Trie_ClearPrefixLimit(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         Trie
		prefix       []byte
		limit        uint32
		deleted      uint32
		allDeleted   bool
		expectedTrie Trie
	}{
		"limit is zero": {},
		"clear prefix limit": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1, 2},
					SubValue:    []byte{1},
					Descendants: 1,
					Children: padRightChildren([]*Node{
						nil, nil, nil,
						{
							Key:      []byte{4},
							SubValue: []byte{1},
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

			deleted, allDeleted := trie.ClearPrefixLimit(testCase.prefix, testCase.limit)

			assert.Equal(t, testCase.deleted, deleted)
			assert.Equal(t, testCase.allDeleted, allDeleted)
			assert.Equal(t, testCase.expectedTrie, trie)
		})
	}
}

func Test_Trie_clearPrefixLimitAtNode(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                Trie
		parent              *Node
		prefix              []byte
		limit               uint32
		deletedMerkleValues map[string]struct{}
		newParent           *Node
		valuesDeleted       uint32
		nodesRemoved        uint32
		allDeleted          bool
	}{
		"limit is zero": {
			allDeleted: true,
		},
		"nil parent": {
			limit:      1,
			allDeleted: true,
		},
		"leaf parent with common prefix": {
			parent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
			prefix:        []byte{1},
			limit:         1,
			valuesDeleted: 1,
			nodesRemoved:  1,
			allDeleted:    true,
		},
		"leaf parent with key equal prefix": {
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			prefix:        []byte{1},
			limit:         1,
			valuesDeleted: 1,
			nodesRemoved:  1,
			allDeleted:    true,
		},
		"leaf parent with key no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
			prefix: []byte{1, 3},
			limit:  1,
			newParent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
			allDeleted: true,
		},
		"leaf parent with key smaller than prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			prefix: []byte{1, 2},
			limit:  1,
			newParent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			allDeleted: true,
		},
		"branch without value parent with common prefix": {
			parent: &Node{
				Key:         []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			prefix:        []byte{1},
			limit:         3,
			valuesDeleted: 2,
			nodesRemoved:  3,
			allDeleted:    true,
		},
		"branch without value with key equal prefix": {
			parent: &Node{
				Key:         []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			prefix:        []byte{1, 2},
			limit:         3,
			valuesDeleted: 2,
			nodesRemoved:  3,
			allDeleted:    true,
		},
		"branch without value with no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 3},
			limit:  1,
			newParent: &Node{
				Key:         []byte{1, 2},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch without value with key smaller than prefix by more than one": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 2, 3},
			limit:  1,
			newParent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch without value with key smaller than prefix by one": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 2},
			limit:  1,
			newParent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch with value with common prefix": {
			parent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			prefix:        []byte{1},
			limit:         2,
			valuesDeleted: 2,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"branch with value with key equal prefix": {
			parent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			prefix:        []byte{1, 2},
			limit:         2,
			valuesDeleted: 2,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"branch with value with no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 3},
			limit:  1,
			newParent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch with value with key smaller than prefix by more than one": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 2, 3},
			limit:  1,
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"branch with value with key smaller than prefix by one": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 2},
			limit:  1,
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
				}),
			},
			allDeleted: true,
		},
		"delete one child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1},
			limit:  1,
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Dirty:       true,
				Generation:  1,
				Descendants: 1,
				Children: padRightChildren([]*Node{
					nil,
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			valuesDeleted: 1,
			nodesRemoved:  1,
		},
		"delete only child of branch": {
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			limit:  1,
			newParent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
				Dirty:    true,
			},
			valuesDeleted: 1,
			nodesRemoved:  1,
			allDeleted:    true,
		},
		"fully delete children of branch with value": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1},
			limit:  2,
			newParent: &Node{
				Key:        []byte{1},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			valuesDeleted: 2,
			nodesRemoved:  2,
		},
		"fully delete children of branch without value": {
			parent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			prefix:        []byte{1},
			limit:         2,
			valuesDeleted: 2,
			nodesRemoved:  3,
			allDeleted:    true,
		},
		"partially delete child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 3,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 3
						Key:         []byte{3},
						SubValue:    []byte{1},
						Descendants: 1,
						Children: padRightChildren([]*Node{
							{ // full key 1, 0, 3, 0, 5
								Key:      []byte{5},
								SubValue: []byte{1},
							},
						}),
					},
					{
						Key:      []byte{6},
						SubValue: []byte{1},
					},
				}),
			},
			prefix: []byte{1, 0},
			limit:  1,
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Dirty:       true,
				Generation:  1,
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 3
						Key:        []byte{3},
						SubValue:   []byte{1},
						Dirty:      true,
						Generation: 1,
					},
					{
						Key:      []byte{6},
						SubValue: []byte{1},
						// Not modified so same generation as before
					},
				}),
			},
			valuesDeleted: 1,
			nodesRemoved:  1,
		},
		"update child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 2
						Key:         []byte{2},
						SubValue:    []byte{1},
						Descendants: 1,
						Children: padRightChildren([]*Node{
							{Key: []byte{1}, SubValue: []byte{1}},
						}),
					},
				}),
			},
			prefix: []byte{1, 0, 2},
			limit:  2,
			newParent: &Node{
				Key:        []byte{1},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			valuesDeleted: 2,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"delete one of two children of branch without value": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0, 3},
			limit:  3,
			newParent: &Node{
				Key:        []byte{1, 1, 4},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			valuesDeleted: 1,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"delete one of two children of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			limit:  3,
			newParent: &Node{
				Key:        []byte{1, 1, 4},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			valuesDeleted: 1,
			nodesRemoved:  2,
			allDeleted:    true,
		},
		"delete child of branch with limit reached": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
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

			newParent, valuesDeleted, nodesRemoved, allDeleted :=
				trie.clearPrefixLimitAtNode(testCase.parent, testCase.prefix,
					testCase.limit, testCase.deletedMerkleValues)

			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.valuesDeleted, valuesDeleted)
			assert.Equal(t, testCase.nodesRemoved, nodesRemoved)
			assert.Equal(t, testCase.allDeleted, allDeleted)
			assert.Equal(t, expectedTrie, trie)
		})
	}
}

func Test_Trie_deleteNodesLimit(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie                Trie
		parent              *Node
		limit               uint32
		deletedMerkleValues map[string]struct{}
		newNode             *Node
		valuesDeleted       uint32
		nodesRemoved        uint32
	}{
		"zero limit": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			newNode: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
		},
		"nil parent": {
			limit: 1,
		},
		"delete leaf": {
			parent: &Node{
				SubValue: []byte{1},
			},
			limit:         2,
			valuesDeleted: 1,
			nodesRemoved:  1,
		},
		"delete branch without value": {
			parent: &Node{
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{},
					{},
				}),
			},
			limit:         3,
			valuesDeleted: 2,
			nodesRemoved:  3,
		},
		"delete branch with value": {
			parent: &Node{
				Key:         []byte{3},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
			limit:         3,
			valuesDeleted: 2,
			nodesRemoved:  2,
		},
		"delete branch and all children": {
			parent: &Node{
				Key:         []byte{3},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			limit:         10,
			valuesDeleted: 2,
			nodesRemoved:  3,
		},
		"delete branch one child only": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{3},
				SubValue:    []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			limit: 1,
			newNode: &Node{
				Key:         []byte{3},
				SubValue:    []byte{1, 2, 3},
				Dirty:       true,
				Generation:  1,
				Descendants: 1,
				Children: padRightChildren([]*Node{
					nil,
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			valuesDeleted: 1,
			nodesRemoved:  1,
		},
		"delete branch children only": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{3},
				SubValue:    []byte{1, 2, 3},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{1}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			limit: 2,
			newNode: &Node{
				Key:        []byte{3},
				SubValue:   []byte{1, 2, 3},
				Dirty:      true,
				Generation: 1,
			},
			valuesDeleted: 2,
			nodesRemoved:  2,
		},
		"delete branch all children except one": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{3},
				Descendants: 3,
				Children: padRightChildren([]*Node{
					nil,
					{Key: []byte{1}, SubValue: []byte{1}},
					nil,
					{Key: []byte{2}, SubValue: []byte{1}},
					nil,
					{Key: []byte{3}, SubValue: []byte{1}},
				}),
			},
			limit: 2,
			newNode: &Node{
				Key:        []byte{3, 5, 3},
				SubValue:   []byte{1},
				Generation: 1,
				Dirty:      true,
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

			newNode, valuesDeleted, nodesRemoved :=
				trie.deleteNodesLimit(testCase.parent,
					testCase.limit, testCase.deletedMerkleValues)

			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.valuesDeleted, valuesDeleted)
			assert.Equal(t, testCase.nodesRemoved, nodesRemoved)
			assert.Equal(t, expectedTrie, trie)
		})
	}
}

func Test_Trie_ClearPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         Trie
		prefix       []byte
		expectedTrie Trie
	}{
		"nil prefix": {
			trie: Trie{
				root: &Node{SubValue: []byte{1}},
			},
		},
		"empty prefix": {
			trie: Trie{
				root: &Node{SubValue: []byte{1}},
			},
			prefix: []byte{},
		},
		"empty trie": {
			prefix: []byte{0x12},
		},
		"clear prefix": {
			trie: Trie{
				root: &Node{
					Key:         []byte{1, 2},
					Descendants: 3,
					Children: padRightChildren([]*Node{
						{ // full key in nibbles 1, 2, 0, 5
							Key:      []byte{5},
							SubValue: []byte{1},
						},
						{ // full key in nibbles 1, 2, 1, 6
							Key:      []byte{6},
							SubValue: []byte("bottom branch"),
							Children: padRightChildren([]*Node{
								{ // full key in nibbles 1, 2, 1, 6, 0, 7
									Key:      []byte{7},
									SubValue: []byte{1},
								},
							}),
						},
					}),
				},
			},
			prefix: []byte{0x12, 0x16},
			expectedTrie: Trie{
				root: &Node{
					Key:      []byte{1, 2, 0, 5},
					SubValue: []byte{1},
					Dirty:    true,
				},
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
		trie                Trie
		parent              *Node
		prefix              []byte
		deletedMerkleValues map[string]struct{}
		newParent           *Node
		nodesRemoved        uint32
	}{
		"delete one of two children of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			newParent: &Node{
				Key:        []byte{1, 1, 4},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			nodesRemoved: 2,
		},
		"nil parent": {},
		"leaf parent with common prefix": {
			parent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
			prefix:       []byte{1},
			nodesRemoved: 1,
		},
		"leaf parent with key equal prefix": {
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			prefix:       []byte{1},
			nodesRemoved: 1,
		},
		"leaf parent with key no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
			prefix: []byte{1, 3},
			newParent: &Node{
				Key:      []byte{1, 2},
				SubValue: []byte{1},
			},
		},
		"leaf parent with key smaller than prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			prefix: []byte{1, 2},
			newParent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
		},
		"branch parent with common prefix": {
			parent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
			prefix:       []byte{1},
			nodesRemoved: 2,
		},
		"branch with key equal prefix": {
			parent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
			prefix:       []byte{1, 2},
			nodesRemoved: 2,
		},
		"branch with no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
			prefix: []byte{1, 3},
			newParent: &Node{
				Key:         []byte{1, 2},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
		},
		"branch with key smaller than prefix by more than one": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
			prefix: []byte{1, 2, 3},
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
		},
		"branch with key smaller than prefix by one": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
			prefix: []byte{1, 2},
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
		},
		"delete one child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0, 3},
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Dirty:       true,
				Generation:  1,
				Descendants: 1,
				Children: padRightChildren([]*Node{
					nil,
					{Key: []byte{4}, SubValue: []byte{1}},
				}),
			},
			nodesRemoved: 1,
		},
		"fully delete child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}},
				}),
			},
			prefix: []byte{1, 0},
			newParent: &Node{
				Key:        []byte{1},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			nodesRemoved: 1,
		},
		"partially delete child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 3
						Key:         []byte{3},
						SubValue:    []byte{1},
						Descendants: 1,
						Children: padRightChildren([]*Node{
							{ // full key 1, 0, 3, 0, 5
								Key:      []byte{5},
								SubValue: []byte{1},
							},
						}),
					},
				}),
			},
			prefix: []byte{1, 0, 3, 0},
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Dirty:       true,
				Generation:  1,
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 3
						Key:        []byte{3},
						SubValue:   []byte{1},
						Dirty:      true,
						Generation: 1,
					},
				}),
			},
			nodesRemoved: 1,
		},
		"delete one of two children of branch without value": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{3}, SubValue: []byte{1}}, // full key 1, 0, 3
					{Key: []byte{4}, SubValue: []byte{1}}, // full key 1, 1, 4
				}),
			},
			prefix: []byte{1, 0, 3},
			newParent: &Node{
				Key:        []byte{1, 1, 4},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			nodesRemoved: 2,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newParent, nodesRemoved := trie.clearPrefixAtNode(
				testCase.parent, testCase.prefix, testCase.deletedMerkleValues)

			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.nodesRemoved, nodesRemoved)
			assert.Equal(t, expectedTrie, trie)
		})
	}
}

func Test_Trie_Delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         Trie
		key          []byte
		expectedTrie Trie
	}{
		"nil key": {
			trie: Trie{
				root: &Node{SubValue: []byte{1}},
			},
		},
		"empty key": {
			trie: Trie{
				root: &Node{SubValue: []byte{1}},
			},
		},
		"empty trie": {
			key: []byte{0x12},
		},
		"delete branch node": {
			trie: Trie{
				generation: 1,
				root: &Node{
					Key:         []byte{1, 2},
					Descendants: 3,
					Children: padRightChildren([]*Node{
						{
							Key:      []byte{5},
							SubValue: []byte{97},
						},
						{ // full key in nibbles 1, 2, 1, 6
							Key:         []byte{6},
							SubValue:    []byte{98},
							Descendants: 1,
							Children: padRightChildren([]*Node{
								{ // full key in nibbles 1, 2, 1, 6, 0, 7
									Key:      []byte{7},
									SubValue: []byte{99},
								},
							}),
						},
					}),
				},
			},
			key: []byte{0x12, 0x16},
			expectedTrie: Trie{
				generation: 1,
				root: &Node{
					Key:         []byte{1, 2},
					Dirty:       true,
					Generation:  1,
					Descendants: 2,
					Children: padRightChildren([]*Node{
						{
							Key:      []byte{5},
							SubValue: []byte{97},
						},
						{ // full key in nibbles 1, 2, 1, 6
							Key:        []byte{6, 0, 7},
							SubValue:   []byte{99},
							Dirty:      true,
							Generation: 1,
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
		trie                Trie
		parent              *Node
		key                 []byte
		deletedMerkleValues map[string]struct{}
		newParent           *Node
		updated             bool
		nodesRemoved        uint32
	}{
		"nil parent": {
			key: []byte{1},
		},
		"leaf parent and nil key": {
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			updated:      true,
			nodesRemoved: 1,
		},
		"leaf parent and empty key": {
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			key:          []byte{},
			updated:      true,
			nodesRemoved: 1,
		},
		"leaf parent matches key": {
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			key:          []byte{1},
			updated:      true,
			nodesRemoved: 1,
		},
		"leaf parent mismatches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
			key: []byte{2},
			newParent: &Node{
				Key:      []byte{1},
				SubValue: []byte{1},
			},
		},
		"branch parent and nil key": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{
						Key:      []byte{2},
						SubValue: []byte{1},
					},
				}),
			},
			newParent: &Node{
				Key:        []byte{1, 0, 2},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			updated:      true,
			nodesRemoved: 1,
		},
		"branch parent and empty key": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			key: []byte{},
			newParent: &Node{
				Key:        []byte{1, 0, 2},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			updated:      true,
			nodesRemoved: 1,
		},
		"branch parent matches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			key: []byte{1},
			newParent: &Node{
				Key:        []byte{1, 0, 2},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			updated:      true,
			nodesRemoved: 1,
		},
		"branch parent child matches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 2
						Key:      []byte{2},
						SubValue: []byte{1},
					},
				}),
			},
			key: []byte{1, 0, 2},
			newParent: &Node{
				Key:        []byte{1},
				SubValue:   []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			updated:      true,
			nodesRemoved: 1,
		},
		"branch parent mismatches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
			key: []byte{2},
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{},
				}),
			},
		},
		"branch parent child mismatches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 2
						Key:      []byte{2},
						SubValue: []byte{1},
					},
				}),
			},
			key: []byte{1, 0, 3},
			newParent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 2
						Key:      []byte{2},
						SubValue: []byte{1},
					},
				}),
			},
		},
		"delete branch child and merge branch and left child": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 2
						Key:      []byte{2},
						SubValue: []byte{1},
					},
					{ // full key 1, 1, 2
						Key:      []byte{2},
						SubValue: []byte{2},
					},
				}),
			},
			key: []byte{1, 0, 2},
			newParent: &Node{
				Key:        []byte{1, 1, 2},
				SubValue:   []byte{2},
				Dirty:      true,
				Generation: 1,
			},
			updated:      true,
			nodesRemoved: 2,
		},
		"delete branch and keep two children": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1},
				SubValue:    []byte{1},
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{2}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			key: []byte{1},
			newParent: &Node{
				Key:         []byte{1},
				Generation:  1,
				Dirty:       true,
				Descendants: 2,
				Children: padRightChildren([]*Node{
					{Key: []byte{2}, SubValue: []byte{1}},
					{Key: []byte{2}, SubValue: []byte{1}},
				}),
			},
			updated: true,
		},
		"handle nonexistent key (no op)": {
			trie: Trie{
				generation: 1,
			},
			parent: &Node{
				Key:         []byte{1, 0, 2, 3},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 2
						Key:      []byte{2},
						SubValue: []byte{1},
					},
					{ // full key 1, 1, 2
						Key:      []byte{2},
						SubValue: []byte{2},
					},
				}),
			},
			key: []byte{1, 0, 2},
			newParent: &Node{
				Key:         []byte{1, 0, 2, 3},
				Descendants: 1,
				Children: padRightChildren([]*Node{
					{ // full key 1, 0, 2
						Key:      []byte{2},
						SubValue: []byte{1},
					},
					{ // full key 1, 1, 2
						Key:      []byte{2},
						SubValue: []byte{2},
					},
				}),
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
			expectedTrie := *testCase.trie.DeepCopy()

			newParent, updated, nodesRemoved := testCase.trie.deleteAtNode(
				testCase.parent, testCase.key, testCase.deletedMerkleValues)

			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.updated, updated)
			assert.Equal(t, testCase.nodesRemoved, nodesRemoved)
			assert.Equal(t, expectedTrie, testCase.trie)
			assert.Equal(t, expectedKey, testCase.key)
		})
	}
}

func Test_handleDeletion(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch            *Node
		deletedKey        []byte
		newNode           *Node
		branchChildMerged bool
	}{
		"branch with value and without children": {
			branch: &Node{
				Key:        []byte{1, 2, 3},
				SubValue:   []byte{5, 6, 7},
				Generation: 1,
			},
			deletedKey: []byte{1, 2, 3, 4},
			newNode: &Node{
				Key:        []byte{1, 2, 3},
				SubValue:   []byte{5, 6, 7},
				Generation: 1,
				Dirty:      true,
			},
		},
		// branch without value and without children cannot happen
		// since it would be turned into a leaf when it only has one child
		// remaining.
		"branch with value and a single child": {
			branch: &Node{
				Key:        []byte{1, 2, 3},
				SubValue:   []byte{5, 6, 7},
				Generation: 1,
				Children: padRightChildren([]*Node{
					nil,
					{Key: []byte{9}, SubValue: []byte{1}},
				}),
			},
			newNode: &Node{
				Key:        []byte{1, 2, 3},
				SubValue:   []byte{5, 6, 7},
				Generation: 1,
				Children: padRightChildren([]*Node{
					nil,
					{Key: []byte{9}, SubValue: []byte{1}},
				}),
			},
		},
		"branch without value and a single leaf child": {
			branch: &Node{
				Key:        []byte{1, 2, 3},
				Generation: 1,
				Children: padRightChildren([]*Node{
					nil,
					{ // full key 1,2,3,1,9
						Key:      []byte{9},
						SubValue: []byte{10},
					},
				}),
			},
			deletedKey: []byte{1, 2, 3, 4},
			newNode: &Node{
				Key:        []byte{1, 2, 3, 1, 9},
				SubValue:   []byte{10},
				Generation: 1,
				Dirty:      true,
			},
			branchChildMerged: true,
		},
		"branch without value and a single branch child": {
			branch: &Node{
				Key:        []byte{1, 2, 3},
				Generation: 1,
				Children: padRightChildren([]*Node{
					nil,
					{
						Key:      []byte{9},
						SubValue: []byte{10},
						Children: padRightChildren([]*Node{
							{Key: []byte{7}, SubValue: []byte{1}},
							nil,
							{Key: []byte{8}, SubValue: []byte{1}},
						}),
					},
				}),
			},
			newNode: &Node{
				Key:        []byte{1, 2, 3, 1, 9},
				SubValue:   []byte{10},
				Generation: 1,
				Dirty:      true,
				Children: padRightChildren([]*Node{
					{Key: []byte{7}, SubValue: []byte{1}},
					nil,
					{Key: []byte{8}, SubValue: []byte{1}},
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

			newNode, branchChildMerged := handleDeletion(testCase.branch, testCase.deletedKey)

			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, testCase.branchChildMerged, branchChildMerged)
			assert.Equal(t, expectedKey, testCase.deletedKey)
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
		"nil slices": {},
		"empty slices": {
			a: []byte{},
			b: []byte{},
		},
		"fully different": {
			a: []byte{1, 2, 3},
			b: []byte{4, 5, 6},
		},
		"fully same": {
			a:      []byte{1, 2, 3},
			b:      []byte{1, 2, 3},
			length: 3,
		},
		"different and common prefix": {
			a:      []byte{1, 2, 3, 4},
			b:      []byte{1, 2, 4, 4},
			length: 2,
		},
		"first bigger than second": {
			a:      []byte{1, 2, 3},
			b:      []byte{1, 2},
			length: 2,
		},
		"first smaller than second": {
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
		"two nil slices": {},
		"four nil slices": {
			otherSlices: [][]byte{nil, nil},
		},
		"only fourth slice not nil": {
			otherSlices: [][]byte{
				nil,
				{1},
			},
			concatenated: []byte{1},
		},
		"two empty slices": {
			sliceOne:     []byte{},
			sliceTwo:     []byte{},
			concatenated: []byte{},
		},
		"three empty slices": {
			sliceOne:     []byte{},
			sliceTwo:     []byte{},
			otherSlices:  [][]byte{{}},
			concatenated: []byte{},
		},
		"concatenate two first slices": {
			sliceOne:     []byte{1, 2},
			sliceTwo:     []byte{3, 4},
			concatenated: []byte{1, 2, 3, 4},
		},

		"concatenate four slices": {
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
}
