// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/assert"
)

func Test_newTries(t *testing.T) {
	t.Parallel()

	tr := trie.NewEmptyTrie()

	rootToTrie := newTries(tr)

	expectedTries := &tries{
		rootToTrie: map[common.Hash]*trie.Trie{
			tr.MustHash(): tr,
		},
	}

	assert.Equal(t, expectedTries, rootToTrie)
}

func Test_tries_softSet(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tries         *tries
		root          common.Hash
		trie          *trie.Trie
		expectedTries *tries
	}{
		"set new in map": {
			tries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{},
			},
			root: common.Hash{1, 2, 3},
			trie: trie.NewEmptyTrie(),
			expectedTries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{
					{1, 2, 3}: trie.NewEmptyTrie(),
				},
			},
		},
		"do not override in map": {
			tries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{
					{1, 2, 3}: {},
				},
			},
			root: common.Hash{1, 2, 3},
			trie: trie.NewEmptyTrie(),
			expectedTries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{
					{1, 2, 3}: {},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.tries.softSet(testCase.root, testCase.trie)

			assert.Equal(t, testCase.expectedTries, testCase.tries)
		})
	}
}

func Test_tries_delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tries         *tries
		root          common.Hash
		expectedTries *tries
	}{
		"not found": {
			tries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{},
			},
			root: common.Hash{1, 2, 3},
			expectedTries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{},
			},
		},
		"deleted": {
			tries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{
					{1, 2, 3}: {},
				},
			},
			root: common.Hash{1, 2, 3},
			expectedTries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.tries.delete(testCase.root)

			assert.Equal(t, testCase.expectedTries, testCase.tries)
		})
	}
}
func Test_tries_get(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tries *tries
		root  common.Hash
		trie  *trie.Trie
	}{
		"found in map": {
			tries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{
					{1, 2, 3}: trie.NewTrie(&node.Leaf{
						Key: []byte{1, 2, 3},
					}),
				},
			},
			root: common.Hash{1, 2, 3},
			trie: trie.NewTrie(&node.Leaf{
				Key: []byte{1, 2, 3},
			}),
		},
		"not found in map": {
			// similar to not found in database
			tries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{},
			},
			root: common.Hash{1, 2, 3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			trieFound := testCase.tries.get(testCase.root)

			assert.Equal(t, testCase.trie, trieFound)
		})
	}
}

func Test_tries_len(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tries  *tries
		length int
	}{
		"empty map": {
			tries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{},
			},
		},
		"non empty map": {
			tries: &tries{
				rootToTrie: map[common.Hash]*trie.Trie{
					{1, 2, 3}: {},
				},
			},
			length: 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			length := testCase.tries.len()

			assert.Equal(t, testCase.length, length)
		})
	}
}
