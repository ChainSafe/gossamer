// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	t.Parallel()

	deltas := New()

	expectedDeltas := &Deltas{
		deletedNodeHashes: make(map[common.Hash]struct{}),
	}
	assert.Equal(t, expectedDeltas, deltas)
}

func Test_Deltas_RecordDeleted(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deltas         Deltas
		nodeHash       common.Hash
		expectedDeltas Deltas
	}{
		"set_in_empty_deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{},
			},
			nodeHash: common.Hash{1},
			expectedDeltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
		},
		"set_in_non_empty_deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			nodeHash: common.Hash{2},
			expectedDeltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{
					{1}: {}, {2}: {},
				},
			},
		},
		"override_in_deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			nodeHash: common.Hash{1},
			expectedDeltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.deltas.RecordDeleted(testCase.nodeHash)
			assert.Equal(t, testCase.expectedDeltas, testCase.deltas)
		})
	}
}

func Test_Deltas_Deleted(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deltas     Deltas
		nodeHashes map[common.Hash]struct{}
	}{
		"empty_deltas": {},
		"non_empty_deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			nodeHashes: map[common.Hash]struct{}{{1}: {}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nodeHashes := testCase.deltas.Deleted()
			assert.Equal(t, testCase.nodeHashes, nodeHashes)
		})
	}
}

func Test_Deltas_MergeWith(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deltas         Deltas
		deltasArg      Getter
		expectedDeltas Deltas
	}{
		"merge_empty_deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			deltasArg: &Deltas{},
			expectedDeltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
		},
		"merge_deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			deltasArg: &Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{
					{1}: {}, {2}: {},
				},
			},
			expectedDeltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{
					{1}: {}, {2}: {},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.deltas.MergeWith(testCase.deltasArg)
			assert.Equal(t, testCase.expectedDeltas, testCase.deltas)
		})
	}
}

func Test_Deltas_DeepCopy(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deltasOriginal *Deltas
		deltasCopy     *Deltas
	}{
		"nil_deltas": {},
		"empty_deltas": {
			deltasOriginal: &Deltas{},
			deltasCopy:     &Deltas{},
		},
		"filled_deltas": {
			deltasOriginal: &Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			deltasCopy: &Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deepCopy := testCase.deltasOriginal.DeepCopy()

			assert.Equal(t, testCase.deltasCopy, deepCopy)
			assertPointersNotEqual(t, testCase.deltasOriginal, deepCopy)
			if testCase.deltasOriginal != nil {
				assertPointersNotEqual(t, testCase.deltasOriginal.deletedNodeHashes, deepCopy.deletedNodeHashes)
			}
		})
	}
}
