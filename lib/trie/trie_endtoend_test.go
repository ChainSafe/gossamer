// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
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

func buildSmallTrie() *Trie {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{}, value: []byte("floof")},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd")},
	}

	for _, test := range tests {
		trie.Put(test.key, test.value)
	}

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
	trie := NewEmptyTrie()

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
	trie := NewEmptyTrie()

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

	trie := NewEmptyTrie()

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
	trie := NewEmptyTrie()

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
	trie := buildSmallTrie()

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
}

func TestDeleteCombineBranch(t *testing.T) {
	trie := buildSmallTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35, 0x46}, value: []byte("raccoon"), op: put},
		{key: []byte{0x01, 0x35, 0x46, 0x77}, value: []byte("rat"), op: put},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: del},
		{key: []byte{0x09, 0xd3}, value: nil, op: get},
	}

	runTests(t, trie, tests)
}

func TestDeleteFromBranch(t *testing.T) {
	trie := NewEmptyTrie()

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
}

func TestDeleteOddKeyLengths(t *testing.T) {
	trie := NewEmptyTrie()

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
}

func TestTrieDiff(t *testing.T) {
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

	trie := NewEmptyTrie()

	var testKey = []byte("testKey")

	tests := []Test{
		{key: testKey, value: testKey},
		{key: []byte("testKey1"), value: []byte("testKey1")},
		{key: []byte("testKey2"), value: []byte("testKey2")},
	}

	for _, test := range tests {
		trie.Put(test.key, test.value)
	}

	newTrie := trie.Snapshot()
	err = trie.Store(storageDB)
	require.NoError(t, err)

	tests = []Test{
		{key: testKey, value: []byte("newTestKey2")},
		{key: []byte("testKey2"), value: []byte("newKey")},
		{key: []byte("testKey3"), value: []byte("testKey3")},
		{key: []byte("testKey4"), value: []byte("testKey2")},
		{key: []byte("testKey5"), value: []byte("testKey5")},
	}

	for _, test := range tests {
		newTrie.Put(test.key, test.value)
	}
	deletedKeys := newTrie.deletedKeys
	require.Len(t, deletedKeys, 3)

	err = newTrie.WriteDirty(storageDB)
	require.NoError(t, err)

	for key := range deletedKeys {
		err = storageDB.Del(key.ToBytes())
		require.NoError(t, err)
	}

	dbTrie := NewEmptyTrie()
	err = dbTrie.Load(storageDB, common.BytesToHash(newTrie.root.GetHash()))
	require.NoError(t, err)
}

func TestDelete(t *testing.T) {
	trie := NewEmptyTrie()

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
	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: put},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: put},
		{key: []byte{0x01, 0x35, 0x79, 0xab}, value: []byte("spaghetti"), op: put},
		{key: []byte{0x01, 0x35, 0x79, 0xab, 0x9}, value: []byte("gnocchi"), op: put},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen"), op: put},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles"), op: put},
		{key: []byte{0xf2}, value: []byte("pho"), op: put},
		{key: []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0x11}, value: []byte("asd"), op: put},
		{key: []byte{0xff, 0xee, 0xdd, 0xcc, 0xaa, 0x11}, value: []byte("fgh"), op: put},
	}

	// prefix to clear cases
	testCases := [][]byte{
		{},
		{0x0},
		{0x01},
		{0x01, 0x30},
		{0x01, 0x35},
		{0x01, 0x35, 0x70},
		{0x01, 0x35, 0x79},
		{0x01, 0x35, 0x79, 0xab},
		{0x07},
		{0x07, 0x30},
		{0xf0},
		{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0x11},
	}

	for _, prefix := range testCases {
		trie := NewEmptyTrie()

		for _, test := range tests {
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

		ssTrie.ClearPrefix(prefix)
		prefixNibbles := codec.KeyLEToNibbles(prefix)
		if len(prefixNibbles) > 0 && prefixNibbles[len(prefixNibbles)-1] == 0 {
			prefixNibbles = prefixNibbles[:len(prefixNibbles)-1]
		}

		for _, test := range tests {
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
	}
}

func TestClearPrefix_Small(t *testing.T) {
	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	trie := NewEmptyTrie()

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

	cases := [][]Test{
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0x01, 0x35, 0x99}, value: []byte("h")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
			{key: []byte{0}, value: nil},
		},
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x30}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
		{
			{key: []byte("asdf"), value: []byte("asdf")},
			{key: []byte("ghjk"), value: []byte("ghjk")},
			{key: []byte("qwerty"), value: []byte("qwerty")},
			{key: []byte("uiopl"), value: []byte("uiopl")},
			{key: []byte("zxcv"), value: []byte("zxcv")},
			{key: []byte("bnm"), value: []byte("bnm")},
		},
	}

	for _, testCase := range cases {
		for _, prefix := range prefixes {
			trieDelete := NewEmptyTrie()
			trieClearPrefix := NewEmptyTrie()

			for _, test := range testCase {
				trieDelete.Put(test.key, test.value)
				trieClearPrefix.Put(test.key, test.value)
			}

			prefixedKeys := trieDelete.GetKeysWithPrefix(prefix)
			for _, key := range prefixedKeys {
				trieDelete.Delete(key)
			}

			trieClearPrefix.ClearPrefix(prefix)

			require.Equal(t, trieClearPrefix.MustHash(), trieDelete.MustHash())
		}
	}
}

func TestSnapshot(t *testing.T) {
	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: put},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: put},
		{key: []byte{0x01, 0x35, 0x79, 0xab}, value: []byte("spaghetti"), op: put},
		{key: []byte{0x01, 0x35, 0x79, 0xab, 0x9}, value: []byte("gnocchi"), op: put},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen"), op: put},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles"), op: put},
		{key: []byte{0xf2}, value: []byte("pho"), op: put},
	}

	expectedTrie := NewEmptyTrie()
	for _, test := range tests {
		expectedTrie.Put(test.key, test.value)
	}

	// put all keys except first
	parentTrie := NewEmptyTrie()
	for i, test := range tests {
		if i == 0 {
			continue
		}
		parentTrie.Put(test.key, test.value)
	}

	newTrie := parentTrie.Snapshot()
	newTrie.Put(tests[0].key, tests[0].value)

	require.Equal(t, expectedTrie.MustHash(), newTrie.MustHash())
	require.NotEqual(t, parentTrie.MustHash(), newTrie.MustHash())
}

func TestNextKey_Random(t *testing.T) {
	generator := newGenerator()

	for i := 0; i < 100; i++ {
		trie := NewEmptyTrie()

		// Generate random test cases.
		testCaseMap := make(map[string]struct{}) // ensure no duplicate keys
		size := 1000 + rand.Intn(10000)

		for ii := 0; ii < size; ii++ {
			const minLen, maxLen = 1, 20
			str := generateRandString(minLen, maxLen, generator)
			if str == "" {
				continue
			}
			testCaseMap[str] = struct{}{}
		}

		testCases := make([][]byte, len(testCaseMap))
		j := 0

		for k := range testCaseMap {
			testCases[j] = []byte(k)
			j++
		}

		sort.Slice(testCases, func(i, j int) bool {
			return bytes.Compare(testCases[i], testCases[j]) < 0
		})

		for _, tc := range testCases {
			trie.Put(tc, tc)
		}

		for idx, tc := range testCases {
			next := trie.NextKey(tc)
			if idx == len(testCases)-1 {
				require.Nil(t, next)
			} else {
				require.Equal(t, testCases[idx+1], next, common.BytesToHex(tc))
			}
		}
	}
}

func Benchmark_Trie_Hash(b *testing.B) {
	generator := newGenerator()
	const kvSize = 1000000
	kv := generateKeyValues(b, generator, kvSize)

	trie := NewEmptyTrie()
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
	base := buildSmallTrie()
	size := 65536

	testCasesA := make([]Test, size)
	expectedA := buildSmallTrie()
	for i := 0; i < size; i++ {
		k := make([]byte, 2)
		_, err := rand.Read(k)
		require.NoError(t, err)
		op := rand.Intn(3)

		switch op {
		case put:
			expectedA.Put(k, k)
		case del:
			expectedA.Delete(k)
		case clearPrefix:
			expectedA.ClearPrefix(k)
		}

		testCasesA[i] = Test{
			key: k,
			op:  op,
		}
	}

	testCasesB := make([]Test, size)
	expectedB := buildSmallTrie()
	for i := 0; i < size; i++ {
		k := make([]byte, 2)
		_, err := rand.Read(k)
		require.NoError(t, err)
		op := rand.Intn(3)

		switch op {
		case put:
			expectedB.Put(k, k)
		case del:
			expectedB.Delete(k)
		case clearPrefix:
			expectedB.ClearPrefix(k)
		}

		testCasesB[i] = Test{
			key: k,
			op:  op,
		}
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)
	trieA := base.Snapshot()
	trieB := base.Snapshot()

	go func() {
		for _, tc := range testCasesA {
			switch tc.op {
			case put:
				trieA.Put(tc.key, tc.key)
			case del:
				trieA.Delete(tc.key)
			case clearPrefix:
				trieA.ClearPrefix(tc.key)
			}
		}
		wg.Done()
	}()

	go func() {
		for _, tc := range testCasesB {
			switch tc.op {
			case put:
				trieB.Put(tc.key, tc.key)
			case del:
				trieB.Delete(tc.key)
			case clearPrefix:
				trieB.ClearPrefix(tc.key)
			}
		}
		wg.Done()
	}()

	wg.Wait()

	require.Equal(t, expectedA.MustHash(), trieA.MustHash())
	require.Equal(t, expectedB.MustHash(), trieB.MustHash())
}

func TestTrie_ClearPrefixLimit(t *testing.T) {
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

	cases := [][]Test{
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x36}, value: []byte("pencil")},
			{key: []byte{0x02}, value: []byte("feather")},
			{key: []byte{0x03}, value: []byte("birds")},
		},
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0x01, 0x35, 0x99}, value: []byte("h")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x30}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
		{
			{key: []byte("asdf"), value: []byte("asdf")},
			{key: []byte("ghjk"), value: []byte("ghjk")},
			{key: []byte("qwerty"), value: []byte("qwerty")},
			{key: []byte("uiopl"), value: []byte("uiopl")},
			{key: []byte("zxcv"), value: []byte("zxcv")},
			{key: []byte("bnm"), value: []byte("bnm")},
		},
	}

	testFn := func(t *testing.T, testCase []Test, prefix []byte) {
		prefixNibbles := codec.KeyLEToNibbles(prefix)
		if len(prefixNibbles) > 0 && prefixNibbles[len(prefixNibbles)-1] == 0 {
			prefixNibbles = prefixNibbles[:len(prefixNibbles)-1]
		}

		for lim := 0; lim < len(testCase)+1; lim++ {
			trieClearPrefix := NewEmptyTrie()

			for _, test := range testCase {
				trieClearPrefix.Put(test.key, test.value)
			}

			num, allDeleted := trieClearPrefix.ClearPrefixLimit(prefix, uint32(lim))
			deleteCount := uint32(0)
			isAllDeleted := true

			for _, test := range testCase {
				val := trieClearPrefix.Get(test.key)

				keyNibbles := codec.KeyLEToNibbles(test.key)
				length := lenCommonPrefix(keyNibbles, prefixNibbles)

				if length == len(prefixNibbles) {
					if val == nil {
						deleteCount++
					} else {
						isAllDeleted = false
						require.Equal(t, test.value, val)
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

	for _, testCase := range cases {
		for _, prefix := range prefixes {
			testFn(t, testCase, prefix)
		}
	}
}

func TestTrie_ClearPrefixLimitSnapshot(t *testing.T) {
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

	cases := [][]Test{
		{
			{key: []byte{0x01}, value: []byte("feather")},
		},
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x36}, value: []byte("pencil")},
			{key: []byte{0x02}, value: []byte("feather")},
			{key: []byte{0x03}, value: []byte("birds")},
		},
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0x01, 0x35, 0x99}, value: []byte("h")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x30}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
		{
			{key: []byte("asdf"), value: []byte("asdf")},
			{key: []byte("ghjk"), value: []byte("ghjk")},
			{key: []byte("qwerty"), value: []byte("qwerty")},
			{key: []byte("uiopl"), value: []byte("uiopl")},
			{key: []byte("zxcv"), value: []byte("zxcv")},
			{key: []byte("bnm"), value: []byte("bnm")},
		},
	}

	for _, testCase := range cases {
		for _, prefix := range prefixes {
			prefixNibbles := codec.KeyLEToNibbles(prefix)
			if len(prefixNibbles) > 0 && prefixNibbles[len(prefixNibbles)-1] == 0 {
				prefixNibbles = prefixNibbles[:len(prefixNibbles)-1]
			}

			for lim := 0; lim < len(testCase)+1; lim++ {
				trieClearPrefix := NewEmptyTrie()

				for _, test := range testCase {
					trieClearPrefix.Put(test.key, test.value)
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

				for _, test := range testCase {
					val := ssTrie.Get(test.key)

					keyNibbles := codec.KeyLEToNibbles(test.key)
					length := lenCommonPrefix(keyNibbles, prefixNibbles)

					if length == len(prefixNibbles) {
						if val == nil {
							deleteCount++
						} else {
							isAllDeleted = false
							require.Equal(t, test.value, val)
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
	}
}

func Test_encodeRoot_fuzz(t *testing.T) {
	generator := newGenerator()

	trie := NewEmptyTrie()

	const randomBatches = 20

	for i := 0; i < randomBatches; i++ {
		const kvSize = 16
		kv := generateKeyValues(t, generator, kvSize)
		for keyString, value := range kv {
			key := []byte(keyString)
			trie.Put(key, value)

			retrievedValue := trie.Get(key)
			assert.Equal(t, value, retrievedValue)

			buffer := bytes.NewBuffer(nil)
			err := trie.root.Encode(buffer)
			require.NoError(t, err)
			require.NotEmpty(t, buffer.Bytes())
		}
	}
}
