// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_NewTries(t *testing.T) {
	t.Parallel()

	rootToTrie := NewTries()

	expectedTries := &Tries{
		rootToTrie:    map[common.Hash]*trie.Trie{},
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}

	assert.Equal(t, expectedTries, rootToTrie)
}

func Test_Tries_SetEmptyTrie(t *testing.T) {
	t.Parallel()

	tries := NewTries()
	tries.SetEmptyTrie()

	expectedTries := &Tries{
		rootToTrie: map[common.Hash]*trie.Trie{
			trie.EmptyHash: trie.NewEmptyTrie(),
		},
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}

	assert.Equal(t, expectedTries, tries)
}

func Test_Tries_SetTrie(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	db := NewMockDatabase(ctrl)
	db.EXPECT().Get(gomock.Any()).Times(0)

	tr := trie.NewTrie(&node.Node{PartialKey: []byte{1}}, db)

	tries := NewTries()
	tries.SetTrie(tr)

	expectedTries := &Tries{
		rootToTrie: map[common.Hash]*trie.Trie{
			tr.MustHash(): tr,
		},
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}

	assert.Equal(t, expectedTries, tries)
}

func Test_Tries_softSet(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		rootToTrie         map[common.Hash]*trie.Trie
		root               common.Hash
		trie               *trie.Trie
		triesGaugeInc      bool
		expectedRootToTrie map[common.Hash]*trie.Trie
	}{
		"set_new_in_map": {
			rootToTrie:    map[common.Hash]*trie.Trie{},
			root:          common.Hash{1, 2, 3},
			trie:          trie.NewEmptyTrie(),
			triesGaugeInc: true,
			expectedRootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: trie.NewEmptyTrie(),
			},
		},
		"do_not_override_in_map": {
			rootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: {},
			},
			root: common.Hash{1, 2, 3},
			trie: trie.NewEmptyTrie(),
			expectedRootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: {},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			triesGauge := NewMockGauge(ctrl)
			if testCase.triesGaugeInc {
				triesGauge.EXPECT().Inc()
			}

			setCounter := NewMockCounter(ctrl)
			if testCase.triesGaugeInc {
				setCounter.EXPECT().Inc()
			}

			tries := &Tries{
				rootToTrie: testCase.rootToTrie,
				triesGauge: triesGauge,
				setCounter: setCounter,
			}

			tries.softSet(testCase.root, testCase.trie)

			assert.Equal(t, testCase.expectedRootToTrie, tries.rootToTrie)
		})
	}
}

func Test_Tries_delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		rootToTrie         map[common.Hash]*trie.Trie
		root               common.Hash
		deleteCounterInc   bool
		expectedRootToTrie map[common.Hash]*trie.Trie
		triesGaugeSet      float64
	}{
		"not_found": {
			rootToTrie: map[common.Hash]*trie.Trie{
				{3, 4, 5}: {},
			},
			root:          common.Hash{1, 2, 3},
			triesGaugeSet: 1,
			expectedRootToTrie: map[common.Hash]*trie.Trie{
				{3, 4, 5}: {},
			},
			deleteCounterInc: true,
		},
		"deleted": {
			rootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: {},
				{3, 4, 5}: {},
			},
			root:          common.Hash{1, 2, 3},
			triesGaugeSet: 1,
			expectedRootToTrie: map[common.Hash]*trie.Trie{
				{3, 4, 5}: {},
			},
			deleteCounterInc: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			triesGauge := NewMockGauge(ctrl)
			triesGauge.EXPECT().Set(testCase.triesGaugeSet)

			deleteCounter := NewMockCounter(ctrl)
			if testCase.deleteCounterInc {
				deleteCounter.EXPECT().Inc()
			}

			tries := &Tries{
				rootToTrie:    testCase.rootToTrie,
				triesGauge:    triesGauge,
				deleteCounter: deleteCounter,
			}

			tries.delete(testCase.root)

			assert.Equal(t, testCase.expectedRootToTrie, tries.rootToTrie)
		})
	}
}
func Test_Tries_get(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	db := NewMockDatabase(ctrl)
	db.EXPECT().Get(gomock.Any()).Times(0)

	testCases := map[string]struct {
		tries *Tries
		root  common.Hash
		trie  *trie.Trie
	}{
		"found_in_map": {
			tries: &Tries{
				rootToTrie: map[common.Hash]*trie.Trie{
					{1, 2, 3}: trie.NewTrie(&node.Node{
						PartialKey:   []byte{1, 2, 3},
						StorageValue: []byte{1},
					}, db),
				},
			},
			root: common.Hash{1, 2, 3},
			trie: trie.NewTrie(&node.Node{
				PartialKey:   []byte{1, 2, 3},
				StorageValue: []byte{1},
			}, db),
		},
		"not_found_in_map": {
			// similar to not found in database
			tries: &Tries{
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

func Test_Tries_len(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tries  *Tries
		length int
	}{
		"empty_map": {
			tries: &Tries{
				rootToTrie: map[common.Hash]*trie.Trie{},
			},
		},
		"non_empty_map": {
			tries: &Tries{
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
