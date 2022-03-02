// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	metricsnoop "github.com/ChainSafe/gossamer/internal/trie/metrics/noop"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	put = iota
	del
	clearPrefix
	get
	getLeaf
)

// writeFailedData writes key value pairs as hexadecimal to the path
// given in tab separated values format (TSV).
func writeFailedData(t *testing.T, kv map[string][]byte, path string) {
	t.Logf("Writing failed test data (%d key values) to %s", len(kv), path)

	lines := make([]string, 0, len(kv))
	for keyString, value := range kv {
		key := []byte(keyString)
		line := fmt.Sprintf("%x\t%x", key, value)
		lines = append(lines, line)
	}

	path, err := filepath.Abs(path)
	require.NoError(t, err)

	err = os.RemoveAll(path)
	require.NoError(t, err)

	data := []byte(strings.Join(lines, "\n"))

	err = os.WriteFile(path, data, os.ModePerm)
	require.NoError(t, err)
}

func buildSmallTrie(metrics *MockMetrics) *Trie {
	metrics.EXPECT().NodesAdd(uint32(1))
	metrics.EXPECT().NodesAdd(uint32(1))
	metrics.EXPECT().NodesAdd(uint32(2))
	metrics.EXPECT().NodesAdd(uint32(2))
	metrics.EXPECT().NodesAdd(uint32(1))

	trie := NewEmptyTrie(metrics)
	trie.Put([]byte{0x01, 0x35}, []byte("pen"))
	trie.Put([]byte{0x01, 0x35, 0x79}, []byte("penguin"))
	trie.Put([]byte{0xf2}, []byte("feather"))
	trie.Put([]byte{0x09, 0xd3}, []byte("noot"))
	trie.Put([]byte{}, []byte("floof"))
	trie.Put([]byte{0x01, 0x35, 0x07}, []byte("odd"))
	return trie
}

func runTests(t *testing.T, trie *Trie, tests []Test) {
	for _, test := range tests {
		switch test.op {
		case put:
			trie.Put(test.key, test.value)
		case get:
			val := trie.Get(test.key)
			assert.Equal(t, test.value, val)
		case del:
			trie.Delete(test.key)
		case getLeaf:
			value := trie.Get(test.key)
			assert.Equal(t, test.value, value)
		}
	}
}

func TestPutAndGetBranch(t *testing.T) {
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)
	call := metrics.EXPECT().NodesAdd(uint32(1))
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(2)).After(call)
	metrics.EXPECT().NodesAdd(uint32(2)).After(call)

	trie := NewEmptyTrie(metrics)

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: put},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: put},
		{key: []byte{0x07}, value: []byte("ramen"), op: put},
		{key: []byte{0xf2}, value: []byte("pho"), op: put},
		{key: []byte("noot"), value: nil, op: get},
		{key: []byte{0}, value: nil, op: get},
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: get},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: get},
		{key: []byte{0x07}, value: []byte("ramen"), op: get},
		{key: []byte{0xf2}, value: []byte("pho"), op: get},
	}

	runTests(t, trie, tests)
}

func TestPutAndGetOddKeyLengths(t *testing.T) {
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)
	call := metrics.EXPECT().NodesAdd(uint32(1))
	call = metrics.EXPECT().NodesAdd(uint32(2)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(2)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	metrics.EXPECT().NodesAdd(uint32(2)).After(call)

	trie := NewEmptyTrie(metrics)

	tests := []Test{
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: put},
		{key: []byte{0x49, 0x29}, value: []byte("nootagain"), op: put},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: put},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: put},
		{key: []byte{0x4f, 0xbc}, value: []byte("stuffagain"), op: put},
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: get},
		{key: []byte{0x49, 0x29}, value: []byte("nootagain"), op: get},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: get},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: get},
		{key: []byte{0x4f, 0xbc}, value: []byte("stuffagain"), op: get},
	}

	runTests(t, trie, tests)
}

func Test_Trie_PutAndGet(t *testing.T) {
	generator := newGenerator()
	const kvSize = 10000
	kv := generateKeyValues(t, generator, kvSize)

	testPutAndGetKeyValues(t, kv)

	if t.Failed() {
		failedDataPath := fmt.Sprintf("./trie_putandget_failed_test_data_%d.tsv", time.Now().Unix())
		writeFailedData(t, kv, failedDataPath)
	}
}

func testPutAndGetKeyValues(t *testing.T, kv map[string][]byte) {
	t.Helper()

	metrics := metricsnoop.New() // we don't care here

	trie := NewEmptyTrie(metrics)

	for keyString, value := range kv {
		key := []byte(keyString)

		trie.Put(key, value)

		retrievedValue := trie.Get(key)
		if !assert.Equal(t, value, retrievedValue) {
			return
		}
	}
}

// Test_Trie_PutAndGet_FailedData tests random data that failed in Test_Trie_PutAndGet.
// It checks every file starting with trie_putandget_failed_test_data_ and
// removes them if the test passes.
func Test_Trie_PutAndGet_FailedData(t *testing.T) {
	var failedTestDataPaths []string
	dirEntries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		path := dirEntry.Name()
		const targetPrefix = "trie_putandget_failed_test_data_"
		if strings.HasPrefix(path, targetPrefix) {
			failedTestDataPaths = append(failedTestDataPaths, path)
		}
	}

	for _, path := range failedTestDataPaths {
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		kv := make(map[string][]byte)

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.Split(line, "\t")
			key, err := hex.DecodeString(parts[0])
			require.NoError(t, err)

			value, err := hex.DecodeString(parts[1])
			require.NoError(t, err)

			kv[string(key)] = value
		}

		testPutAndGetKeyValues(t, kv)

		if !t.Failed() {
			err = os.RemoveAll(path)
			require.NoError(t, err)
		}
	}
}

func TestGetPartialKey(t *testing.T) {
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)
	call := metrics.EXPECT().NodesAdd(uint32(1))
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	call = metrics.EXPECT().NodesSub(uint32(1)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	metrics.EXPECT().NodesAdd(uint32(2)).After(call)

	trie := NewEmptyTrie(metrics)

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: put},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: put},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: put},
		{key: []byte{}, value: []byte("floof"), op: put},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: getLeaf},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: del},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: getLeaf},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: getLeaf},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: put},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: getLeaf},
		{key: []byte{0xf2}, value: []byte("pen"), op: put},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: put},
		{key: []byte{}, value: []byte("floof"), op: get},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: getLeaf},
		{key: []byte{0xf2}, value: []byte("pen"), op: getLeaf},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: getLeaf},
	}

	runTests(t, trie, tests)
}

func TestDeleteSmall(t *testing.T) {
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)

	trie := buildSmallTrie(metrics)

	var totalNodesDelta int
	metrics.EXPECT().NodesAdd(gomock.Any()).Do(func(n uint32) {
		totalNodesDelta += int(n)
	}).AnyTimes()
	metrics.EXPECT().NodesSub(gomock.Any()).Do(func(n uint32) {
		totalNodesDelta -= int(n)
	}).AnyTimes()

	tests := []Test{
		{key: []byte{}, value: []byte("floof"), op: del},
		{key: []byte{}, value: nil, op: get},
		{key: []byte{}, value: []byte("floof"), op: put},

		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: del},
		{key: []byte{0x09, 0xd3}, value: nil, op: get},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: get},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: get},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: put},

		{key: []byte{0xf2}, value: []byte("feather"), op: del},
		{key: []byte{0xf2}, value: nil, op: get},
		{key: []byte{0xf2}, value: []byte("feather"), op: put},

		{key: []byte{}, value: []byte("floof"), op: del},
		{key: []byte{0xf2}, value: []byte("feather"), op: del},
		{key: []byte{}, value: nil, op: get},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: get},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: get},
		{key: []byte{}, value: []byte("floof"), op: put},
		{key: []byte{0xf2}, value: []byte("feather"), op: put},

		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: del},
		{key: []byte{0x01, 0x35, 0x79}, value: nil, op: get},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: get},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: put},

		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: del},
		{key: []byte{0x01, 0x35}, value: nil, op: get},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: get},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: put},

		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: del},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: get},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: get},
	}

	runTests(t, trie, tests)

	const expectedTotalNodesDelta = -1
	assert.Equal(t, expectedTotalNodesDelta, totalNodesDelta)
}

func TestDeleteCombineBranch(t *testing.T) {
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)

	trie := buildSmallTrie(metrics)

	call := metrics.EXPECT().NodesAdd(uint32(1))
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	metrics.EXPECT().NodesSub(uint32(2)).After(call)

	tests := []Test{
		{key: []byte{0x01, 0x35, 0x46}, value: []byte("raccoon"), op: put},
		{key: []byte{0x01, 0x35, 0x46, 0x77}, value: []byte("rat"), op: put},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: del},
		{key: []byte{0x09, 0xd3}, value: nil, op: get},
	}

	runTests(t, trie, tests)
}

func TestDeleteFromBranch(t *testing.T) {
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)

	var totalNodes uint32
	metrics.EXPECT().NodesAdd(gomock.Any()).Do(func(n uint32) {
		totalNodes += n
	}).AnyTimes()
	metrics.EXPECT().NodesSub(gomock.Any()).Do(func(n uint32) {
		totalNodes -= n
	}).AnyTimes()

	trie := NewEmptyTrie(metrics)

	tests := []Test{
		{key: []byte{0x06, 0x15, 0xfc}, value: []byte("noot"), op: put},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: put},
		{key: []byte{0x06, 0xaf, 0xb1}, value: []byte("odd"), op: put},
		{key: []byte{0x06, 0xa3, 0xff}, value: []byte("stuff"), op: put},
		{key: []byte{0x43, 0x21}, value: []byte("stuffagain"), op: put},
		{key: []byte{0x06, 0x15, 0xfc}, value: []byte("noot"), op: get},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: get},
		{key: []byte{0x06, 0x15, 0xfc}, value: []byte("noot"), op: del},
		{key: []byte{0x06, 0x15, 0xfc}, value: nil, op: get},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: get},
		{key: []byte{0x06, 0xaf, 0xb1}, value: []byte("odd"), op: get},
		{key: []byte{0x06, 0xaf, 0xb1}, value: []byte("odd"), op: del},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: get},
		{key: []byte{0x06, 0xa3, 0xff}, value: []byte("stuff"), op: get},
		{key: []byte{0x06, 0xa3, 0xff}, value: []byte("stuff"), op: del},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: get},
	}

	runTests(t, trie, tests)

	const expectedTotalNodes uint32 = 3
	assert.Equal(t, expectedTotalNodes, totalNodes)
}

func TestDeleteOddKeyLengths(t *testing.T) {
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)
	var totalNodes uint32
	metrics.EXPECT().NodesAdd(gomock.Any()).Do(func(n uint32) {
		totalNodes += n
	}).AnyTimes()
	metrics.EXPECT().NodesSub(gomock.Any()).Do(func(n uint32) {
		totalNodes -= n
	}).AnyTimes()

	trie := NewEmptyTrie(metrics)

	tests := []Test{
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: put},
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: get},
		{key: []byte{0x49, 0x29}, value: []byte("nootagain"), op: put},
		{key: []byte{0x49, 0x29}, value: []byte("nootagain"), op: get},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: put},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: get},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: put},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: get},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: del},
		{key: []byte{0x43, 0x0c}, value: nil, op: get},
		{key: []byte{0xf4, 0xbc}, value: []byte("spaghetti"), op: put},
		{key: []byte{0xf4, 0xbc}, value: []byte("spaghetti"), op: get},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: get},
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: get},
	}

	runTests(t, trie, tests)

	const expectedTotalNodes uint32 = 6
	assert.Equal(t, expectedTotalNodes, totalNodes)
}

func TestTrieDiff(t *testing.T) {
	ctrl := gomock.NewController(t)

	cfg := &chaindb.Config{
		DataDir: t.TempDir(),
	}

	db, err := chaindb.NewBadgerDB(cfg)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = db.Close()
		require.NoError(t, err)
	})

	storageDB := chaindb.NewTable(db, "storage")
	t.Cleanup(func() {
		err = storageDB.Close()
		require.NoError(t, err)
	})

	metrics := NewMockMetrics(ctrl)
	call := metrics.EXPECT().NodesAdd(uint32(1))
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(2)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(1)).After(call)
	metrics.EXPECT().NodesAdd(uint32(1)).After(call)

	trie := NewEmptyTrie(metrics)

	keyValues := []keyValue{
		{key: []byte("testKey"), value: []byte("testKey")},
		{key: []byte("testKey1"), value: []byte("testKey1")},
		{key: []byte("testKey2"), value: []byte("testKey2")},
	}

	for _, keyValue := range keyValues {
		trie.Put(keyValue.key, keyValue.value)
	}

	newTrie := trie.Snapshot()
	err = trie.Store(storageDB)
	require.NoError(t, err)

	keyValues = []keyValue{
		{key: []byte("testKey"), value: []byte("newTestKey2")},
		{key: []byte("testKey2"), value: []byte("newKey")},
		{key: []byte("testKey3"), value: []byte("testKey3")},
		{key: []byte("testKey4"), value: []byte("testKey2")},
		{key: []byte("testKey5"), value: []byte("testKey5")},
	}

	for _, keyValue := range keyValues {
		newTrie.Put(keyValue.key, keyValue.value)
	}
	deletedKeys := newTrie.deletedKeys
	require.Len(t, deletedKeys, 3)

	err = newTrie.WriteDirty(storageDB)
	require.NoError(t, err)

	for key := range deletedKeys {
		err = storageDB.Del(key.ToBytes())
		require.NoError(t, err)
	}

	metrics.EXPECT().NodesAdd(uint32(1)).After(call)

	dbTrie := NewEmptyTrie(metrics)
	err = dbTrie.Load(storageDB, common.BytesToHash(newTrie.root.GetHash()))
	require.NoError(t, err)
}

func TestDelete(t *testing.T) {
	metrics := metricsnoop.New() // random values so cannot predict them

	trie := NewEmptyTrie(metrics)

	generator := newGenerator()
	const kvSize = 100
	kv := generateKeyValues(t, generator, kvSize)

	for keyString, value := range kv {
		key := []byte(keyString)
		trie.Put(key, value)
	}

	dcTrie := trie.DeepCopy()

	// Take Snapshot of the trie.
	ssTrie := trie.Snapshot()

	// Get the Trie root hash for all the 3 tries.
	tHash, err := trie.Hash()
	require.NoError(t, err)

	dcTrieHash, err := dcTrie.Hash()
	require.NoError(t, err)

	ssTrieHash, err := ssTrie.Hash()
	require.NoError(t, err)

	// Root hash for all the 3 tries should be equal.
	require.Equal(t, tHash, dcTrieHash)
	require.Equal(t, dcTrieHash, ssTrieHash)

	for keyString, value := range kv {
		key := []byte(keyString)
		switch generator.Int31n(2) {
		case 0:
			ssTrie.Delete(key)
			retrievedValue := ssTrie.Get(key)
			assert.Nil(t, retrievedValue, "for key %x", key)
		case 1:
			retrievedValue := ssTrie.Get(key)
			assert.Equal(t, value, retrievedValue, "for key %x", key)
		}
	}

	// Get the updated root hash of all tries.
	tHash, err = trie.Hash()
	require.NoError(t, err)

	dcTrieHash, err = dcTrie.Hash()
	require.NoError(t, err)

	ssTrieHash, err = ssTrie.Hash()
	require.NoError(t, err)

	// Only the current trie should have a different root hash since it is updated.
	require.NotEqual(t, ssTrie, dcTrieHash)
	require.NotEqual(t, ssTrie, tHash)
	require.Equal(t, dcTrieHash, tHash)
}

func TestClearPrefix(t *testing.T) {
	t.Parallel()

	trieKeyValues := []keyValue{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi")},
		{key: []byte{0x01, 0x35, 0x79, 0xab}, value: []byte("spaghetti")},
		{key: []byte{0x01, 0x35, 0x79, 0xab, 0x9}, value: []byte("gnocchi")},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen")},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles")},
		{key: []byte{0xf2}, value: []byte("pho")},
		{key: []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0x11}, value: []byte("asd")},
		{key: []byte{0xff, 0xee, 0xdd, 0xcc, 0xaa, 0x11}, value: []byte("fgh")},
	}

	testCases := map[string]struct {
		prefixToClear   []byte
		metricsNodesSub uint32
	}{
		"empty prefix": {
			prefixToClear:   []byte{},
			metricsNodesSub: 14,
		},
		"0 prefix": {
			prefixToClear:   []byte{0x0},
			metricsNodesSub: 2,
		},
		"1 prefix": {
			prefixToClear:   []byte{0x01},
			metricsNodesSub: 2,
		},
		"0x0130 prefix": {
			prefixToClear:   []byte{0x01, 0x30},
			metricsNodesSub: 5,
		},
		"0x0135 prefix": {
			prefixToClear:   []byte{0x01, 0x35},
			metricsNodesSub: 5,
		},
		"0x013570 prefix": {
			prefixToClear:   []byte{0x01, 0x35, 0x70},
			metricsNodesSub: 1,
		},
		"0x013579 prefix": {
			prefixToClear:   []byte{0x01, 0x35, 0x79},
			metricsNodesSub: 3,
		},
		"0x013579ab prefix": {
			prefixToClear:   []byte{0x01, 0x35, 0x79, 0xab},
			metricsNodesSub: 2,
		},
		"0x07 prefix": {
			prefixToClear:   []byte{0x07},
			metricsNodesSub: 2,
		},
		"0x0730 prefix": {
			prefixToClear:   []byte{0x07, 0x30},
			metricsNodesSub: 4,
		},
		"0xf0 prefix": {
			prefixToClear:   []byte{0xf0},
			metricsNodesSub: 2,
		},
		"0xffeeddccbb11 prefix": {
			prefixToClear:   []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0x11},
			metricsNodesSub: 2,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			metrics := NewMockMetrics(ctrl)
			var previousCall *gomock.Call
			for _, n := range []uint32{1, 1, 1, 1, 2, 2, 2, 2, 2} {
				call := metrics.EXPECT().NodesAdd(n)
				if previousCall != nil {
					call.After(previousCall)
				}
				previousCall = call
			}

			trie := NewEmptyTrie(metrics)

			for _, test := range trieKeyValues {
				trie.Put(test.key, test.value)
			}

			dcTrie := trie.DeepCopy()

			// Take Snapshot of the trie.
			ssTrie := trie.Snapshot()

			// Get the Trie root hash for all the 3 tries.
			tHash, err := trie.Hash()
			require.NoError(t, err)

			dcTrieHash, err := dcTrie.Hash()
			require.NoError(t, err)

			ssTrieHash, err := ssTrie.Hash()
			require.NoError(t, err)

			// Root hash for all the 3 tries should be equal.
			require.Equal(t, tHash, dcTrieHash)
			require.Equal(t, dcTrieHash, ssTrieHash)

			metrics.EXPECT().NodesSub(testCase.metricsNodesSub).After(previousCall)
			ssTrie.ClearPrefix(testCase.prefixToClear)
			prefixNibbles := codec.KeyLEToNibbles(testCase.prefixToClear)
			if len(prefixNibbles) > 0 && prefixNibbles[len(prefixNibbles)-1] == 0 {
				prefixNibbles = prefixNibbles[:len(prefixNibbles)-1]
			}

			for _, test := range trieKeyValues {
				res := ssTrie.Get(test.key)

				keyNibbles := codec.KeyLEToNibbles(test.key)
				length := lenCommonPrefix(keyNibbles, prefixNibbles)
				if length == len(prefixNibbles) {
					require.Nil(t, res)
				} else {
					require.Equal(t, test.value, res)
				}
			}

			// Get the updated root hash of all tries.
			tHash, err = trie.Hash()
			require.NoError(t, err)

			dcTrieHash, err = dcTrie.Hash()
			require.NoError(t, err)

			ssTrieHash, err = ssTrie.Hash()
			require.NoError(t, err)

			// Only the current trie should have a different root hash since it is updated.
			require.NotEqual(t, ssTrieHash, dcTrieHash)
			require.NotEqual(t, ssTrieHash, tHash)
			require.Equal(t, dcTrieHash, tHash)
		})

	}
}

func TestClearPrefix_Small(t *testing.T) {
	ctrl := gomock.NewController(t)

	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	metrics := NewMockMetrics(ctrl)
	call := metrics.EXPECT().NodesAdd(uint32(1))
	call = metrics.EXPECT().NodesAdd(uint32(2)).After(call)
	call = metrics.EXPECT().NodesAdd(uint32(2)).After(call)
	metrics.EXPECT().NodesSub(uint32(4)).After(call)

	trie := NewEmptyTrie(metrics)

	dcTrie := trie.DeepCopy()

	// Take Snapshot of the trie.
	ssTrie := trie.Snapshot()

	// Get the Trie root hash for all the 3 tries.
	tHash, err := trie.Hash()
	require.NoError(t, err)

	dcTrieHash, err := dcTrie.Hash()
	require.NoError(t, err)

	ssTrieHash, err := ssTrie.Hash()
	require.NoError(t, err)

	// Root hash for all the 3 tries should be equal.
	require.Equal(t, tHash, dcTrieHash)
	require.Equal(t, dcTrieHash, ssTrieHash)

	for _, key := range keys {
		ssTrie.Put([]byte(key), []byte(key))
	}

	ssTrie.ClearPrefix([]byte("noo"))

	expectedRoot := &node.Leaf{
		Key:        codec.KeyLEToNibbles([]byte("other")),
		Value:      []byte("other"),
		Generation: 1,
	}
	expectedRoot.SetDirty(true)

	require.Equal(t, expectedRoot, ssTrie.root)

	// Get the updated root hash of all tries.
	tHash, err = trie.Hash()
	require.NoError(t, err)

	dcTrieHash, err = dcTrie.Hash()
	require.NoError(t, err)

	ssTrieHash, err = ssTrie.Hash()
	require.NoError(t, err)

	// Only the current trie should have a different root hash since it is updated.
	require.NotEqual(t, ssTrie, dcTrieHash)
	require.NotEqual(t, ssTrie, tHash)
	require.Equal(t, dcTrieHash, tHash)
}

func TestTrie_ClearPrefixVsDelete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"A": {keyValues: getDBKeyValuesA()},
		"B": {keyValues: getDBKeyValuesB()},
		"C": {keyValues: getDBKeyValuesC()},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			prefixes := [][]byte{
				{},
				{0x0},
				{0x01},
				{0x01, 0x35},
				{0xf},
				{0xf2},
				{0x01, 0x30},
				{0x01, 0x35, 0x70},
				{0x01, 0x35, 0x77},
				{0xf2, 0x0},
				{0x07},
				{0x09},
				[]byte("a"),
			}

			for _, prefix := range prefixes {
				metrics := metricsnoop.New()

				trieDelete := NewEmptyTrie(metrics)
				trieClearPrefix := NewEmptyTrie(metrics)

				for _, keyValue := range testCase.keyValues {
					trieDelete.Put(keyValue.key, keyValue.value)
					trieClearPrefix.Put(keyValue.key, keyValue.value)
				}

				prefixedKeys := trieDelete.GetKeysWithPrefix(prefix)
				for _, key := range prefixedKeys {
					trieDelete.Delete(key)
				}

				trieClearPrefix.ClearPrefix(prefix)

				require.Equal(t, trieClearPrefix.MustHash(), trieDelete.MustHash())
			}
		})
	}
}

func TestSnapshot(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	keyValues := []keyValue{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi")},
		{key: []byte{0x01, 0x35, 0x79, 0xab}, value: []byte("spaghetti")},
		{key: []byte{0x01, 0x35, 0x79, 0xab, 0x9}, value: []byte("gnocchi")},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen")},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles")},
		{key: []byte{0xf2}, value: []byte("pho")},
	}

	metrics := NewMockMetrics(ctrl)
	var previousCall *gomock.Call
	for _, n := range []uint32{1, 1, 1, 1, 2, 2, 2} {
		call := metrics.EXPECT().NodesAdd(n)
		if previousCall != nil {
			call.After(previousCall)
		}
		previousCall = call
	}

	expectedTrie := NewEmptyTrie(metrics)
	for _, keyValue := range keyValues {
		expectedTrie.Put(keyValue.key, keyValue.value)
	}

	// put all keys except first
	for _, n := range []uint32{1, 1, 1, 2, 2, 2, 1} {
		call := metrics.EXPECT().NodesAdd(n)
		if previousCall != nil {
			call.After(previousCall)
		}
		previousCall = call
	}

	parentTrie := NewEmptyTrie(metrics)
	for i, keyValue := range keyValues {
		if i == 0 {
			continue
		}
		parentTrie.Put(keyValue.key, keyValue.value)
	}

	newTrie := parentTrie.Snapshot()
	newTrie.Put(keyValues[0].key, keyValues[0].value)

	require.Equal(t, expectedTrie.MustHash(), newTrie.MustHash())
	require.NotEqual(t, parentTrie.MustHash(), newTrie.MustHash())
}

func Test_Trie_NextKey_Random(t *testing.T) {
	generator := newGenerator()
	metrics := metricsnoop.New() // random values so cannot predict them

	trie := NewEmptyTrie(metrics)

	const minKVSize, maxKVSize = 1000, 10000
	kvSize := minKVSize + generator.Intn(maxKVSize-minKVSize)
	kv := generateKeyValues(t, generator, kvSize)

	sortedKeys := make([][]byte, 0, len(kv))
	for keyString := range kv {
		key := []byte(keyString)
		sortedKeys = append(sortedKeys, key)
	}

	sort.Slice(sortedKeys, func(i, j int) bool {
		return bytes.Compare(sortedKeys[i], sortedKeys[j]) < 0
	})

	for _, key := range sortedKeys {
		value := []byte{1}
		trie.Put(key, value)
	}

	for i, key := range sortedKeys {
		nextKey := trie.NextKey(key)

		var expectedNextKey []byte
		isLastKey := i == len(sortedKeys)-1
		if !isLastKey {
			expectedNextKey = sortedKeys[i+1]
		}
		require.Equal(t, expectedNextKey, nextKey)
	}
}

func Benchmark_Trie_Hash(b *testing.B) {
	generator := newGenerator()
	const kvSize = 1000000
	kv := generateKeyValues(b, generator, kvSize)

	metrics := metricsnoop.New()
	trie := NewEmptyTrie(metrics)
	for keyString, value := range kv {
		key := []byte(keyString)
		trie.Put(key, value)
	}

	b.StartTimer()
	_, err := trie.Hash()
	b.StopTimer()

	require.NoError(b, err)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func TestTrie_ConcurrentSnapshotWrites(t *testing.T) {
	ctrl := gomock.NewController(t)

	generator := newGenerator()
	const size = 1000
	const workers = 4

	testCases := make([][]Test, workers)
	expectedTries := make([]*Trie, workers)

	for i := 0; i < workers; i++ {
		testCases[i] = make([]Test, size)
		mockMetrics := NewMockMetrics(ctrl)
		expectedTries[i] = buildSmallTrie(mockMetrics)
		mockMetrics.EXPECT().NodesAdd(gomock.AssignableToTypeOf(uint32(0))).AnyTimes()
		mockMetrics.EXPECT().NodesSub(gomock.AssignableToTypeOf(uint32(0))).AnyTimes()
		for j := 0; j < size; j++ {
			k := make([]byte, 2)
			_, err := generator.Read(k)
			require.NoError(t, err)
			op := generator.Intn(3)

			switch op {
			case put:
				expectedTries[i].Put(k, k)
			case del:
				expectedTries[i].Delete(k)
			case clearPrefix:
				expectedTries[i].ClearPrefix(k)
			}

			testCases[i][j] = Test{
				key: k,
				op:  op,
			}
		}
	}

	startWg := new(sync.WaitGroup)
	finishWg := new(sync.WaitGroup)
	startWg.Add(workers)
	finishWg.Add(workers)
	snapshotedTries := make([]*Trie, workers)

	var builtTrie *Trie
	for i := 0; i < workers; i++ {
		mockMetrics := NewMockMetrics(ctrl)
		builtTrie = buildSmallTrie(mockMetrics)
		builtTrie.metrics = metricsnoop.New() // so it's fast
		snapshotedTries[i] = builtTrie.Snapshot()

		go func(trie *Trie, operations []Test,
			startWg, finishWg *sync.WaitGroup) {
			defer finishWg.Done()
			startWg.Done()
			startWg.Wait()
			for _, operation := range operations {
				switch operation.op {
				case put:
					trie.Put(operation.key, operation.key)
				case del:
					trie.Delete(operation.key)
				case clearPrefix:
					trie.ClearPrefix(operation.key)
				}
			}
		}(snapshotedTries[i], testCases[i], startWg, finishWg)
	}

	finishWg.Wait()

	for i := 0; i < workers; i++ {
		assert.Equal(t,
			expectedTries[i].MustHash(),
			snapshotedTries[i].MustHash())
	}
}

func TestTrie_ClearPrefixLimit(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"custom 1": {keyValues: []keyValue{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x36}, value: []byte("pencil")},
			{key: []byte{0x02}, value: []byte("feather")},
			{key: []byte{0x03}, value: []byte("birds")},
		}},
		"custom 2": {keyValues: []keyValue{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0x01, 0x35, 0x99}, value: []byte("h")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		}},
		"B": {keyValues: getDBKeyValuesB()},
		"C": {keyValues: getDBKeyValuesC()},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			prefixes := [][]byte{
				{},
				{0x00},
				{0x01},
				{0x01, 0x35},
				{0xf0},
				{0xf2},
				{0x01, 0x30},
				{0x01, 0x35, 0x70},
				{0x01, 0x35, 0x77},
				{0xf2, 0x0},
				{0x07},
				{0x09},
			}

			for _, prefix := range prefixes {
				prefixNibbles := codec.KeyLEToNibbles(prefix)
				if len(prefixNibbles) > 0 && prefixNibbles[len(prefixNibbles)-1] == 0 {
					prefixNibbles = prefixNibbles[:len(prefixNibbles)-1]
				}

				for lim := 0; lim < len(testCase.keyValues)+1; lim++ {
					metrics := metricsnoop.New() // hard to predict with prefix for loop
					trieClearPrefix := NewEmptyTrie(metrics)

					for _, keyValue := range testCase.keyValues {
						trieClearPrefix.Put(keyValue.key, keyValue.value)
					}

					num, allDeleted := trieClearPrefix.ClearPrefixLimit(prefix, uint32(lim))
					deleteCount := uint32(0)
					isAllDeleted := true

					for _, keyValue := range testCase.keyValues {
						val := trieClearPrefix.Get(keyValue.key)

						keyNibbles := codec.KeyLEToNibbles(keyValue.key)
						length := lenCommonPrefix(keyNibbles, prefixNibbles)

						if length == len(prefixNibbles) {
							if val == nil {
								deleteCount++
							} else {
								isAllDeleted = false
								require.Equal(t, keyValue.value, val)
							}
						} else {
							require.NotNil(t, val)
						}
					}
					require.Equal(t, num, deleteCount)
					require.LessOrEqual(t, deleteCount, uint32(lim))
					if lim > 0 {
						require.Equal(t, allDeleted, isAllDeleted)
					}
				}
			}
		})
	}
}

func TestTrie_ClearPrefixLimitSnapshot(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyValues []keyValue
	}{
		"custom 1": {keyValues: []keyValue{
			{key: []byte{0x01}, value: []byte("feather")},
		}},
		"custom 2": {keyValues: []keyValue{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x36}, value: []byte("pencil")},
			{key: []byte{0x02}, value: []byte("feather")},
			{key: []byte{0x03}, value: []byte("birds")},
		}},
		"custom 3": {keyValues: []keyValue{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0x01, 0x35, 0x99}, value: []byte("h")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		}},
		"B": {keyValues: getDBKeyValuesB()},
		"C": {keyValues: getDBKeyValuesC()},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			prefixes := [][]byte{
				{},
				{0x00},
				{0x01},
				{0x01, 0x35},
				{0xf0},
				{0xf2},
				{0x01, 0x30},
				{0x01, 0x35, 0x70},
				{0x01, 0x35, 0x77},
				{0xf2, 0x0},
				{0x07},
				{0x09},
			}

			for _, prefix := range prefixes {
				prefixNibbles := codec.KeyLEToNibbles(prefix)
				if len(prefixNibbles) > 0 && prefixNibbles[len(prefixNibbles)-1] == 0 {
					prefixNibbles = prefixNibbles[:len(prefixNibbles)-1]
				}

				for lim := 0; lim < len(testCase.keyValues)+1; lim++ {
					metrics := metricsnoop.New() // hard to predict with prefix for loop
					trieClearPrefix := NewEmptyTrie(metrics)

					for _, keyValue := range testCase.keyValues {
						trieClearPrefix.Put(keyValue.key, keyValue.value)
					}

					dcTrie := trieClearPrefix.DeepCopy()

					// Take Snapshot of the trie.
					ssTrie := trieClearPrefix.Snapshot()

					// Get the Trie root hash for all the 3 tries.
					tHash, err := trieClearPrefix.Hash()
					require.NoError(t, err)

					dcTrieHash, err := dcTrie.Hash()
					require.NoError(t, err)

					ssTrieHash, err := ssTrie.Hash()
					require.NoError(t, err)

					// Root hash for all the 3 tries should be equal.
					require.Equal(t, tHash, dcTrieHash)
					require.Equal(t, dcTrieHash, ssTrieHash)

					num, allDeleted := ssTrie.ClearPrefixLimit(prefix, uint32(lim))
					deleteCount := uint32(0)
					isAllDeleted := true

					for _, keyValue := range testCase.keyValues {
						val := ssTrie.Get(keyValue.key)

						keyNibbles := codec.KeyLEToNibbles(keyValue.key)
						length := lenCommonPrefix(keyNibbles, prefixNibbles)

						if length == len(prefixNibbles) {
							if val == nil {
								deleteCount++
							} else {
								isAllDeleted = false
								require.Equal(t, keyValue.value, val)
							}
						} else {
							require.NotNil(t, val)
						}
					}
					require.LessOrEqual(t, deleteCount, uint32(lim))
					require.Equal(t, num, deleteCount)
					if lim > 0 {
						require.Equal(t, allDeleted, isAllDeleted)
					}

					// Get the updated root hash of all tries.
					tHash, err = trieClearPrefix.Hash()
					require.NoError(t, err)

					dcTrieHash, err = dcTrie.Hash()
					require.NoError(t, err)

					ssTrieHash, err = ssTrie.Hash()
					require.NoError(t, err)

					// If node got deleted then root hash must be updated else it has same root hash.
					if num > 0 {
						require.NotEqual(t, ssTrieHash, dcTrieHash)
						require.NotEqual(t, ssTrieHash, tHash)
					} else {
						require.Equal(t, ssTrieHash, tHash)
					}

					require.Equal(t, dcTrieHash, tHash)
				}
			}
		})
	}
}

func Test_encodeRoot_fuzz(t *testing.T) {
	t.Parallel()

	generator := newGenerator()

	metrics := metricsnoop.New() // random values so cannot predict them
	trie := NewEmptyTrie(metrics)

	const randomBatches = 3

	for i := 0; i < randomBatches; i++ {
		const kvSize = 16
		kv := generateKeyValues(t, generator, kvSize)
		for keyString, value := range kv {
			key := []byte(keyString)
			trie.Put(key, value)

			retrievedValue := trie.Get(key)
			assert.Equal(t, value, retrievedValue)
		}
		buffer := bytes.NewBuffer(nil)
		err := trie.root.Encode(buffer)
		require.NoError(t, err)
		require.NotEmpty(t, buffer.Bytes())
	}
}
