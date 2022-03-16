// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) chaindb.Database {
	testDatadirPath := t.TempDir()
	db, err := utils.SetupDatabase(testDatadirPath, true)
	require.NoError(t, err)
	return chaindb.NewTable(db, "trie")
}

type keyValue struct {
	key   []byte
	value []byte
}

func getDBKeyValuesA() []keyValue {
	return []keyValue{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x3}, value: []byte("f")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
		{key: []byte{0}, value: nil},
	}
}

func getDBKeyValuesB() []keyValue {
	return []keyValue{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x30}, value: []byte("f")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
	}
}

func getDBKeyValuesC() []keyValue {
	return []keyValue{
		{key: []byte("asdf"), value: []byte("asdf")},
		{key: []byte("ghjk"), value: []byte("ghjk")},
		{key: []byte("qwerty"), value: []byte("qwerty")},
		{key: []byte("uiopl"), value: []byte("uiopl")},
		{key: []byte("zxcv"), value: []byte("zxcv")},
		{key: []byte("bnm"), value: []byte("bnm")},
	}
}

func TestTrie_DatabaseStoreAndLoad(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues    []keyValue
		putNodesAdd  []int
		loadNodesAdd []int
	}{
		"first": {
			keyValues:    getDBKeyValuesA(),
			putNodesAdd:  []int{1, 1, 1, 2, 1, 2, 1, 1},
			loadNodesAdd: []int{0, 3},
		},
		// "second": {
		// 	keyValues: getDBKeyValuesB(),
		// 	metricsNodesAdd: []int{
		// 		// Put
		// 		1, 1, 2, 2, 1, 2, 1,
		// 		// Load
		// 		3,
		// 	},
		// },
		// "third": {
		// 	keyValues:       getDBKeyValuesC(),
		// 	metricsNodesAdd: []int{1, 2, 2, 2, 1, 1, 3},
		// },
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			// t.Parallel()
			ctrl := gomock.NewController(t)

			metrics := NewMockMetrics(ctrl)
			configMockMetricsPrinter(t, metrics)
			// var previousCall *gomock.Call
			// for _, n := range testCase.putNodesAdd {
			// 	call := metrics.EXPECT().NodesAdd(n)
			// 	if previousCall != nil {
			// 		call.After(previousCall)
			// 	}
			// 	previousCall = call
			// }

			trie := NewEmptyTrie(metrics)

			for _, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)
			}
			t.Log(countNodesRecursively(trie.root))
			t.Log(countNodesFromStats(trie.root))

			db := newTestDB(t)
			err := trie.Store(db)
			require.NoError(t, err)

			metrics = NewMockMetrics(ctrl)
			configMockMetricsPrinter(t, metrics)
			// var previousCall *gomock.Call
			// for _, n := range testCase.putNodesAdd {
			// 	call := metrics.EXPECT().NodesAdd(n)
			// 	if previousCall != nil {
			// 		call.After(previousCall)
			// 	}
			// 	previousCall = call
			// }

			res := NewEmptyTrie(metrics)
			err = res.Load(db, trie.MustHash())
			require.NoError(t, err)
			require.Equal(t, trie.root, res.root)
			t.Log(countNodesRecursively(res.root))
			t.Log(countNodesFromStats(res.root))

			for _, keyValue := range testCase.keyValues {
				val, err := GetFromDB(db, trie.MustHash(), keyValue.key)
				require.NoError(t, err)
				require.Equal(t, keyValue.value, val)
			}
		})
	}
}

func TestTrie_WriteDirty_Put(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues       []keyValue
		metricsNodesAdd []int
	}{
		"first": {
			keyValues:       getDBKeyValuesA(),
			metricsNodesAdd: []int{1, 1, 1, 2, 1, 4, 1, 1, 1, 4},
		},
		// "second": {
		// 	keyValues:       getDBKeyValuesB(),
		// 	metricsNodesAdd: []int{1, 1, 2, 2, 1, 4, 1, 1, 4},
		// },
		// "third": {
		// 	keyValues:       getDBKeyValuesC(),
		// 	metricsNodesAdd: []int{1, 2, 2, 2, 1, 1, 3},
		// },
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			metrics := NewMockMetrics(ctrl)
			configMockMetricsPrinter(t, metrics)
			// var previousCall *gomock.Call
			// for _, n := range testCase.metricsNodesAdd {
			// 	call := metrics.EXPECT().NodesAdd(n)
			// 	if previousCall != nil {
			// 		call.After(previousCall)
			// 	}
			// 	previousCall = call
			// }

			trie := NewEmptyTrie(metrics)
			db := newTestDB(t)

			for i, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)
				err := trie.WriteDirty(db)
				require.NoError(t, err)

				for j, kv := range testCase.keyValues {
					if j > i {
						break
					}

					val, err := GetFromDB(db, trie.MustHash(), kv.key)
					require.NoError(t, err)
					require.Equal(t, kv.value, val, fmt.Sprintf("key=%x", kv.key))
				}
			}
			t.Logf("trie size: %d", trie.root.(*node.Branch).Descendants)

			err := trie.Store(db)
			require.NoError(t, err)

			trie.Put([]byte("asdf"), []byte("notapenguin"))
			err = trie.WriteDirty(db)
			require.NoError(t, err)

			res := NewEmptyTrie(metrics)
			err = res.Load(db, trie.MustHash())
			require.NoError(t, err)
			require.Equal(t, trie.MustHash(), res.MustHash())

			for _, keyValue := range testCase.keyValues {
				val, err := GetFromDB(db, trie.MustHash(), keyValue.key)
				require.NoError(t, err)
				if bytes.Equal(keyValue.key, []byte("asdf")) {
					continue
				}
				require.Equal(t, keyValue.value, val)
			}

			val, err := GetFromDB(db, trie.MustHash(), []byte("asdf"))
			require.NoError(t, err)
			require.Equal(t, []byte("notapenguin"), val)
		})
	}
}

func TestTrie_WriteDirty_PutReplace(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues           []keyValue
		metricsNodesAddPut  []int
		metricsNodesAddLoad []int
	}{
		"first": {
			keyValues:           getDBKeyValuesA(),
			metricsNodesAddPut:  []int{1, 1, 1, 2, 1, 2, 1, 1},
			metricsNodesAddLoad: []int{2},
		},
		"second": {
			keyValues:           getDBKeyValuesB(),
			metricsNodesAddPut:  []int{1, 1, 2, 2, 1, 2, 1},
			metricsNodesAddLoad: []int{2},
		},
		"third": {
			keyValues:           getDBKeyValuesC(),
			metricsNodesAddPut:  []int{1, 2, 2, 2, 1, 1},
			metricsNodesAddLoad: []int{2},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			metrics := NewMockMetrics(ctrl)
			var previousCall *gomock.Call
			for _, n := range testCase.metricsNodesAddPut {
				call := metrics.EXPECT().NodesAdd(n)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			trie := NewEmptyTrie(metrics)

			db := newTestDB(t)

			for _, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)

				err := trie.WriteDirty(db)
				require.NoError(t, err)
			}

			for _, keyValue := range testCase.keyValues {
				// overwrite existing values
				trie.Put(keyValue.key, keyValue.key)

				err := trie.WriteDirty(db)
				require.NoError(t, err)
			}

			for _, n := range testCase.metricsNodesAddLoad {
				call := metrics.EXPECT().NodesAdd(n)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			res := NewEmptyTrie(metrics)
			err := res.Load(db, trie.MustHash())
			require.NoError(t, err)
			require.Equal(t, trie.MustHash(), res.MustHash())

			for _, keyValue := range testCase.keyValues {
				val, err := GetFromDB(db, trie.MustHash(), keyValue.key)
				require.NoError(t, err)
				require.Equal(t, keyValue.key, val)
			}
		})
	}
}

func TestTrie_WriteDirty_Delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues       []keyValue
		metricsNodesAdd []int
	}{
		"first": {
			keyValues: getDBKeyValuesA(),
			metricsNodesAdd: []int{
				1, 1, 1, 2, 1, 4, 1, 1, 0, 3,
				1, 1, 1, 2, 1, 4, 1, 1, 0, 3,
				1, 1, 1, 2, 1, 4, 1, 1, 0, 3,
				1, 1, 1, 2, 1, 4, 1, 1, -1, 3,
				1, 1, 1, 2, 1, 4, 1, 1, 0, 3,
				1, 1, 1, 2, 1, 4, 1, 1, 0, 3,
				1, 1, 1, 2, 1, 4, 1, 1, 0, 3,
				1, 1, 1, 2, 1, 4, 1, 1, 0, 3},
		},
		"second": {
			keyValues: getDBKeyValuesB(),
			metricsNodesAdd: []int{
				1, 1, 2, 2, 1, 4, 1, 0, 3, 1,
				1, 2, 2, 1, 4, 1, 0, 3, 1, 1,
				2, 2, 1, 4, 1, 0, 3, 1, 1, 2,
				2, 1, 4, 1, -1, 3, 1, 1, 2, 2,
				1, 4, 1, 0, 3, 1, 1, 2, 2, 1,
				4, 1, 0, 3, 1, 1, 2, 2, 1, 4,
				1, 0, 3},
		},
		"third": {
			keyValues: getDBKeyValuesC(),
			metricsNodesAdd: []int{
				1, 2, 2, 2, 1, 1, 0, 3, 1, 2,
				2, 2, 1, 1, 0, 3, 1, 2, 2, 2,
				1, 1, 0, 3, 1, 2, 2, 2, 1, 1,
				0, 3, 1, 2, 2, 2, 1, 1, 0, 3,
				1, 2, 2, 2, 1, 1, 0, 3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			metrics := NewMockMetrics(ctrl)

			var previousCall *gomock.Call
			for _, n := range testCase.metricsNodesAdd {
				call := metrics.EXPECT().NodesAdd(n)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			for _, curr := range testCase.keyValues {
				trie := NewEmptyTrie(metrics)

				for _, keyValue := range testCase.keyValues {
					trie.Put(keyValue.key, keyValue.value)
				}

				db := newTestDB(t)
				err := trie.Store(db)
				require.NoError(t, err)

				err = trie.DeleteFromDB(db, curr.key)
				require.NoError(t, err)

				res := NewEmptyTrie(metrics)
				err = res.Load(db, trie.MustHash())
				require.NoError(t, err)
				require.Equal(t, trie.MustHash(), res.MustHash())

				for _, keyValue := range testCase.keyValues {
					val, err := GetFromDB(db, trie.MustHash(), keyValue.key)
					require.NoError(t, err)

					if bytes.Equal(keyValue.key, curr.key) {
						require.Nil(t, val, fmt.Sprintf("key=%x", keyValue.key))
						continue
					}

					require.Equal(t, keyValue.value, val)
				}
			}
		})
	}
}

func TestTrie_WriteDirty_ClearPrefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues       []keyValue
		metricsNodesAdd []int
		metricsNodesSub []int
	}{
		"first": {
			keyValues:       getDBKeyValuesA(),
			metricsNodesAdd: []int{1, 1, 1, 2, 1, 2, 1, 1, 2},
			metricsNodesSub: []int{3},
		},
		"second": {
			keyValues:       getDBKeyValuesB(),
			metricsNodesAdd: []int{1, 1, 2, 2, 1, 2, 1, 2},
			metricsNodesSub: []int{4},
		},
		"third": {
			keyValues:       getDBKeyValuesC(),
			metricsNodesAdd: []int{1, 2, 2, 2, 1, 1, 2},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			metrics := NewMockMetrics(ctrl)
			var previousCall *gomock.Call
			for _, n := range testCase.metricsNodesAdd {
				call := metrics.EXPECT().NodesAdd(n)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			previousCall = nil
			for _, n := range testCase.metricsNodesSub {
				call := metrics.EXPECT().NodesAdd(-n)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			trie := NewEmptyTrie(metrics)

			for _, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)
			}

			db := newTestDB(t)
			err := trie.Store(db)
			require.NoError(t, err)

			err = trie.ClearPrefixFromDB(db, []byte{0x01, 0x35})
			require.NoError(t, err)

			res := NewEmptyTrie(metrics)
			err = res.Load(db, trie.MustHash())
			require.NoError(t, err)

			require.Equal(t, trie.MustHash(), res.MustHash())
		})
	}
}

func TestTrie_GetFromDB(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues       []keyValue
		metricsNodesAdd []int
	}{
		"first": {
			keyValues:       getDBKeyValuesA(),
			metricsNodesAdd: []int{1, 1, 1, 2, 1, 2, 1, 1},
		},
		"second": {
			keyValues:       getDBKeyValuesB(),
			metricsNodesAdd: []int{1, 1, 2, 2, 1, 2, 1},
		},
		"third": {
			keyValues:       getDBKeyValuesC(),
			metricsNodesAdd: []int{1, 2, 2, 2, 1, 1},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			metrics := NewMockMetrics(ctrl)
			var previousCall *gomock.Call
			for _, n := range testCase.metricsNodesAdd {
				call := metrics.EXPECT().NodesAdd(n)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			trie := NewEmptyTrie(metrics)

			for _, keyValue := range testCase.keyValues {
				trie.Put(keyValue.key, keyValue.value)
			}

			db := newTestDB(t)
			err := trie.Store(db)
			require.NoError(t, err)

			root := trie.MustHash()

			for _, keyValue := range testCase.keyValues {
				val, err := GetFromDB(db, root, keyValue.key)
				require.NoError(t, err)
				require.Equal(t, keyValue.value, val)
			}
		})
	}
}
