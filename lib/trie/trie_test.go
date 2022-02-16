// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_NewEmptyTrie(t *testing.T) {
	expectedTrie := &Trie{
		childTries:  make(map[common.Hash]*Trie),
		deletedKeys: map[common.Hash]struct{}{},
	}
	trie := NewEmptyTrie()
	assert.Equal(t, expectedTrie, trie)
}

func Test_NewTrie(t *testing.T) {
	root := &node.Leaf{
		Key:   []byte{0},
		Value: []byte{17},
	}
	expectedTrie := &Trie{
		root: &node.Leaf{
			Key:   []byte{0},
			Value: []byte{17},
		},
		childTries:  make(map[common.Hash]*Trie),
		deletedKeys: map[common.Hash]struct{}{},
	}
	trie := NewTrie(root)
	assert.Equal(t, expectedTrie, trie)
}

func Test_Trie_Snapshot(t *testing.T) {
	t.Parallel()

	trie := &Trie{
		generation: 8,
		root:       &node.Leaf{Key: []byte{8}},
		childTries: map[common.Hash]*Trie{
			{1}: {
				generation: 1,
				root:       &node.Leaf{Key: []byte{1}},
				deletedKeys: map[common.Hash]struct{}{
					{1}: {},
				},
			},
			{2}: {
				generation: 2,
				root:       &node.Leaf{Key: []byte{2}},
				deletedKeys: map[common.Hash]struct{}{
					{2}: {},
				},
			},
		},
		deletedKeys: map[common.Hash]struct{}{
			{1}: {},
			{2}: {},
		},
	}

	expectedTrie := &Trie{
		generation: 9,
		root:       &node.Leaf{Key: []byte{8}},
		childTries: map[common.Hash]*Trie{
			{1}: {
				generation:  2,
				root:        &node.Leaf{Key: []byte{1}},
				deletedKeys: map[common.Hash]struct{}{},
			},
			{2}: {
				generation:  3,
				root:        &node.Leaf{Key: []byte{2}},
				deletedKeys: map[common.Hash]struct{}{},
			},
		},
		deletedKeys: map[common.Hash]struct{}{},
	}

	newTrie := trie.Snapshot()

	assert.Equal(t, expectedTrie, newTrie)
}

func Test_Trie_RootNode(t *testing.T) {
	t.Parallel()

	trie := Trie{
		root: &node.Leaf{
			Key: []byte{1, 2, 3},
		},
	}
	expectedRoot := &node.Leaf{
		Key: []byte{1, 2, 3},
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
		root         node.Node
		bufferCalls  bufferCalls
		errWrapped   error
		errMessage   string
		expectedRoot node.Node
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
			root: &node.Leaf{
				Key: []byte{1, 2},
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
			expectedRoot: &node.Leaf{
				Key: []byte{1, 2},
			},
		},
		"root encoding success": {
			root: &node.Leaf{
				Key: []byte{1, 2},
			},
			bufferCalls: bufferCalls{
				writeCalls: []writeCall{
					{written: []byte{66}},
					{written: []byte{18}},
					{written: []byte{0}},
				},
				lenCall:     true,
				lenReturn:   3,
				bytesCall:   true,
				bytesReturn: []byte{66, 18, 0},
			},
			expectedRoot: &node.Leaf{
				Key:      []byte{1, 2},
				Encoding: []byte{66, 18, 0},
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
				root: &node.Leaf{
					Key: []byte{1, 2, 3},
				},
			},
			hash: common.Hash{
				0x84, 0x7c, 0x95, 0x42, 0x8d, 0x9c, 0xcf, 0xce,
				0xa7, 0x27, 0x15, 0x33, 0x48, 0x74, 0x99, 0x11,
				0x83, 0xb8, 0xe8, 0xc4, 0x80, 0x88, 0xea, 0x4d,
				0x9f, 0x57, 0x82, 0x94, 0xc9, 0x76, 0xf4, 0x6f},
			expectedTrie: Trie{
				root: &node.Leaf{
					Key:      []byte{1, 2, 3},
					Encoding: []byte{67, 1, 35, 0},
				},
			},
		},
		"branch root": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{1, 2, 3},
					Value: []byte("branch"),
					Children: [16]node.Node{
						&node.Leaf{Key: []byte{9}},
					},
				},
			},
			hash: common.Hash{
				0xbc, 0x4b, 0x90, 0x4c, 0x65, 0xb1, 0x3b, 0x9b,
				0xcf, 0xe2, 0x32, 0xe3, 0xe6, 0x50, 0x20, 0xd8,
				0x21, 0x96, 0xce, 0xbf, 0x4c, 0xa4, 0xd, 0xaa,
				0xbe, 0x27, 0xab, 0x13, 0xcb, 0xf0, 0xfd, 0xd7},
			expectedTrie: Trie{
				root: &node.Branch{
					Key:   []byte{1, 2, 3},
					Value: []byte("branch"),
					Children: [16]node.Node{
						&node.Leaf{
							Key:      []byte{9},
							Encoding: []byte{0x41, 0x09, 0x00},
						},
					},
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

		root := &node.Branch{
			Key:   []byte{0xa},
			Value: []byte("root"),
			Children: [16]node.Node{
				&node.Leaf{ // index 0
					Key:   []byte{2, 0xb},
					Value: []byte("leaf"),
				},
				nil,
				&node.Leaf{ // index 2
					Key:   []byte{0xb},
					Value: []byte("leaf"),
				},
			},
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

		root := &node.Branch{
			Key:   []byte{0xa, 0xb},
			Value: []byte("root"),
			Children: [16]node.Node{
				nil, nil, nil,
				&node.Branch{ // branch with value at child index 3
					Key:   []byte{0xb},
					Value: []byte("branch 1"),
					Children: [16]node.Node{
						nil, nil, nil,
						&node.Leaf{ // leaf at child index 3
							Key:   []byte{0xc},
							Value: []byte("bottom leaf"),
						},
					},
				},
				nil, nil, nil,
				&node.Leaf{ // leaf at child index 7
					Key:   []byte{0xd},
					Value: []byte("top leaf"),
				},
				nil,
				&node.Branch{ // branch without value at child index 9
					Key:   []byte{0xe},
					Value: []byte("branch 2"),
					Children: [16]node.Node{
						&node.Leaf{ // leaf at child index 0
							Key:   []byte{0xf},
							Value: []byte("bottom leaf 2"),
						}, nil, nil,
					},
				},
			},
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
			root:        nil,
			childTries:  make(map[common.Hash]*Trie),
			deletedKeys: make(map[common.Hash]struct{}),
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
				root: &node.Leaf{
					Key: []byte{2},
				},
			},
			nextKey: []byte{2},
		},
		"key smaller than root leaf full key": {
			trie: Trie{
				root: &node.Leaf{
					Key: []byte{2},
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
				root: &node.Leaf{
					Key: []byte{2},
				},
			},
			nextKey: []byte{2},
		},
		"key smaller than root leaf full key": {
			trie: Trie{
				root: &node.Leaf{
					Key: []byte{2},
				},
			},
			key:     []byte{1},
			nextKey: []byte{2},
		},
		"key equal to root leaf full key": {
			trie: Trie{
				root: &node.Leaf{
					Key: []byte{2},
				},
			},
			key: []byte{2},
		},
		"key greater than root leaf full key": {
			trie: Trie{
				root: &node.Leaf{
					Key: []byte{2},
				},
			},
			key: []byte{3},
		},
		"key smaller than root branch full key": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{2},
					Value: []byte("branch"),
					Children: [16]node.Node{
						&node.Leaf{
							Key: []byte{1},
						},
					},
				},
			},
			key:     []byte{1},
			nextKey: []byte{2},
		},
		"key equal to root branch full key": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{2},
					Value: []byte("branch"),
					Children: [16]node.Node{
						&node.Leaf{
							Key: []byte{1},
						},
					},
				},
			},
			key: []byte{2, 0, 1},
		},
		"key smaller than leaf full key": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{1},
					Value: []byte("branch"),
					Children: [16]node.Node{
						nil, nil,
						&node.Leaf{
							// full key [1, 2, 3]
							Key: []byte{3},
						},
					},
				},
			},
			key:     []byte{1, 2, 2},
			nextKey: []byte{1, 2, 3},
		},
		"key equal to leaf full key": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{1},
					Value: []byte("branch"),
					Children: [16]node.Node{
						nil, nil,
						&node.Leaf{
							// full key [1, 2, 3]
							Key: []byte{3},
						},
					},
				},
			},
			key: []byte{1, 2, 3},
		},
		"key greater than leaf full key": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{1},
					Value: []byte("branch"),
					Children: [16]node.Node{
						nil, nil,
						&node.Leaf{
							// full key [1, 2, 3]
							Key: []byte{3},
						},
					},
				},
			},
			key: []byte{1, 2, 4},
		},
		"next key branch with value": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{1},
					Value: []byte("top branch"),
					Children: [16]node.Node{
						nil, nil,
						&node.Branch{
							// full key [1, 2, 3]
							Key:   []byte{3},
							Value: []byte("branch 1"),
							Children: [16]node.Node{
								nil, nil, nil, nil,
								&node.Leaf{
									// full key [1, 2, 3, 4, 5]
									Key:   []byte{0x5},
									Value: []byte("bottom leaf"),
								},
							},
						},
					},
				},
			},
			key:     []byte{1},
			nextKey: []byte{1, 2, 3},
		},
		"next key go through branch without value": {
			trie: Trie{
				root: &node.Branch{
					Key: []byte{1},
					Children: [16]node.Node{
						nil, nil,
						&node.Branch{
							// full key [1, 2, 3]
							Key: []byte{3},
							Children: [16]node.Node{
								nil, nil, nil, nil,
								&node.Leaf{
									// full key [1, 2, 3, 4, 5]
									Key:   []byte{0x5},
									Value: []byte("bottom leaf"),
								},
							},
						},
					},
				},
			},
			key:     []byte{0},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"next key leaf from bottom branch": {
			trie: Trie{
				root: &node.Branch{
					Key: []byte{1},
					Children: [16]node.Node{
						nil, nil,
						&node.Branch{
							// full key [1, 2, 3]
							Key:   []byte{3},
							Value: []byte("bottom branch"),
							Children: [16]node.Node{
								nil, nil, nil, nil,
								&node.Leaf{
									// full key [1, 2, 3, 4, 5]
									Key:   []byte{0x5},
									Value: []byte("bottom leaf"),
								},
							},
						},
					},
				},
			},
			key:     []byte{1, 2, 3},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"next key greater than branch": {
			trie: Trie{
				root: &node.Branch{
					Key: []byte{1},
					Children: [16]node.Node{
						nil, nil,
						&node.Branch{
							// full key [1, 2, 3]
							Key:   []byte{3},
							Value: []byte("bottom branch"),
							Children: [16]node.Node{
								nil, nil, nil, nil,
								&node.Leaf{
									// full key [1, 2, 3, 4, 5]
									Key:   []byte{0x5},
									Value: []byte("bottom leaf"),
								},
							},
						},
					},
				},
			},
			key:     []byte{1, 2, 3},
			nextKey: []byte{1, 2, 3, 4, 5},
		},
		"key smaller length and greater than root branch full key": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{2, 0},
					Value: []byte("branch"),
					Children: [16]node.Node{
						&node.Leaf{Key: []byte{1}},
					},
				},
			},
			key: []byte{3},
		},
		"key smaller length and greater than root leaf full key": {
			trie: Trie{
				root: &node.Leaf{
					Key:   []byte{2, 0},
					Value: []byte("leaf"),
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
				root: &node.Leaf{
					Key:   []byte{1, 2, 0, 5},
					Value: []byte{1},
				},
			},
			key:   []byte{0x12, 0x16},
			value: []byte{2},
			expectedTrie: Trie{
				generation: 1,
				root: &node.Branch{
					Key:        []byte{1, 2},
					Generation: 1,
					Dirty:      true,
					Children: [16]node.Node{
						&node.Leaf{
							Key:        []byte{5},
							Value:      []byte{1},
							Generation: 1,
							Dirty:      true,
						},
						&node.Leaf{
							Key:        []byte{6},
							Value:      []byte{2},
							Generation: 1,
							Dirty:      true,
						},
					},
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

func Test_Trie_put(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         Trie
		key          []byte
		value        []byte
		expectedTrie Trie
	}{
		"nil everything": {
			trie: Trie{
				generation: 1,
			},
			expectedTrie: Trie{
				generation: 1,
				root: &node.Leaf{
					Generation: 1,
					Dirty:      true,
				},
			},
		},
		"empty trie with nil key and value": {
			trie: Trie{
				generation: 1,
			},
			value: []byte{3, 4},
			expectedTrie: Trie{
				generation: 1,
				root: &node.Leaf{
					Value:      []byte{3, 4},
					Generation: 1,
					Dirty:      true,
				},
			},
		},
		"empty trie with key and value": {
			trie: Trie{
				generation: 1,
			},
			key:   []byte{1, 2},
			value: []byte{3, 4},
			expectedTrie: Trie{
				generation: 1,
				root: &node.Leaf{
					Key:        []byte{1, 2},
					Value:      []byte{3, 4},
					Generation: 1,
					Dirty:      true,
				},
			},
		},
		"trie with key and value": {
			trie: Trie{
				generation: 1,
				root: &node.Leaf{
					Key:   []byte{1, 0, 5},
					Value: []byte{1},
				},
			},
			key:   []byte{1, 1, 6},
			value: []byte{2},
			expectedTrie: Trie{
				generation: 1,
				root: &node.Branch{
					Key:        []byte{1},
					Generation: 1,
					Dirty:      true,
					Children: [16]node.Node{
						&node.Leaf{
							Key:        []byte{5},
							Value:      []byte{1},
							Generation: 1,
							Dirty:      true,
						},
						&node.Leaf{
							Key:        []byte{6},
							Value:      []byte{2},
							Generation: 1,
							Dirty:      true,
						},
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			trie.put(testCase.key, testCase.value)

			assert.Equal(t, testCase.expectedTrie, trie)
		})
	}
}

func Test_Trie_insert(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie    Trie
		parent  Node
		key     []byte
		value   Node
		newNode Node
	}{
		"nil parent": {
			trie: Trie{
				generation: 1,
			},
			key:   []byte{1},
			value: &node.Leaf{},
			newNode: &node.Leaf{
				Key:        []byte{1},
				Generation: 1,
			},
		},
		"branch parent": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte("branch"),
				Children: [16]node.Node{
					nil,
					&node.Leaf{Key: []byte{2}},
				},
			},
			key: []byte{1, 0},
			value: &node.Leaf{
				Value: []byte("leaf"),
			},
			newNode: &node.Branch{
				Key:        []byte{1},
				Value:      []byte("branch"),
				Generation: 1,
				Dirty:      true,
				Children: [16]node.Node{
					&node.Leaf{
						Key:        []byte{},
						Value:      []byte("leaf"),
						Generation: 1,
						Dirty:      true,
					},
					&node.Leaf{Key: []byte{2}},
				},
			},
		},
		"override leaf parent": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key:   []byte{1},
				Value: []byte("original leaf"),
			},
			key: []byte{1},
			value: &node.Leaf{
				Value: []byte("new leaf"),
			},
			newNode: &node.Leaf{
				Key:        []byte{1},
				Value:      []byte("new leaf"),
				Generation: 1,
				Dirty:      true,
			},
		},
		"write same leaf value as child to parent leaf": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key:   []byte{1},
				Value: []byte("same"),
			},
			key: []byte{1},
			value: &node.Leaf{
				Value: []byte("same"),
			},
			newNode: &node.Leaf{
				Key:        []byte{1},
				Value:      []byte("same"),
				Generation: 1,
			},
		},
		"write leaf as child to parent leaf": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key:   []byte{1},
				Value: []byte("original leaf"),
			},
			key: []byte{1, 0},
			value: &node.Leaf{
				Value: []byte("leaf"),
			},
			newNode: &node.Branch{
				Key:        []byte{1},
				Value:      []byte("original leaf"),
				Dirty:      true,
				Generation: 1,
				Children: [16]node.Node{
					&node.Leaf{
						Key:        []byte{},
						Value:      []byte("leaf"),
						Generation: 1,
					},
				},
			},
		},
		"write leaf as divergent child next to parent leaf": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key:   []byte{1, 2},
				Value: []byte("original leaf"),
			},
			key: []byte{2, 3},
			value: &node.Leaf{
				Value: []byte("leaf"),
			},
			newNode: &node.Branch{
				Key:        []byte{},
				Dirty:      true,
				Generation: 1,
				Children: [16]node.Node{
					nil,
					&node.Leaf{
						Key:        []byte{2},
						Value:      []byte("original leaf"),
						Dirty:      true,
						Generation: 1,
					},
					&node.Leaf{
						Key:        []byte{3},
						Value:      []byte("leaf"),
						Generation: 1,
					},
				},
			},
		},
		"write leaf into nil leaf": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key: []byte{1},
			},
			key: []byte{1},
			value: &node.Leaf{
				Value: []byte("leaf"),
			},
			newNode: &node.Leaf{
				Key:        []byte{1},
				Value:      []byte("leaf"),
				Dirty:      true,
				Generation: 1,
			},
		},
		"write leaf as child to nil value leaf": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key: []byte{1, 2},
			},
			key: []byte{1},
			value: &node.Leaf{
				Value: []byte("leaf"),
			},
			newNode: &node.Branch{
				Key:        []byte{1},
				Value:      []byte("leaf"),
				Dirty:      true,
				Generation: 1,
				Children: [16]node.Node{
					nil, nil,
					&node.Leaf{
						Key:        []byte{},
						Dirty:      true,
						Generation: 1,
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newNode := trie.insert(testCase.parent, testCase.key, testCase.value)

			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, expectedTrie, trie)
		})
	}
}

func Test_Trie_insertInBranch(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		parent  *node.Branch
		key     []byte
		value   Node
		newNode Node
	}{
		"update with branch": {
			parent: &node.Branch{
				Key:   []byte{2},
				Value: []byte("old"),
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			key: []byte{2},
			value: &node.Branch{
				Value: []byte("new"),
			},
			newNode: &node.Branch{
				Key:   []byte{2},
				Value: []byte("new"),
				Dirty: true,
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
		},
		"update with leaf": {
			parent: &node.Branch{
				Key:   []byte{2},
				Value: []byte("old"),
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			key: []byte{2},
			value: &node.Leaf{
				Value: []byte("new"),
			},
			newNode: &node.Branch{
				Key:   []byte{2},
				Value: []byte("new"),
				Dirty: true,
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
		},
		"add leaf as direct child": {
			parent: &node.Branch{
				Key:   []byte{2},
				Value: []byte{5},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			key: []byte{2, 3, 4, 5},
			value: &node.Leaf{
				Value: []byte{6},
			},
			newNode: &node.Branch{
				Key:   []byte{2},
				Value: []byte{5},
				Dirty: true,
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					nil, nil,
					&node.Leaf{
						Key:   []byte{4, 5},
						Value: []byte{6},
						Dirty: true,
					},
				},
			},
		},
		"add leaf as nested child": {
			parent: &node.Branch{
				Key:   []byte{2},
				Value: []byte{5},
				Children: [16]node.Node{
					nil, nil, nil,
					&node.Branch{
						Key: []byte{4},
						Children: [16]node.Node{
							&node.Leaf{Key: []byte{1}},
						},
					},
				},
			},
			key: []byte{2, 3, 4, 5, 6},
			value: &node.Leaf{
				Value: []byte{6},
			},
			newNode: &node.Branch{
				Key:   []byte{2},
				Value: []byte{5},
				Dirty: true,
				Children: [16]node.Node{
					nil, nil, nil,
					&node.Branch{
						Key:   []byte{4},
						Dirty: true,
						Children: [16]node.Node{
							&node.Leaf{Key: []byte{1}},
							nil, nil, nil, nil,
							&node.Leaf{
								Key:   []byte{6},
								Value: []byte{6},
								Dirty: true,
							},
						},
					},
				},
			},
		},
		"split branch for longer key": {
			parent: &node.Branch{
				Key:   []byte{2, 3},
				Value: []byte{5},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			key: []byte{2, 4, 5, 6},
			value: &node.Leaf{
				Value: []byte{6},
			},
			newNode: &node.Branch{
				Key:   []byte{2},
				Dirty: true,
				Children: [16]node.Node{
					nil, nil, nil,
					&node.Branch{
						Key:   []byte{},
						Value: []byte{5},
						Dirty: true,
						Children: [16]node.Node{
							&node.Leaf{Key: []byte{1}},
						},
					},
					&node.Leaf{
						Key:   []byte{5, 6},
						Value: []byte{6},
					},
				},
			},
		},
		"split root branch": {
			parent: &node.Branch{
				Key:   []byte{2, 3},
				Value: []byte{5},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			key: []byte{3},
			value: &node.Leaf{
				Value: []byte{6},
			},
			newNode: &node.Branch{
				Key:   []byte{},
				Dirty: true,
				Children: [16]node.Node{
					nil, nil,
					&node.Branch{
						Key:   []byte{3},
						Value: []byte{5},
						Dirty: true,
						Children: [16]node.Node{
							&node.Leaf{Key: []byte{1}},
						},
					},
					&node.Leaf{
						Key:   []byte{},
						Value: []byte{6},
					},
				},
			},
		},
		"update with leaf at empty key": {
			parent: &node.Branch{
				Key:   []byte{2},
				Value: []byte{5},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			key: []byte{},
			value: &node.Leaf{
				Value: []byte{6},
			},
			newNode: &node.Branch{
				Key:   []byte{},
				Value: []byte{6},
				Dirty: true,
				Children: [16]node.Node{
					nil, nil,
					&node.Branch{
						Key:   []byte{},
						Value: []byte{5},
						Dirty: true,
						Children: [16]node.Node{
							&node.Leaf{Key: []byte{1}},
						},
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := new(Trie)

			newNode := trie.insertInBranch(testCase.parent, testCase.key, testCase.value)

			assert.Equal(t, testCase.newNode, newNode)
			assert.Equal(t, new(Trie), trie) // check no mutation
		})
	}
}

func Test_Trie_LoadFromMap(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie         Trie
		data         map[string]string
		expectedTrie Trie
		errWrapped   error
		errMessage   string
	}{
		"nil data": {},
		"empty data": {
			data: map[string]string{},
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
		"load into empty trie": {
			data: map[string]string{
				"0x01":   "0x06",
				"0x0120": "0x07",
				"0x0130": "0x08",
			},
			expectedTrie: Trie{
				root: &node.Branch{
					Key:   []byte{00, 01},
					Value: []byte{6},
					Dirty: true,
					Children: [16]node.Node{
						nil, nil,
						&node.Leaf{
							Key:   []byte{0},
							Value: []byte{7},
							Dirty: true,
						},
						&node.Leaf{
							Key:   []byte{0},
							Value: []byte{8},
							Dirty: true,
						},
					},
				},
			},
		},
		"override trie": {
			trie: Trie{
				root: &node.Branch{
					Key:   []byte{00, 01},
					Value: []byte{106},
					Dirty: true,
					Children: [16]node.Node{
						&node.Leaf{
							Value: []byte{9},
						},
						nil,
						&node.Leaf{
							Key:   []byte{0},
							Value: []byte{107},
							Dirty: true,
						},
					},
				},
			},
			data: map[string]string{
				"0x01":   "0x06",
				"0x0120": "0x07",
				"0x0130": "0x08",
			},
			expectedTrie: Trie{
				root: &node.Branch{
					Key:   []byte{00, 01},
					Value: []byte{6},
					Dirty: true,
					Children: [16]node.Node{
						&node.Leaf{
							Value: []byte{9},
						},
						nil,
						&node.Leaf{
							Key:   []byte{0},
							Value: []byte{7},
							Dirty: true,
						},
						&node.Leaf{
							Key:   []byte{0},
							Value: []byte{8},
							Dirty: true,
						},
					},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := testCase.trie.LoadFromMap(testCase.data)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}

			assert.Equal(t, testCase.expectedTrie, testCase.trie)
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
				root: &node.Branch{
					Key: []byte{0, 1},
					Children: [16]node.Node{
						&node.Branch{ // full key 0, 1, 0, 3
							Key: []byte{3},
							Children: [16]node.Node{
								&node.Leaf{ // full key 0, 1, 0, 0, 4
									Key: []byte{4},
								},
								&node.Leaf{ // full key 0, 1, 0, 1, 5
									Key: []byte{5},
								},
							},
						},
						&node.Leaf{ // full key 0, 1, 1, 9
							Key: []byte{9},
						},
					},
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
		parent       Node
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
			parent: &node.Branch{
				Key: []byte{1, 2, 3},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{4}},
					&node.Leaf{Key: []byte{5}},
				},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"parent branch and empty key": {
			parent: &node.Branch{
				Key: []byte{1, 2, 3},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{4}},
					&node.Leaf{Key: []byte{5}},
				},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"search key smaller than branch key with no full common prefix": {
			parent: &node.Branch{
				Key: []byte{1, 2, 3},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{4}},
					&node.Leaf{Key: []byte{5}},
				},
			},
			key:          []byte{1, 3},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"common prefix smaller tan search key": {
			parent: &node.Branch{
				Key: []byte{1, 2},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{4}},
					&node.Leaf{Key: []byte{5}},
				},
			},
			key:          []byte{1, 2, 3},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"recursive call": {
			parent: &node.Branch{
				Key: []byte{1, 2, 3},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{4}},
					&node.Leaf{Key: []byte{5}},
				},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2, 3, 0},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04}},
		},
		"parent leaf with search key equal to common prefix": {
			parent: &node.Leaf{
				Key: []byte{1, 2, 3},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2, 3},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent leaf with empty search key": {
			parent: &node.Leaf{
				Key: []byte{1, 2, 3},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent leaf with too deep search key": {
			parent: &node.Leaf{
				Key: []byte{1, 2, 3},
			},
			prefix:       []byte{9, 8, 7},
			key:          []byte{1, 2, 3, 4},
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"parent leaf with shorter matching search key": {
			parent: &node.Leaf{
				Key: []byte{1, 2, 3},
			},
			prefix: []byte{9, 8, 7},
			key:    []byte{1, 2},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent leaf with not matching search key": {
			parent: &node.Leaf{
				Key: []byte{1, 2, 3},
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
		parent       Node
		prefix       []byte
		keys         [][]byte
		expectedKeys [][]byte
	}{
		"nil parent returns keys passed": {
			keys:         [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2}},
		},
		"leaf parent": {
			parent: &node.Leaf{
				Key: []byte{1, 2, 3},
			},
			prefix: []byte{9, 8, 7},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23}},
		},
		"parent branch without value": {
			parent: &node.Branch{
				Key: []byte{1, 2, 3},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{4}},
					&node.Leaf{Key: []byte{5}},
				},
			},
			prefix: []byte{9, 8, 7},
			keys:   [][]byte{{1}, {2}},
			expectedKeys: [][]byte{{1}, {2},
				{0x98, 0x71, 0x23, 0x04},
				{0x98, 0x71, 0x23, 0x15}},
		},
		"parent branch with empty value": {
			parent: &node.Branch{
				Key:   []byte{1, 2, 3},
				Value: []byte{},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{4}},
					&node.Leaf{Key: []byte{5}},
				},
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
				root: &node.Branch{
					Key:   []byte{0, 1},
					Value: []byte{1, 3},
					Children: [16]node.Node{
						&node.Branch{ // full key 0, 1, 0, 3
							Key:   []byte{3},
							Value: []byte{1, 2},
							Children: [16]node.Node{
								&node.Leaf{Key: []byte{1}},
							},
						},
						&node.Leaf{ // full key 0, 1, 1, 9
							Key:   []byte{9},
							Value: []byte{1, 2, 3, 4, 5},
						},
					},
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
		parent Node
		key    []byte
		value  []byte
	}{
		"nil parent": {
			key: []byte{1},
		},
		"leaf key match": {
			parent: &node.Leaf{
				Key:   []byte{1},
				Value: []byte{2},
			},
			key:   []byte{1},
			value: []byte{2},
		},
		"leaf key mismatch": {
			parent: &node.Leaf{
				Key:   []byte{1, 2},
				Value: []byte{2},
			},
			key: []byte{1},
		},
		"branch key match": {
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{2},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			key:   []byte{1},
			value: []byte{2},
		},
		"branch key with empty search key": {
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{2},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			value: []byte{2},
		},
		"branch key mismatch with shorter search key": {
			parent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{2},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			key: []byte{1},
		},
		"bottom leaf in branch": {
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					nil, nil,
					&node.Branch{ // full key 1, 2, 3
						Key:   []byte{3},
						Value: []byte{2},
						Children: [16]node.Node{
							nil, nil, nil, nil,
							&node.Leaf{ // full key 1, 2, 3, 4, 5
								Key:   []byte{5},
								Value: []byte{3},
							},
						},
					},
				},
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
			const copyChildren = true
			var expectedParent Node
			if testCase.parent != nil {
				expectedParent = testCase.parent.Copy(copyChildren)
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
				root: &node.Branch{
					Key:   []byte{1, 2},
					Value: []byte{1},
					Children: [16]node.Node{
						nil, nil, nil,
						&node.Leaf{
							Key: []byte{4},
						},
					},
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

func Test_Trie_clearPrefixLimit(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie          Trie
		parent        Node
		prefix        []byte
		limit         uint32
		expectedLimit uint32
		newParent     Node
		updated       bool
		allDeleted    bool
	}{
		"limit is zero": {
			allDeleted: true,
		},
		"nil parent": {
			limit:         1,
			expectedLimit: 1,
			allDeleted:    true,
		},
		"leaf parent with common prefix": {
			parent: &node.Leaf{
				Key: []byte{1, 2},
			},
			prefix:     []byte{1},
			limit:      1,
			updated:    true,
			allDeleted: true,
		},
		"leaf parent with key equal prefix": {
			parent: &node.Leaf{
				Key: []byte{1},
			},
			prefix:     []byte{1},
			limit:      1,
			updated:    true,
			allDeleted: true,
		},
		"leaf parent with key no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key: []byte{1, 2},
			},
			prefix:        []byte{1, 3},
			limit:         1,
			expectedLimit: 1,
			newParent: &node.Leaf{
				Key: []byte{1, 2},
			},
			allDeleted: true,
		},
		"leaf parent with key smaller than prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key: []byte{1},
			},
			prefix:        []byte{1, 2},
			limit:         1,
			expectedLimit: 1,
			newParent: &node.Leaf{
				Key: []byte{1},
			},
			allDeleted: true,
		},
		"branch without value parent with common prefix": {
			parent: &node.Branch{
				Key: []byte{1, 2},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			prefix:        []byte{1},
			limit:         3,
			expectedLimit: 1,
			updated:       true,
			allDeleted:    true,
		},
		"branch without value with key equal prefix": {
			parent: &node.Branch{
				Key: []byte{1, 2},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			prefix:        []byte{1, 2},
			limit:         3,
			expectedLimit: 1,
			updated:       true,
			allDeleted:    true,
		},
		"branch without value with no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key: []byte{1, 2},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			prefix:        []byte{1, 3},
			limit:         1,
			expectedLimit: 1,
			newParent: &node.Branch{
				Key: []byte{1, 2},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			allDeleted: true,
		},
		"branch without value with key smaller than prefix by more than one": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			prefix:        []byte{1, 2, 3},
			limit:         1,
			expectedLimit: 1,
			newParent: &node.Branch{
				Key: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			allDeleted: true,
		},
		"branch without value with key smaller than prefix by one": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			prefix:        []byte{1, 2},
			limit:         1,
			expectedLimit: 1,
			newParent: &node.Branch{
				Key: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			allDeleted: true,
		},
		"branch with value with common prefix": {
			parent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			prefix:     []byte{1},
			limit:      2,
			updated:    true,
			allDeleted: true,
		},
		"branch with value with key equal prefix": {
			parent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			prefix:     []byte{1, 2},
			limit:      2,
			updated:    true,
			allDeleted: true,
		},
		"branch with value with no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			prefix:        []byte{1, 3},
			limit:         1,
			expectedLimit: 1,
			newParent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			allDeleted: true,
		},
		"branch with value with key smaller than prefix by more than one": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			prefix:        []byte{1, 2, 3},
			limit:         1,
			expectedLimit: 1,
			newParent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			allDeleted: true,
		},
		"branch with value with key smaller than prefix by one": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			prefix:        []byte{1, 2},
			limit:         1,
			expectedLimit: 1,
			newParent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
				},
			},
			allDeleted: true,
		},
		"delete one child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{3}},
					&node.Leaf{Key: []byte{4}},
				},
			},
			prefix: []byte{1},
			limit:  1,
			newParent: &node.Branch{
				Key:        []byte{1},
				Value:      []byte{1},
				Dirty:      true,
				Generation: 1,
				Children: [16]node.Node{
					nil,
					&node.Leaf{Key: []byte{4}},
				},
			},
			updated: true,
		},
		"delete only child of branch": {
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{3}},
				},
			},
			prefix: []byte{1, 0},
			limit:  1,
			newParent: &node.Leaf{
				Key:   []byte{1},
				Value: []byte{1},
				Dirty: true,
			},
			updated:    true,
			allDeleted: true,
		},
		"fully delete children of branch with value": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{3}},
					&node.Leaf{Key: []byte{4}},
				},
			},
			prefix: []byte{1},
			limit:  2,
			newParent: &node.Leaf{
				Key:        []byte{1},
				Value:      []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			updated: true,
		},
		"fully delete children of branch without value": {
			parent: &node.Branch{
				Key: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{3}},
					&node.Leaf{Key: []byte{4}},
				},
			},
			prefix:     []byte{1},
			limit:      2,
			updated:    true,
			allDeleted: true,
		},
		"partially delete child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Branch{ // full key 1, 0, 3
						Key:   []byte{3},
						Value: []byte{1},
						Children: [16]node.Node{
							&node.Leaf{ // full key 1, 0, 3, 0, 5
								Key: []byte{5},
							},
						},
					},
					&node.Leaf{
						Key: []byte{6},
					},
				},
			},
			prefix: []byte{1, 0},
			limit:  1,
			newParent: &node.Branch{
				Key:        []byte{1},
				Value:      []byte{1},
				Dirty:      true,
				Generation: 1,
				Children: [16]node.Node{
					&node.Leaf{ // full key 1, 0, 3
						Key:        []byte{3},
						Value:      []byte{1},
						Dirty:      true,
						Generation: 1,
					},
					&node.Leaf{
						Key: []byte{6},
						// Not modified so same generation as before
					},
				},
			},
			updated: true,
		},
		"update child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Branch{ // full key 1, 0, 2
						Key:   []byte{2},
						Value: []byte{1},
						Children: [16]node.Node{
							&node.Leaf{Key: []byte{1}},
						},
					},
				},
			},
			prefix: []byte{1, 0, 2},
			limit:  2,
			newParent: &node.Leaf{
				Key:        []byte{1},
				Value:      []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			updated:    true,
			allDeleted: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newParent, updated, allDeleted := trie.clearPrefixLimit(testCase.parent,
				testCase.prefix, &testCase.limit)

			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.expectedLimit, testCase.limit)
			assert.Equal(t, testCase.updated, updated)
			assert.Equal(t, testCase.allDeleted, allDeleted)
			assert.Equal(t, expectedTrie, trie)
		})
	}
}

func Test_Trie_deleteNodes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie        Trie
		parent      Node
		prefix      []byte
		limit       uint32
		newLimit    uint32
		newNode     Node
		oneDeletion bool
	}{
		"zero limit": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key: []byte{1},
			},
			newNode: &node.Leaf{
				Key: []byte{1},
			},
		},
		"nil parent": {
			limit:    1,
			newLimit: 1,
		},
		"delete leaf": {
			parent:   &node.Leaf{},
			limit:    2,
			newLimit: 1,
		},
		"delete branch without value": {
			parent: &node.Branch{
				Children: [16]node.Node{
					&node.Leaf{},
					&node.Leaf{},
				},
			},
			limit:    3,
			newLimit: 1,
		},
		"delete branch with value": {
			parent: &node.Branch{
				Key:   []byte{3},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
			limit:    3,
			newLimit: 1,
		},
		"delete branch and all children": {
			parent: &node.Branch{
				Key: []byte{3},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			limit:    10,
			newLimit: 8,
		},
		"delete branch one child only": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{3},
				Value: []byte{1, 2, 3},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			limit: 1,
			newNode: &node.Branch{
				Key:        []byte{3},
				Value:      []byte{1, 2, 3},
				Dirty:      true,
				Generation: 1,
				Children: [16]node.Node{
					nil,
					&node.Leaf{Key: []byte{2}},
				},
			},
		},
		"delete branch children only": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{3},
				Value: []byte{1, 2, 3},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{1}},
					&node.Leaf{Key: []byte{2}},
				},
			},
			limit:    2,
			newLimit: 0,
			newNode: &node.Leaf{
				Key:        []byte{3},
				Value:      []byte{1, 2, 3},
				Dirty:      true,
				Generation: 1,
			},
		},
		"delete branch all children except one": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key: []byte{3},
				Children: [16]node.Node{
					nil,
					&node.Leaf{Key: []byte{1}},
					nil,
					&node.Leaf{Key: []byte{2}},
					nil,
					&node.Leaf{Key: []byte{3}},
				},
			},
			prefix:   []byte{1, 2},
			limit:    2,
			newLimit: 0,
			newNode: &node.Leaf{
				Key:        []byte{3, 5, 3},
				Generation: 1,
				Dirty:      true,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newNode := trie.deleteNodesLimit(testCase.parent, testCase.prefix, &testCase.limit)

			assert.Equal(t, testCase.limit, testCase.limit)
			assert.Equal(t, testCase.newNode, newNode)
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
				root: &node.Leaf{},
			},
		},
		"empty prefix": {
			trie: Trie{
				root: &node.Leaf{},
			},
			prefix: []byte{},
		},
		"empty trie": {
			prefix: []byte{0x12},
		},
		"clear prefix": {
			trie: Trie{
				root: &node.Branch{
					Key: []byte{1, 2},
					Children: [16]node.Node{
						&node.Leaf{ // full key in nibbles 1, 2, 0, 5
							Key: []byte{5},
						},
						&node.Branch{ // full key in nibbles 1, 2, 1, 6
							Key:   []byte{6},
							Value: []byte("bottom branch"),
							Children: [16]node.Node{
								&node.Leaf{ // full key in nibbles 1, 2, 1, 6, 0, 7
									Key: []byte{7},
								},
							},
						},
					},
				},
			},
			prefix: []byte{0x12, 0x16},
			expectedTrie: Trie{
				root: &node.Leaf{
					Key:   []byte{1, 2, 0, 5},
					Dirty: true,
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

func Test_Trie_clearPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie      Trie
		parent    Node
		prefix    []byte
		newParent Node
		updated   bool
	}{
		"nil parent": {},
		"leaf parent with common prefix": {
			parent: &node.Leaf{
				Key: []byte{1, 2},
			},
			prefix:  []byte{1},
			updated: true,
		},
		"leaf parent with key equal prefix": {
			parent: &node.Leaf{
				Key: []byte{1},
			},
			prefix:  []byte{1},
			updated: true,
		},
		"leaf parent with key no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key: []byte{1, 2},
			},
			prefix: []byte{1, 3},
			newParent: &node.Leaf{
				Key: []byte{1, 2},
			},
		},
		"leaf parent with key smaller than prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key: []byte{1},
			},
			prefix: []byte{1, 2},
			newParent: &node.Leaf{
				Key: []byte{1},
			},
		},
		"branch parent with common prefix": {
			parent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
			prefix:  []byte{1},
			updated: true,
		},
		"branch with key equal prefix": {
			parent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
			prefix:  []byte{1, 2},
			updated: true,
		},
		"branch with no common prefix": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
			prefix: []byte{1, 3},
			newParent: &node.Branch{
				Key:   []byte{1, 2},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
		},
		"branch with key smaller than prefix by more than one": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
			prefix: []byte{1, 2, 3},
			newParent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
		},
		"branch with key smaller than prefix by one": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
			prefix: []byte{1, 2},
			newParent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
		},
		"delete one child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{3}},
					&node.Leaf{Key: []byte{4}},
				},
			},
			prefix: []byte{1, 0, 3},
			newParent: &node.Branch{
				Key:        []byte{1},
				Value:      []byte{1},
				Dirty:      true,
				Generation: 1,
				Children: [16]node.Node{
					nil,
					&node.Leaf{Key: []byte{4}},
				},
			},
			updated: true,
		},
		"fully delete child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{3}},
				},
			},
			prefix: []byte{1, 0},
			newParent: &node.Leaf{
				Key:        []byte{1},
				Value:      []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			updated: true,
		},
		"partially delete child of branch": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Branch{ // full key 1, 0, 3
						Key:   []byte{3},
						Value: []byte{1},
						Children: [16]node.Node{
							&node.Leaf{ // full key 1, 0, 3, 0, 5
								Key: []byte{5},
							},
						},
					},
				},
			},
			prefix: []byte{1, 0, 3, 0},
			newParent: &node.Branch{
				Key:        []byte{1},
				Value:      []byte{1},
				Dirty:      true,
				Generation: 1,
				Children: [16]node.Node{
					&node.Leaf{ // full key 1, 0, 3
						Key:        []byte{3},
						Value:      []byte{1},
						Dirty:      true,
						Generation: 1,
					},
				},
			},
			updated: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trie := testCase.trie
			expectedTrie := *trie.DeepCopy()

			newParent, updated := trie.clearPrefix(testCase.parent,
				testCase.prefix)

			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.updated, updated)
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
				root: &node.Leaf{},
			},
		},
		"empty key": {
			trie: Trie{
				root: &node.Leaf{},
			},
		},
		"empty trie": {
			key: []byte{0x12},
		},
		"delete branch node": {
			trie: Trie{
				generation: 1,
				root: &node.Branch{
					Key: []byte{1, 2},
					Children: [16]node.Node{
						&node.Leaf{
							Key:   []byte{5},
							Value: []byte{97},
						},
						&node.Branch{ // full key in nibbles 1, 2, 1, 6
							Key:   []byte{6},
							Value: []byte{98},
							Children: [16]node.Node{
								&node.Leaf{ // full key in nibbles 1, 2, 1, 6, 0, 7
									Key:   []byte{7},
									Value: []byte{99},
								},
							},
						},
					},
				},
			},
			key: []byte{0x12, 0x16},
			expectedTrie: Trie{
				generation: 1,
				root: &node.Branch{
					Key:        []byte{1, 2},
					Dirty:      true,
					Generation: 1,
					Children: [16]node.Node{
						&node.Leaf{
							Key:   []byte{5},
							Value: []byte{97},
						},
						&node.Leaf{ // full key in nibbles 1, 2, 1, 6
							Key:        []byte{6, 0, 7},
							Value:      []byte{99},
							Dirty:      true,
							Generation: 1,
						},
					},
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

func Test_Trie_delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		trie      Trie
		parent    Node
		key       []byte
		newParent Node
		updated   bool
	}{
		"nil parent": {
			key: []byte{1},
		},
		"leaf parent and nil key": {
			parent: &node.Leaf{
				Key: []byte{1},
			},
			updated: true,
		},
		"leaf parent and empty key": {
			parent: &node.Leaf{
				Key: []byte{1},
			},
			key:     []byte{},
			updated: true,
		},
		"leaf parent matches key": {
			parent: &node.Leaf{
				Key: []byte{1},
			},
			key:     []byte{1},
			updated: true,
		},
		"leaf parent mismatches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Leaf{
				Key: []byte{1},
			},
			key: []byte{2},
			newParent: &node.Leaf{
				Key: []byte{1},
			},
		},
		"branch parent and nil key": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{
						Key: []byte{2},
					},
				},
			},
			newParent: &node.Leaf{
				Key:        []byte{1, 0, 2},
				Dirty:      true,
				Generation: 1,
			},
			updated: true,
		},
		"branch parent and empty key": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{
						Key: []byte{2},
					},
				},
			},
			key: []byte{},
			newParent: &node.Leaf{
				Key:        []byte{1, 0, 2},
				Dirty:      true,
				Generation: 1,
			},
			updated: true,
		},
		"branch parent matches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{
						Key: []byte{2},
					},
				},
			},
			key: []byte{1},
			newParent: &node.Leaf{
				Key:        []byte{1, 0, 2},
				Dirty:      true,
				Generation: 1,
			},
			updated: true,
		},
		"branch parent child matches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{ // full key 1, 0, 2
						Key: []byte{2},
					},
				},
			},
			key: []byte{1, 0, 2},
			newParent: &node.Leaf{
				Key:        []byte{1},
				Value:      []byte{1},
				Dirty:      true,
				Generation: 1,
			},
			updated: true,
		},
		"branch parent mismatches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
			key: []byte{2},
			newParent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{},
				},
			},
		},
		"branch parent child mismatches key": {
			trie: Trie{
				generation: 1,
			},
			parent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{ // full key 1, 0, 2
						Key: []byte{2},
					},
				},
			},
			key: []byte{1, 0, 3},
			newParent: &node.Branch{
				Key:   []byte{1},
				Value: []byte{1},
				Children: [16]node.Node{
					&node.Leaf{ // full key 1, 0, 2
						Key: []byte{2},
					},
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
			expectedTrie := *testCase.trie.DeepCopy()

			newParent, updated := testCase.trie.delete(testCase.parent, testCase.key)

			assert.Equal(t, testCase.newParent, newParent)
			assert.Equal(t, testCase.updated, updated)
			assert.Equal(t, expectedTrie, testCase.trie)
			assert.Equal(t, expectedKey, testCase.key)
		})
	}
}

func Test_handleDeletion(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		branch     *node.Branch
		deletedKey []byte
		newNode    Node
	}{
		"branch with value and without children": {
			branch: &node.Branch{
				Key:        []byte{1, 2, 3},
				Value:      []byte{5, 6, 7},
				Generation: 1,
			},
			deletedKey: []byte{1, 2, 3, 4},
			newNode: &node.Leaf{
				Key:        []byte{1, 2, 3},
				Value:      []byte{5, 6, 7},
				Generation: 1,
				Dirty:      true,
			},
		},
		// branch without value and without children cannot happen
		// since it would be turned into a leaf when it only has one child
		// remaining.
		"branch with value and a single child": {
			branch: &node.Branch{
				Key:        []byte{1, 2, 3},
				Value:      []byte{5, 6, 7},
				Generation: 1,
				Children: [16]node.Node{
					nil,
					&node.Leaf{Key: []byte{9}},
				},
			},
			newNode: &node.Branch{
				Key:        []byte{1, 2, 3},
				Value:      []byte{5, 6, 7},
				Generation: 1,
				Children: [16]node.Node{
					nil,
					&node.Leaf{Key: []byte{9}},
				},
			},
		},
		"branch without value and a single leaf child": {
			branch: &node.Branch{
				Key:        []byte{1, 2, 3},
				Generation: 1,
				Children: [16]node.Node{
					nil,
					&node.Leaf{ // full key 1,2,3,1,9
						Key:   []byte{9},
						Value: []byte{10},
					},
				},
			},
			deletedKey: []byte{1, 2, 3, 4},
			newNode: &node.Leaf{
				Key:        []byte{1, 2, 3, 1, 9},
				Value:      []byte{10},
				Generation: 1,
				Dirty:      true,
			},
		},
		"branch without value and a single branch child": {
			branch: &node.Branch{
				Key:        []byte{1, 2, 3},
				Generation: 1,
				Children: [16]node.Node{
					nil,
					&node.Branch{
						Key:   []byte{9},
						Value: []byte{10},
						Children: [16]node.Node{
							&node.Leaf{Key: []byte{7}},
							nil,
							&node.Leaf{Key: []byte{8}},
						},
					},
				},
			},
			newNode: &node.Branch{
				Key:        []byte{1, 2, 3, 1, 9},
				Value:      []byte{10},
				Generation: 1,
				Dirty:      true,
				Children: [16]node.Node{
					&node.Leaf{Key: []byte{7}},
					nil,
					&node.Leaf{Key: []byte{8}},
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
			if testCase.deletedKey != nil {
				expectedKey = make([]byte, len(testCase.deletedKey))
				copy(expectedKey, testCase.deletedKey)
			}

			newNode := handleDeletion(testCase.branch, testCase.deletedKey)

			assert.Equal(t, testCase.newNode, newNode)
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
			concatenated := append(slice1, slice2...)
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
