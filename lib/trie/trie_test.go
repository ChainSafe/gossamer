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
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
)

type commonPrefixTest struct {
	a, b   []byte
	output int
}

var commonPrefixTests = []commonPrefixTest{
	{a: []byte{}, b: []byte{}, output: 0},
	{a: []byte{0x00}, b: []byte{}, output: 0},
	{a: []byte{0x00}, b: []byte{0x00}, output: 1},
	{a: []byte{0x00}, b: []byte{0x00, 0x01}, output: 1},
	{a: []byte{0x01}, b: []byte{0x00, 0x01, 0x02}, output: 0},
	{a: []byte{0x00, 0x01, 0x02, 0x00}, b: []byte{0x00, 0x01, 0x02}, output: 3},
	{a: []byte{0x00, 0x01, 0x02, 0x00, 0xff}, b: []byte{0x00, 0x01, 0x02, 0x00}, output: 4},
	{a: []byte{0x00, 0x01, 0x02, 0x00, 0xff}, b: []byte{0x00, 0x01, 0x02, 0x00, 0xff, 0x00}, output: 5},
}

func TestCommonPrefix(t *testing.T) {
	for i, test := range commonPrefixTests {
		test := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output := lenCommonPrefix(test.a, test.b)
			if output != test.output {
				t.Errorf("Fail: got %d expected %d", output, test.output)
			}
		})
	}
}

//nolint:revive
const (
	PUT          = 0
	DEL          = 1
	CLEAR_PREFIX = 2
	GET          = 3
	GETLEAF      = 4
)

func TestNewEmptyTrie(t *testing.T) {
	trie := NewEmptyTrie()
	if trie == nil {
		t.Error("did not initialise trie")
	}
}

func TestNewTrie(t *testing.T) {
	trie := NewTrie(&node.Leaf{Key: []byte{0}, Value: []byte{17}})
	if trie == nil {
		t.Error("did not initialise trie")
	}
}

func TestEntries(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
	}

	for _, test := range tests {
		trie.Put(test.key, test.value)
	}

	entries := trie.Entries()
	if len(entries) != len(tests) {
		t.Fatal("length of trie.Entries does not equal length of values put into trie")
	}

	for _, test := range tests {
		if entries[string(test.key)] == nil {
			t.Fatal("did not get entry in trie")
		}
	}
}

func hexDecode(in string) []byte {
	out, _ := hex.DecodeString(in)
	return out
}

func writeToTestFile(tests []Test) error {
	testString := ""
	for _, test := range tests {
		testString = fmt.Sprintf("%s%s\n%s\n", testString, test.key, test.value)
	}

	fp, err := filepath.Abs("./failing_test_data")
	if err != nil {
		return err
	}
	os.Remove(fp)
	err = os.WriteFile(fp, []byte(testString), 0644)
	if err != nil {
		return err
	}

	return nil
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
	for i, test := range tests {
		test := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if test.op == PUT {
				trie.Put(test.key, test.value)
			} else if test.op == GET {
				val := trie.Get(test.key)
				if !bytes.Equal(val, test.value) {
					t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
				}
			} else if test.op == DEL {
				trie.Delete(test.key)
			} else if test.op == GETLEAF {
				leaf := trie.tryGet(test.key)
				if leaf == nil {
					t.Errorf("Fail to get key %x: nil leaf", test.key)
				} else if !bytes.Equal(leaf.Value, test.value) {
					t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, leaf.Value)
				} else if !bytes.Equal(leaf.Key, test.pk) {
					t.Errorf("Fail to get correct partial key %x with key %x: got %x", test.pk, test.key, leaf.Key)
				}
			}
		})
	}
}

func TestLoadTrieFromMap(t *testing.T) {
	data := map[string]string{"0x1234": "0x5678", "0xaabbcc": "0xddeeff"}
	testTrie := &Trie{}

	err := testTrie.LoadFromMap(data)
	if err != nil {
		t.Fatal(err)
	}

	expectedTrie := &Trie{}
	var keyBytes, valueBytes []byte
	for key, value := range data {
		keyBytes, err = common.HexToBytes(key)
		if err != nil {
			t.Fatal(err)
		}
		valueBytes, err = common.HexToBytes(value)
		if err != nil {
			t.Fatal(err)
		}
		expectedTrie.Put(keyBytes, valueBytes)
	}

	testhash, err := testTrie.Hash()
	if err != nil {
		t.Fatal(err)
	}
	expectedhash, err := expectedTrie.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(testhash[:], expectedhash[:]) {
		t.Fatalf("Fail: got %x expected %x", testhash, expectedhash)
	}
}

func TestPutAndGetBranch(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x07}, value: []byte("ramen"), op: PUT},
		{key: []byte{0xf2}, value: []byte("pho"), op: PUT},
		{key: []byte("noot"), value: nil, op: GET},
		{key: []byte{0}, value: nil, op: GET},
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: GET},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: GET},
		{key: []byte{0x07}, value: []byte("ramen"), op: GET},
		{key: []byte{0xf2}, value: []byte("pho"), op: GET},
	}

	runTests(t, trie, tests)
}

func TestPutAndGetOddKeyLengths(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: PUT},
		{key: []byte{0x49, 0x29}, value: []byte("nootagain"), op: PUT},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: PUT},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: PUT},
		{key: []byte{0x4f, 0xbc}, value: []byte("stuffagain"), op: PUT},
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: GET},
		{key: []byte{0x49, 0x29}, value: []byte("nootagain"), op: GET},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: GET},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: GET},
		{key: []byte{0x4f, 0xbc}, value: []byte("stuffagain"), op: GET},
	}

	runTests(t, trie, tests)
}

func TestPutAndGet(t *testing.T) {
	for i := 0; i < 10; i++ {
		trie := NewEmptyTrie()
		rt := GenerateRandomTests(t, 10000)
		for _, test := range rt {
			trie.Put(test.key, test.value)

			val := trie.Get(test.key)
			if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}
		}

		for _, test := range rt {
			val := trie.Get(test.key)
			if !bytes.Equal(val, test.value) {
				writeToTestFile(rt)
				t.Fatalf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}
		}
	}
}

// this test is used to debug random tests that fail
// in TestPutAndGet, random tests are generated and if a case fails, it's saved to trie/test_data
// if the trie/test_data exists, this test runs the case in that file
// otherwise it's skipped
func TestFailingTests(t *testing.T) {
	fp, err := filepath.Abs("./failing_test_data")
	if err != nil {
		t.Error(err)
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		t.SkipNow()
	}

	slicedData := strings.Split(string(data), "\n")
	tests := []Test{}
	for i := 0; i < len(slicedData)-2; i += 2 {
		test := Test{key: []byte(slicedData[i]), value: []byte(slicedData[i+1])}
		tests = append(tests, test)
	}

	trie := NewEmptyTrie()

	hasFailed := false
	passedFailingTest := false
	rt := tests
	for i, test := range rt {
		if len(test.key) != 0 {
			trie.Put(test.key, test.value)

			val := trie.Get(test.key)
			if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}

			failingKey := hexDecode("")
			failingVal := hexDecode("")

			if bytes.Equal(test.key, failingKey) {
				passedFailingTest = true
			}

			val = trie.Get(failingKey)
			if !bytes.Equal(val, failingVal) && !hasFailed && passedFailingTest {
				t.Errorf("Fail to get key %x with value %x: got %x", failingKey, failingVal, val)
				t.Logf("test failed at insertion of key %x index %d", test.key, i)
				hasFailed = true
			}
		}
	}

	for _, test := range rt {
		if len(test.key) != 0 {
			val := trie.Get(test.key)
			if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}
		}
	}
}

func TestGetPartialKey(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: PUT},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: PUT},
		{key: []byte{}, value: []byte("floof"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), pk: []byte{9}, op: GETLEAF},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: DEL},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), pk: []byte{0x9}, op: GETLEAF},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), pk: []byte{0x1, 0x3, 0x5}, op: GETLEAF},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: PUT},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), pk: []byte{7}, op: GETLEAF},
		{key: []byte{0xf2}, value: []byte("pen"), op: PUT},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: PUT},
		{key: []byte{}, value: []byte("floof"), op: GET},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), pk: []byte{0x3, 0x5}, op: GETLEAF},
		{key: []byte{0xf2}, value: []byte("pen"), pk: []byte{0x2}, op: GETLEAF},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), pk: []byte{0x0d, 0x03}, op: GETLEAF},
	}

	runTests(t, trie, tests)
}

func TestDeleteSmall(t *testing.T) {
	trie := buildSmallTrie()

	tests := []Test{
		{key: []byte{}, value: []byte("floof"), op: DEL},
		{key: []byte{}, value: nil, op: GET},
		{key: []byte{}, value: []byte("floof"), op: PUT},

		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: DEL},
		{key: []byte{0x09, 0xd3}, value: nil, op: GET},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: GET},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: GET},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: PUT},

		{key: []byte{0xf2}, value: []byte("feather"), op: DEL},
		{key: []byte{0xf2}, value: nil, op: GET},
		{key: []byte{0xf2}, value: []byte("feather"), op: PUT},

		{key: []byte{}, value: []byte("floof"), op: DEL},
		{key: []byte{0xf2}, value: []byte("feather"), op: DEL},
		{key: []byte{}, value: nil, op: GET},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: GET},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: GET},
		{key: []byte{}, value: []byte("floof"), op: PUT},
		{key: []byte{0xf2}, value: []byte("feather"), op: PUT},

		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: DEL},
		{key: []byte{0x01, 0x35, 0x79}, value: nil, op: GET},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: GET},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: PUT},

		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: DEL},
		{key: []byte{0x01, 0x35}, value: nil, op: GET},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: GET},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: PUT},

		{key: []byte{0x01, 0x35, 0x07}, value: []byte("odd"), op: DEL},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin"), op: GET},
		{key: []byte{0x01, 0x35}, value: []byte("pen"), op: GET},
	}

	runTests(t, trie, tests)
}

func TestDeleteCombineBranch(t *testing.T) {
	trie := buildSmallTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35, 0x46}, value: []byte("raccoon"), op: PUT},
		{key: []byte{0x01, 0x35, 0x46, 0x77}, value: []byte("rat"), op: PUT},
		{key: []byte{0x09, 0xd3}, value: []byte("noot"), op: DEL},
		{key: []byte{0x09, 0xd3}, value: nil, op: GET},
	}

	runTests(t, trie, tests)
}

func TestDeleteFromBranch(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x06, 0x15, 0xfc}, value: []byte("noot"), op: PUT},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: PUT},
		{key: []byte{0x06, 0xaf, 0xb1}, value: []byte("odd"), op: PUT},
		{key: []byte{0x06, 0xa3, 0xff}, value: []byte("stuff"), op: PUT},
		{key: []byte{0x43, 0x21}, value: []byte("stuffagain"), op: PUT},
		{key: []byte{0x06, 0x15, 0xfc}, value: []byte("noot"), op: GET},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: GET},
		{key: []byte{0x06, 0x15, 0xfc}, value: []byte("noot"), op: DEL},
		{key: []byte{0x06, 0x15, 0xfc}, value: nil, op: GET},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: GET},
		{key: []byte{0x06, 0xaf, 0xb1}, value: []byte("odd"), op: GET},
		{key: []byte{0x06, 0xaf, 0xb1}, value: []byte("odd"), op: DEL},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: GET},
		{key: []byte{0x06, 0xa3, 0xff}, value: []byte("stuff"), op: GET},
		{key: []byte{0x06, 0xa3, 0xff}, value: []byte("stuff"), op: DEL},
		{key: []byte{0x06, 0x2b, 0xa9}, value: []byte("nootagain"), op: GET},
	}

	runTests(t, trie, tests)
}

func TestDeleteOddKeyLengths(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: PUT},
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: GET},
		{key: []byte{0x49, 0x29}, value: []byte("nootagain"), op: PUT},
		{key: []byte{0x49, 0x29}, value: []byte("nootagain"), op: GET},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: PUT},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: GET},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: PUT},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: GET},
		{key: []byte{0x43, 0x0c}, value: []byte("odd"), op: DEL},
		{key: []byte{0x43, 0x0c}, value: nil, op: GET},
		{key: []byte{0xf4, 0xbc}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0xf4, 0xbc}, value: []byte("spaghetti"), op: GET},
		{key: []byte{0x4f, 0x4d}, value: []byte("stuff"), op: GET},
		{key: []byte{0x43, 0xc1}, value: []byte("noot"), op: GET},
	}

	runTests(t, trie, tests)
}

func TestTrieDiff(t *testing.T) {
	testDataDirPath := t.TempDir()

	cfg := &chaindb.Config{
		DataDir:  testDataDirPath,
		InMemory: false,
	}

	db, err := chaindb.NewBadgerDB(cfg)
	require.NoError(t, err)

	storageDB := chaindb.NewTable(db, "storage")

	defer db.Close()
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

	rt := GenerateRandomTests(t, 100)
	for _, test := range rt {
		trie.Put(test.key, test.value)
	}

	// DeepCopy the trie.
	dcTrie, err := trie.DeepCopy()
	require.NoError(t, err)

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

	for i, test := range rt {
		test := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := rand.Int() % 2
			var val []byte
			switch r {
			case 0:
				ssTrie.Delete(test.key)
				val = ssTrie.Get(test.key)
				if val != nil {
					t.Errorf("Fail to delete key %x with value %x: got %x", test.key, test.value, val)
				}
			case 1:
				val = ssTrie.Get(test.key)
				if !bytes.Equal(test.value, val) {
					t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
				}
			}
		})
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

func TestGetKeysWithPrefix(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen"), op: PUT},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles"), op: PUT},
		{key: []byte{0xf2}, value: []byte("pho"), op: PUT},
		{key: []byte(":key1"), value: []byte("value1"), op: PUT},
		{key: []byte(":key2"), value: []byte("value2"), op: PUT},
		{key: []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0x11}, value: []byte("asd"), op: PUT},
		{key: []byte{0xff, 0xee, 0xdd, 0xcc, 0xaa, 0x11}, value: []byte("fgh"), op: PUT},
	}

	for _, test := range tests {
		trie.Put(test.key, test.value)
	}

	expected := [][]byte{{0x01, 0x35}, {0x01, 0x35, 0x79}}
	keys := trie.GetKeysWithPrefix([]byte{0x01})
	require.Equal(t, expected, keys)

	expected = [][]byte{{0x01, 0x35}, {0x01, 0x35, 0x79}, {0x07, 0x3a}, {0x07, 0x3b}}
	keys = trie.GetKeysWithPrefix([]byte{0x0})
	require.Equal(t, expected, keys)

	expected = [][]byte{{0x07, 0x3a}, {0x07, 0x3b}}
	keys = trie.GetKeysWithPrefix([]byte{0x07, 0x30})
	require.Equal(t, expected, keys)

	expected = [][]byte{[]byte(":key1")}
	keys = trie.GetKeysWithPrefix([]byte(":key1"))
	require.Equal(t, expected, keys)

	expected = [][]byte{}
	keys = trie.GetKeysWithPrefix([]byte{0xff, 0xee, 0xbb, 0xcc, 0xbb, 0x11})
	require.Equal(t, expected, keys)
}

func TestNextKey(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x01, 0x35, 0x7a}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen"), op: PUT},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles"), op: PUT},
		{key: []byte{0xf2}, value: []byte("pho"), op: PUT},
	}

	for _, test := range tests {
		trie.Put(test.key, test.value)
	}

	testCases := []struct {
		input    []byte
		expected []byte
	}{
		{
			tests[0].key,
			tests[1].key,
		},
		{
			tests[1].key,
			tests[2].key,
		},
		{
			tests[2].key,
			tests[3].key,
		},
		{
			tests[3].key,
			tests[4].key,
		},
		{
			tests[4].key,
			tests[5].key,
		},
		{
			tests[5].key,
			nil,
		},
	}

	for _, tc := range testCases {
		next := trie.NextKey(tc.input)
		require.Equal(t, tc.expected, next)
	}
}

func TestNextKey_MoreAncestors(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79, 0xab}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79, 0xab, 0x9}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79, 0xab, 0xf}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen"), op: PUT},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles"), op: PUT},
		{key: []byte{0xf2}, value: []byte("pho"), op: PUT},
	}

	for _, test := range tests {
		trie.Put(test.key, test.value)
	}

	testCases := []struct {
		input    []byte
		expected []byte
	}{
		{
			tests[0].key,
			tests[1].key,
		},
		{
			tests[1].key,
			tests[2].key,
		},
		{
			tests[2].key,
			tests[3].key,
		},
		{
			tests[3].key,
			tests[4].key,
		},
		{
			tests[4].key,
			tests[5].key,
		},
		{
			tests[5].key,
			tests[6].key,
		},
		{
			tests[6].key,
			tests[7].key,
		},
		{
			tests[7].key,
			nil,
		},
		{
			[]byte{},
			tests[0].key,
		},
		{
			[]byte{0},
			tests[0].key,
		},
		{
			[]byte{0x01},
			tests[0].key,
		},
		{
			[]byte{0x02},
			tests[5].key,
		},
		{
			[]byte{0x05, 0x12, 0x34},
			tests[5].key,
		},
		{
			[]byte{0xf},
			tests[7].key,
		},
	}

	for _, tc := range testCases {
		next := trie.NextKey(tc.input)
		require.Equal(t, tc.expected, next, common.BytesToHex(tc.input))
	}
}

func TestNextKey_Again(t *testing.T) {
	trie := NewEmptyTrie()

	var testCases = []string{
		"asdf",
		"bnm",
		"ghjk",
		"qwerty",
		"uiopl",
		"zxcv",
	}

	for _, tc := range testCases {
		trie.Put([]byte(tc), []byte(tc))
	}

	for i, tc := range testCases {
		next := trie.NextKey([]byte(tc))
		if i == len(testCases)-1 {
			require.Nil(t, next)
		} else {
			require.Equal(t, []byte(testCases[i+1]), next, common.BytesToHex([]byte(tc)))
		}
	}
}

func TestNextKey_HostAPI(t *testing.T) {
	trie := NewEmptyTrie()

	var testCases = []string{
		":code",
		":heappages",
	}

	for _, tc := range testCases {
		trie.Put([]byte(tc), []byte(tc))
	}

	nextCases := []string{"Opti", "Option"}

	for _, tc := range nextCases {
		next := trie.NextKey([]byte(tc))
		require.Nil(t, next)
	}
}

func TestClearPrefix(t *testing.T) {
	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79, 0xab}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79, 0xab, 0x9}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen"), op: PUT},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles"), op: PUT},
		{key: []byte{0xf2}, value: []byte("pho"), op: PUT},
		{key: []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0x11}, value: []byte("asd"), op: PUT},
		{key: []byte{0xff, 0xee, 0xdd, 0xcc, 0xaa, 0x11}, value: []byte("fgh"), op: PUT},
	}

	buildTrie := func() *Trie {
		trie := NewEmptyTrie()

		for _, test := range tests {
			trie.Put(test.key, test.value)
		}

		return trie
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
		trie := buildTrie()

		// DeepCopy the trie.
		dcTrie, err := trie.DeepCopy()
		require.NoError(t, err)

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

	// DeepCopy the trie.
	dcTrie, err := trie.DeepCopy()
	require.NoError(t, err)

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
		Key:   codec.KeyLEToNibbles([]byte("other")),
		Value: []byte("other"),
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

			require.Equal(t, trieClearPrefix.MustHash(), trieDelete.MustHash(),
				fmt.Sprintf("tries not equal! prefix=0x%x\n, %s, %s", prefix, trieClearPrefix, trieDelete),
			)
		}
	}
}

func TestSnapshot(t *testing.T) {
	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79, 0xab}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79, 0xab, 0x9}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen"), op: PUT},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles"), op: PUT},
		{key: []byte{0xf2}, value: []byte("pho"), op: PUT},
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

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func TestNextKey_Random(t *testing.T) {
	for i := 0; i < 100; i++ {
		trie := NewEmptyTrie()

		// Generate random test cases.
		testCaseMap := make(map[string]struct{}) // ensure no duplicate keys
		size := 1000 + rand.Intn(10000)

		for ii := 0; ii < size; ii++ {
			str := RandStringBytes(1 + rand.Intn(20))
			if len(str) == 0 {
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
	rt := GenerateRandomTests(b, 1000000)
	trie := NewEmptyTrie()
	for i := range rt {
		test := &rt[i]
		trie.Put(test.key, test.value)
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
		case PUT:
			expectedA.Put(k, k)
		case DEL:
			expectedA.Delete(k)
		case CLEAR_PREFIX:
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
		case PUT:
			expectedB.Put(k, k)
		case DEL:
			expectedB.Delete(k)
		case CLEAR_PREFIX:
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
			case PUT:
				trieA.Put(tc.key, tc.key)
			case DEL:
				trieA.Delete(tc.key)
			case CLEAR_PREFIX:
				trieA.ClearPrefix(tc.key)
			}
		}
		wg.Done()
	}()

	go func() {
		for _, tc := range testCasesB {
			switch tc.op {
			case PUT:
				trieB.Put(tc.key, tc.key)
			case DEL:
				trieB.Delete(tc.key)
			case CLEAR_PREFIX:
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

	testFn := func(testCase []Test, prefix []byte) {
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
		t.Run("Test", func(t *testing.T) {
			for _, prefix := range prefixes {
				testFn(testCase, prefix)
			}
		})
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

				// DeepCopy the trie.
				dcTrie, err := trieClearPrefix.DeepCopy()
				require.NoError(t, err)

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
