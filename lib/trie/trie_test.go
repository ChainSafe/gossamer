// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package trie

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/require"
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

var (
	PUT     = 0
	GET     = 1
	DEL     = 2
	GETLEAF = 3
)

func TestNewEmptyTrie(t *testing.T) {
	trie := NewEmptyTrie()
	if trie == nil {
		t.Error("did not initialize trie")
	}
}

func TestNewTrie(t *testing.T) {
	trie := NewTrie(&leaf{key: []byte{0}, value: []byte{17}})
	if trie == nil {
		t.Error("did not initialize trie")
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
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Fatal(err)
		}
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
	err = ioutil.WriteFile(fp, []byte(testString), 0644)
	if err != nil {
		return err
	}

	return nil
}

func buildSmallTrie(t *testing.T) *Trie {
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
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Fatal(err)
		}
	}

	return trie
}

func runTests(t *testing.T, trie *Trie, tests []Test) {
	for i, test := range tests {
		test := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if test.op == PUT {
				err := trie.Put(test.key, test.value)
				if err != nil {
					t.Errorf("Fail to put key %x with value %x: %s", test.key, test.value, err)
				}
			} else if test.op == GET {
				val, err := trie.Get(test.key)
				if err != nil {
					t.Errorf("Error when attempting to get key %x: %s", test.key, err.Error())
				} else if !bytes.Equal(val, test.value) {
					t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
				}
			} else if test.op == DEL {
				err := trie.Delete(test.key)
				if err != nil {
					t.Errorf("Fail to delete key %x: %s", test.key, err.Error())
				}
			} else if test.op == GETLEAF {
				leaf, err := trie.getLeaf(test.key)
				if leaf == nil {
					t.Errorf("Fail to get key %x: nil leaf", test.key)
				} else if err != nil {
					t.Errorf("Fail to get key %x: %s", test.key, err.Error())
				} else if !bytes.Equal(leaf.value, test.value) {
					t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, leaf.value)
				} else if !bytes.Equal(leaf.key, test.pk) {
					t.Errorf("Fail to get correct partial key %x with key %x: got %x", test.pk, test.key, leaf.key)
				}
			}
		})
	}
}

func TestLoadTrie(t *testing.T) {
	data := map[string]string{"0x1234": "0x5678", "0xaabbcc": "0xddeeff"}
	testTrie := &Trie{}

	err := testTrie.Load(data)
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
		err = expectedTrie.Put(keyBytes, valueBytes)
		if err != nil {
			t.Fatal(err)
		}
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
			err := trie.Put(test.key, test.value)
			if err != nil {
				t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
			}

			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}
		}

		for _, test := range rt {
			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
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

	data, err := ioutil.ReadFile(fp)
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
			err := trie.Put(test.key, test.value)
			if err != nil {
				t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
			}

			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}

			failingKey := hexDecode("")
			failingVal := hexDecode("")

			if bytes.Equal(test.key, failingKey) {
				passedFailingTest = true
			}

			val, err = trie.Get(failingKey)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", failingKey, err.Error())
			} else if !bytes.Equal(val, failingVal) && !hasFailed && passedFailingTest {
				t.Errorf("Fail to get key %x with value %x: got %x", failingKey, failingVal, val)
				t.Logf("test failed at insertion of key %x index %d", test.key, i)
				hasFailed = true
			}
		}
	}

	for _, test := range rt {
		if len(test.key) != 0 {
			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
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
	trie := buildSmallTrie(t)

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
	trie := buildSmallTrie(t)

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

func TestDelete(t *testing.T) {
	trie := NewEmptyTrie()

	rt := GenerateRandomTests(t, 100)
	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	for i, test := range rt {
		test := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := rand.Int() % 2
			switch r {
			case 0:
				err := trie.Delete(test.key)
				if err != nil {
					t.Errorf("Fail to delete key %x: %s", test.key, err.Error())
				}

				val, err := trie.Get(test.key)
				if err != nil {
					t.Errorf("Error when attempting to get deleted key %x: %s", test.key, err.Error())
				} else if val != nil {
					t.Errorf("Fail to delete key %x with value %x: got %x", test.key, test.value, val)
				}
			case 1:
				val, err := trie.Get(test.key)
				if err != nil {
					t.Errorf("Error when attempting to get key %x: %s", test.key, err.Error())
				} else if !bytes.Equal(test.value, val) {
					t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
				}
			}
		})
	}
}

func TestGetKeysWithPrefix(t *testing.T) {
	trie := NewEmptyTrie()

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("spaghetti"), op: PUT},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("gnocchi"), op: PUT},
		{key: []byte{0x07, 0x3a}, value: []byte("ramen"), op: PUT},
		{key: []byte{0x07, 0x3b}, value: []byte("noodles"), op: PUT},
		{key: []byte{0xf2}, value: []byte("pho"), op: PUT},
	}

	for _, test := range tests {
		trie.Put(test.key, test.value)
	}

	expected := [][]byte{{0x01, 0x35}, {0x01, 0x35, 0x79}}
	keys := trie.GetKeysWithPrefix([]byte{0x01})
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("Fail: got %v expected %v", keys, expected)
	}

	expected = [][]byte{{0x01, 0x35}, {0x01, 0x35, 0x79}, {0x07, 0x3a}, {0x07, 0x3b}}
	keys = trie.GetKeysWithPrefix([]byte{0x0})
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("Fail: got %v expected %v", keys, expected)
	}

	expected = [][]byte{{0x07, 0x3a}, {0x07, 0x3b}}
	keys = trie.GetKeysWithPrefix([]byte{0x07, 0x30})
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("Fail: got %v expected %v", keys, expected)
	}
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
			nil,
		},
	}

	for _, tc := range testCases {
		next := trie.NextKey(tc.input)
		require.Equal(t, tc.expected, next)
	}
}
