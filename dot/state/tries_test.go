// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewTries(t *testing.T) {
	t.Parallel()

	db := &chaindb.BadgerDB{}
	tr := trie.NewEmptyTrie()

	rootToTrie, err := NewTries(db, tr)
	require.NoError(t, err)

	expectedTries := &Tries{
		rootToTrie: map[common.Hash]*trie.Trie{
			tr.MustHash(): tr,
		},
		db:            db,
		triesGauge:    triesGauge,
		setCounter:    setCounter,
		deleteCounter: deleteCounter,
	}

	assert.Equal(t, expectedTries, rootToTrie)
}

//go:generate mockgen -destination=mock_gauge_test.go -package $GOPACKAGE github.com/prometheus/client_golang/prometheus Gauge
//go:generate mockgen -destination=mock_counter_test.go -package $GOPACKAGE github.com/prometheus/client_golang/prometheus Counter
//go:generate mockgen -destination=mock_database_test.go -package $GOPACKAGE github.com/ChainSafe/chaindb Database

func Test_tries_getValue(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	type dbGetCall struct {
		nodeHash []byte
		node     node.Node
		err      error
	}

	testCases := map[string]struct {
		rootToTrie map[common.Hash]*trie.Trie
		trieRoot   common.Hash
		key        []byte
		dbGetCalls []dbGetCall
		value      []byte
		errWrapped error
		errMessage string
	}{
		"trie found in memory and key not found": {
			rootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: trie.NewTrie(&node.Leaf{
					Key:   []byte{1, 2},
					Value: []byte{3, 4},
				}),
			},
			trieRoot: common.Hash{1, 2, 3},
			key:      []byte{0x23},
		},
		"trie found in memory and key found": {
			rootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: trie.NewTrie(&node.Leaf{
					Key:   []byte{1, 2},
					Value: []byte{3, 4},
				}),
			},
			trieRoot: common.Hash{1, 2, 3},
			key:      []byte{0x12},
			value:    []byte{3, 4},
		},
		"trie not found in memory and empty hash in database": {
			rootToTrie: map[common.Hash]*trie.Trie{},
			trieRoot:   trie.EmptyHash,
			key:        []byte{},
		},
		"trie not found in memory and not found in database": {
			rootToTrie: map[common.Hash]*trie.Trie{},
			trieRoot:   trie.EmptyHash,
			key:        []byte{0x12},
			dbGetCalls: []dbGetCall{
				{nodeHash: trie.EmptyHash[:], err: errTest},
			},
			errWrapped: errTest,
			errMessage: "cannot get value from database: " +
				"cannot find root hash key " +
				"0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314: " +
				"test error",
		},
		// "trie not found in memory and found in database": {
		// 	rootToTrie: map[common.Hash]*trie.Trie{},
		// 	trieRoot:   trie.EmptyHash,
		// 	key:        []byte{0x12},
		// 	dbGetCalls: []dbGetCall{
		// 		{
		// 			nodeHash: trie.EmptyHash[:],
		// 			node: &node.Leaf{
		// 				Key:   []byte{1, 2},
		// 				Value: []byte{3, 4},
		// 			},
		// 		},
		// 	},
		// 	value: []byte{3, 4},
		// },
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			db := NewMockDatabase(ctrl)
			var previousCall *gomock.Call
			for _, call := range testCase.dbGetCalls {
				var encodedNode []byte
				if call.node != nil {
					buffer := bytes.NewBuffer(nil)
					err := call.node.Encode(buffer)
					require.NoError(t, err)
					encodedNode = buffer.Bytes()
				}

				call := db.EXPECT().Get(call.nodeHash).
					Return(encodedNode, call.err)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			tries := Tries{
				rootToTrie: testCase.rootToTrie,
				db:         db,
			}

			value, err := tries.getValue(testCase.trieRoot, testCase.key)

			assert.Equal(t, testCase.value, value)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
		})
	}
}

func Test_tries_softSetTrieInMemory(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		rootToTrie         map[common.Hash]*trie.Trie
		root               common.Hash
		trie               *trie.Trie
		triesGaugeInc      bool
		expectedRootToTrie map[common.Hash]*trie.Trie
	}{
		"set new in map": {
			rootToTrie:    map[common.Hash]*trie.Trie{},
			root:          common.Hash{1, 2, 3},
			trie:          trie.NewEmptyTrie(),
			triesGaugeInc: true,
			expectedRootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: trie.NewEmptyTrie(),
			},
		},
		"do not override in map": {
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

			tries.softSetTrieInMemory(testCase.root, testCase.trie)

			assert.Equal(t, testCase.expectedRootToTrie, tries.rootToTrie)
		})
	}
}

func Test_tries_deleteTrieFromMemory(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		rootToTrie         map[common.Hash]*trie.Trie
		root               common.Hash
		deleteCounterInc   bool
		expectedRootToTrie map[common.Hash]*trie.Trie
		triesGaugeSet      float64
	}{
		"not found": {
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

			tries.deleteTrieFromMemory(testCase.root)

			assert.Equal(t, testCase.expectedRootToTrie, tries.rootToTrie)
		})
	}
}
func Test_tries_getTrie(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")
	exampleHashHex := "0x847c95428d9ccfcea72715334874991183b8e8c48088ea4d9f578294c976f46f"
	exampleHash, err := common.HexToHash(exampleHashHex)
	require.NoError(t, err)

	type dbGetCall struct {
		nodeHash []byte
		node     node.Node
		err      error
	}

	testCases := map[string]struct {
		rootToTrie         map[common.Hash]*trie.Trie
		root               common.Hash
		dbGetCalls         []dbGetCall
		trie               *trie.Trie
		errWrapped         error
		errMessage         string
		expectedRootToTrie map[common.Hash]*trie.Trie
	}{
		"empty hash": {
			rootToTrie:         map[common.Hash]*trie.Trie{},
			root:               trie.EmptyHash,
			trie:               trie.NewEmptyTrie(),
			expectedRootToTrie: map[common.Hash]*trie.Trie{},
		},
		"found in map": {
			rootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: trie.NewTrie(&node.Leaf{
					Key: []byte{1, 2, 3},
				}),
			},
			root: common.Hash{1, 2, 3},
			trie: trie.NewTrie(&node.Leaf{
				Key: []byte{1, 2, 3},
			}),
			expectedRootToTrie: map[common.Hash]*trie.Trie{
				{1, 2, 3}: trie.NewTrie(&node.Leaf{
					Key: []byte{1, 2, 3},
				}),
			},
		},
		"not found in map and database get error": {
			// similar to not found in database
			rootToTrie: map[common.Hash]*trie.Trie{},
			root:       exampleHash,
			dbGetCalls: []dbGetCall{
				{nodeHash: exampleHash[:], err: errTest},
			},
			errWrapped: errTest,
			errMessage: "cannot load root from database: " +
				"failed to find root key " +
				exampleHashHex + ": " +
				"test error",
			expectedRootToTrie: map[common.Hash]*trie.Trie{},
		},
		"not found in map and found in database": {
			rootToTrie: map[common.Hash]*trie.Trie{},
			root:       exampleHash,
			dbGetCalls: []dbGetCall{
				{
					nodeHash: exampleHash[:],
					node: &node.Leaf{
						Key: []byte{1, 2, 3},
					},
				},
			},
			trie: trie.NewTrie(&node.Leaf{
				Key:      []byte{1, 2, 3},
				Encoding: []byte{0x43, 0x01, 0x23, 0x00},
				HashDigest: []byte{
					0x84, 0x7c, 0x95, 0x42, 0x8d, 0x9c, 0xcf, 0xce,
					0xa7, 0x27, 0x15, 0x33, 0x48, 0x74, 0x99, 0x11,
					0x83, 0xb8, 0xe8, 0xc4, 0x80, 0x88, 0xea, 0x4d,
					0x9f, 0x57, 0x82, 0x94, 0xc9, 0x76, 0xf4, 0x6f},
			}),
			expectedRootToTrie: map[common.Hash]*trie.Trie{
				exampleHash: trie.NewTrie(&node.Leaf{
					Key:      []byte{1, 2, 3},
					Encoding: []byte{0x43, 0x01, 0x23, 0x00},
					HashDigest: []byte{
						0x84, 0x7c, 0x95, 0x42, 0x8d, 0x9c, 0xcf, 0xce,
						0xa7, 0x27, 0x15, 0x33, 0x48, 0x74, 0x99, 0x11,
						0x83, 0xb8, 0xe8, 0xc4, 0x80, 0x88, 0xea, 0x4d,
						0x9f, 0x57, 0x82, 0x94, 0xc9, 0x76, 0xf4, 0x6f},
				}),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			db := NewMockDatabase(ctrl)
			var previousCall *gomock.Call
			for _, call := range testCase.dbGetCalls {
				var encodedNode []byte
				if call.node != nil {
					buffer := bytes.NewBuffer(nil)
					err := call.node.Encode(buffer)
					require.NoError(t, err)
					encodedNode = buffer.Bytes()
				}

				call := db.EXPECT().Get(call.nodeHash).
					Return(encodedNode, call.err)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			tries := Tries{
				rootToTrie: testCase.rootToTrie,
				db:         db,
			}

			trieFound, err := tries.getTrie(testCase.root)

			assert.Equal(t, testCase.trie, trieFound)
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.expectedRootToTrie, tries.rootToTrie)
		})
	}

	t.Run("root hash mismatch from database panics", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		rootHash := common.Hash{1}

		rootNode := &node.Leaf{
			Key: []byte{1, 2, 3},
		}

		buffer := bytes.NewBuffer(nil)
		err := rootNode.Encode(buffer)
		require.NoError(t, err)
		encodedNode := buffer.Bytes()

		db := NewMockDatabase(ctrl)
		db.EXPECT().Get(rootHash[:]).Return(encodedNode, nil)

		tries := Tries{
			rootToTrie: map[common.Hash]*trie.Trie{},
			db:         db,
		}

		expectedPanicMessage := "trie does not have expected root, " +
			"expected " +
			"0x0100000000000000000000000000000000000000000000000000000000000000 " +
			"but got " +
			"0x847c95428d9ccfcea72715334874991183b8e8c48088ea4d9f578294c976f46f"
		assert.PanicsWithValue(t, expectedPanicMessage, func() {
			_, _ = tries.getTrie(rootHash)
		})
	})
}

func Test_tries_triesInMemory(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tries  *Tries
		length int
	}{
		"empty map": {
			tries: &Tries{
				rootToTrie: map[common.Hash]*trie.Trie{},
			},
		},
		"non empty map": {
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

			length := testCase.tries.triesInMemory()

			assert.Equal(t, testCase.length, length)
		})
	}
}
