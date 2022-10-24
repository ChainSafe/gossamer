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
		deletedNodeHashes:  make(map[common.Hash]struct{}),
		insertedNodeHashes: make(map[common.Hash]struct{}),
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
		"set in empty deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{},
			},
			nodeHash: common.Hash{1},
			expectedDeltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
		},
		"set in non empty deltas": {
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
		"override in deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			nodeHash: common.Hash{1},
			expectedDeltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
		},
		"remove pending insertion": {
			deltas: Deltas{
				deletedNodeHashes:  map[common.Hash]struct{}{},
				insertedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			nodeHash: common.Hash{1},
			expectedDeltas: Deltas{
				deletedNodeHashes:  map[common.Hash]struct{}{},
				insertedNodeHashes: map[common.Hash]struct{}{},
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

func Test_Deltas_RecordInserted(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deltas         Deltas
		nodeHash       common.Hash
		expectedDeltas Deltas
	}{
		"set in empty deltas": {
			deltas: Deltas{
				insertedNodeHashes: map[common.Hash]struct{}{},
			},
			nodeHash: common.Hash{1},
			expectedDeltas: Deltas{
				insertedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
		},
		"set in non empty deltas": {
			deltas: Deltas{
				insertedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			nodeHash: common.Hash{2},
			expectedDeltas: Deltas{
				insertedNodeHashes: map[common.Hash]struct{}{
					{1}: {}, {2}: {},
				},
			},
		},
		"override in deltas": {
			deltas: Deltas{
				insertedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
			nodeHash: common.Hash{1},
			expectedDeltas: Deltas{
				insertedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			},
		},
		"remove pending deletion": {
			deltas: Deltas{
				deletedNodeHashes:  map[common.Hash]struct{}{{1}: {}},
				insertedNodeHashes: map[common.Hash]struct{}{},
			},
			nodeHash: common.Hash{1},
			expectedDeltas: Deltas{
				deletedNodeHashes:  map[common.Hash]struct{}{},
				insertedNodeHashes: map[common.Hash]struct{}{},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.deltas.RecordInserted(testCase.nodeHash)
			assert.Equal(t, testCase.expectedDeltas, testCase.deltas)
		})
	}
}

func Test_Deltas_Get(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deltas             Deltas
		deletedNodeHashes  map[common.Hash]struct{}
		insertedNodeHashes map[common.Hash]struct{}
	}{
		"empty deltas": {},
		"non empty deltas": {
			deltas: Deltas{
				insertedNodeHashes: map[common.Hash]struct{}{{1}: {}},
				deletedNodeHashes:  map[common.Hash]struct{}{{2}: {}},
			},
			insertedNodeHashes: map[common.Hash]struct{}{{1}: {}},
			deletedNodeHashes:  map[common.Hash]struct{}{{2}: {}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			insertedNodeHashes, deletedNodeHashes := testCase.deltas.Get()
			assert.Equal(t, testCase.deletedNodeHashes, deletedNodeHashes)
			assert.Equal(t, testCase.insertedNodeHashes, insertedNodeHashes)
		})
	}
}

func Test_Deltas_MergeWith(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		deltas         Deltas
		deltasArg      Getter
		mergeDeleted   bool
		expectedDeltas Deltas
	}{
		"merge empty deltas": {
			deltas: Deltas{
				deletedNodeHashes:  map[common.Hash]struct{}{{1}: {}},
				insertedNodeHashes: map[common.Hash]struct{}{{2}: {}},
			},
			deltasArg:    &Deltas{},
			mergeDeleted: true,
			expectedDeltas: Deltas{
				deletedNodeHashes:  map[common.Hash]struct{}{{1}: {}},
				insertedNodeHashes: map[common.Hash]struct{}{{2}: {}},
			},
		},
		"merge deltas": {
			deltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{
					{1}: {},
					{2}: {},
				},
				insertedNodeHashes: map[common.Hash]struct{}{
					{5}: {},
					{6}: {},
				},
			},
			deltasArg: &Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{
					{1}: {}, // already set as deleted
					{5}: {}, // remove from inserted
					{3}: {}, // add as deleted
				},
				insertedNodeHashes: map[common.Hash]struct{}{
					{6}: {}, // already set as deleted
					{2}: {}, // remove from deleted
					{7}: {}, // add as inserted
				},
			},
			mergeDeleted: true,
			expectedDeltas: Deltas{
				deletedNodeHashes: map[common.Hash]struct{}{
					{1}: {},
					{3}: {},
				},
				insertedNodeHashes: map[common.Hash]struct{}{
					{6}: {},
					{7}: {},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.deltas.MergeWith(testCase.deltasArg, testCase.mergeDeleted)
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
		"nil deltas": {},
		"empty deltas": {
			deltasOriginal: &Deltas{},
			deltasCopy:     &Deltas{},
		},
		"filled deltas": {
			deltasOriginal: &Deltas{
				deletedNodeHashes:  map[common.Hash]struct{}{{1}: {}},
				insertedNodeHashes: map[common.Hash]struct{}{{2}: {}},
			},
			deltasCopy: &Deltas{
				deletedNodeHashes:  map[common.Hash]struct{}{{1}: {}},
				insertedNodeHashes: map[common.Hash]struct{}{{2}: {}},
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
