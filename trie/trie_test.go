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
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/polkadb"
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
	for _, test := range commonPrefixTests {
		output := lenCommonPrefix(test.a, test.b)
		if output != test.output {
			t.Errorf("Fail: got %d expected %d", output, test.output)
		}
	}
}

func newEmpty() *Trie {
	db := &Database{
		db: polkadb.NewMemDatabase(),
	}
	t := NewEmptyTrie(db)
	return t
}

func TestNewEmptyTrie(t *testing.T) {
	trie := newEmpty()
	if trie == nil {
		t.Error("did not initialize trie")
	}
}

func TestNewTrie(t *testing.T) {
	db := &Database{
		db: polkadb.NewMemDatabase(),
	}
	trie := NewTrie(db, &leaf{key: []byte{0}, value: []byte{17}})
	if trie == nil {
		t.Error("did not initialize trie")
	}
}

type randTest struct {
	key   []byte
	value []byte
}

func generateRandTest(size int) []randTest {
	rt := make([]randTest, size)
	r := *rand.New(rand.NewSource(rand.Int63()))
	for i := range rt {
		rt[i] = randTest{}
		buf := make([]byte, r.Intn(379)+1)
		r.Read(buf)
		if !keyExists(rt[0:i], buf) {
			rt[i].key = buf

			buf = make([]byte, r.Intn(128))
			r.Read(buf)
			rt[i].value = buf
		}
	}
	return rt
}

func keyExists(rt []randTest, key []byte) bool {
	for _, test := range rt {
		if bytes.Equal(test.key, key) {
			return true
		}
	}

	return false
}

func hexDecode(in string) []byte {
	out, _ := hex.DecodeString(in)
	return out
}

func writeToTestFile(s string) error {
	fp, err := filepath.Abs("./test_data")
	if err != nil {
		return err
	}
	os.Remove(fp)
	err = ioutil.WriteFile(fp, []byte(s), 0644)
	if err != nil {
		return err
	}

	return nil
}

func TestBranch(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x01, 0x35}
	value1 := []byte("spaghetti")
	key2 := []byte{0x01, 0x35, 0x79}
	value2 := []byte("gnocchi")
	key3 := []byte{0x07}
	value3 := []byte("ramen")
	key4 := []byte{0xf2}
	value4 := []byte("pho")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err := trie.Get([]byte("noot"))
	if err != nil {
		t.Errorf("Fail to get key %x: %s", "noot", err.Error())
	} else if !bytes.Equal(val, nil) {
		t.Errorf("Fail to get key %x with nil value: got %x", "noot", val)
	}

	val, err = trie.Get([]byte{0})
	if err != nil {
		t.Errorf("Fail to get key %x: %s", []byte{0}, err.Error())
	} else if !bytes.Equal(val, nil) {
		t.Errorf("Fail to get key %x with nil value: got %x", []byte{0}, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}
}

func TestBranchMore(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x01}
	value1 := []byte("spaghetti")
	key2 := []byte{0x02}
	value2 := []byte("gnocchi")
	key3 := []byte{0xf7}
	value3 := []byte("ramen")
	key4 := []byte{0x43}
	value4 := []byte("pho")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err := trie.Get([]byte{0})
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, nil) {
		t.Errorf("Fail to get key %x with nil value: got %x", "noot", val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}
}

func TestPutAndGetOddKeyLengths(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x43, 0xc1}
	value1 := []byte("noot")
	key2 := []byte{0x49, 0x29}
	value2 := []byte("nootagain")
	key3 := []byte{0x43, 0x0c}
	value3 := []byte("odd")
	key4 := []byte{0x4f, 0x4d}
	value4 := []byte("stuff")
	key5 := []byte{0xf4, 0xbc}
	value5 := []byte("spaghetti")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	val, err := trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}

	err = trie.Put(key5, value5)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	val, err = trie.Get(key5)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key5, err.Error())
	} else if !bytes.Equal(val, value5) {
		t.Errorf("Fail to get key %x with value %x: got %x", key5, value5, val)
	}
}

func TestPutAndGet(t *testing.T) {
	for i := 0; i < 50; i++ {
		trie := newEmpty()
		rt := generateRandTest(1000)
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
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)

				tests := ""
				for _, othertest := range rt {
					tests = fmt.Sprintf("%s%s\n%s\n", tests, othertest.key, othertest.value)
				}

				err := writeToTestFile(tests)
				if err != nil {
					t.Error(err)
				}
			}
		}
	}
}

func TestFailingTests(t *testing.T) {
	fp, err := filepath.Abs("./test_data")
	if err != nil {
		t.Error(err)
	}

	data, err := ioutil.ReadFile(fp)
	if err != nil {
		t.Error(err)
	}

	slicedData := strings.Split(string(data), "\n")
	tests := []randTest{}
	for i := 0; i < len(slicedData)-2; i+=2 {
		//t.Logf("key: %x val: %x\n", []byte(slicedData[i]), []byte(slicedData[i+1]))
		test := randTest{key: []byte(slicedData[i]), value: []byte(slicedData[i+1])}
		tests = append(tests, test)
	}

	trie := newEmpty()

	hasFailed := false
	passedFailingTest := false
	rt := tests
	for i, test := range rt {
		//t.Log(i)
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

			failingKey := hexDecode("4a")
			failingVal := hexDecode("3ec63d0c43de23f118898262217895bbe5f1f9d81861129ae76bdf5a6141f3a404934ce885a28fb9631040c84509b26d11bbbd441750edff1588906fb523785befa17224d1a457a41cffca6c5e")

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

func TestUpdateLeaf(t *testing.T) {
	trie := newEmpty()

	// case 1: leaf -> branch w/ two children
	rt := []randTest{
		{[]byte{0xfa}, []byte("odd")},
		{[]byte{0xfb, 0x0c}, []byte("noot")},
		{[]byte{0x0f}, []byte("nootagain")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	//trie.Print()
	trie = newEmpty()

	// case 2: leaf -> branch w/ prev leaf as value, new leaf as child
	rt = []randTest{
		{[]byte{0x0f}, []byte("nootagain")},
		{[]byte{0xfa}, []byte("odd")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	//trie.Print()
	trie = newEmpty()

	// case 3: leaf -> branch w/ new leaf as value, prev leaf as child
	rt = []randTest{
		{[]byte{0xfb}, []byte("noot")},
		{[]byte{0x0f}, []byte("nootagain")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	//trie.Print()
	trie = newEmpty()

	// case 4: replace leaf
	rt = []randTest{
		{[]byte{0xfa}, []byte("odd")},
		{[]byte{0xfb}, []byte("noot")},
		{[]byte{0xfa}, []byte("nootagain")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}
}

func TestGetPartialKey(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x01, 0x35}
	value1 := []byte("pen")
	key2 := []byte{0x01, 0x35, 0x79}
	value2 := []byte("penguin")
	key3 := []byte{0xf2}
	value3 := []byte("feather")
	key4 := []byte{0x09, 0xd3}
	value4 := []byte("noot")
	key5 := []byte{}
	value5 := []byte("floof")
	key6 := []byte{0x01, 0x35, 0x07}
	value6 := []byte("odd")

	pk0 := []byte{0x1, 0x3, 0x5}
	pk1 := []byte{0x3, 0x5}
	pk2 := []byte{}
	pk3 := []byte{0x2}
	pk4 := []byte{0x0d, 0x03}

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key5, value5)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	err = trie.Put(key6, value6)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	var val []byte
	leaf, err := trie.getLeaf(key2)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key2)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(leaf.value, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	} else if !bytes.Equal(leaf.key, pk2) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk2, leaf.key)
	}

	err = trie.Put(key6, value6)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	leaf, err = trie.getLeaf(key1)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key1)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(leaf.value, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	} else if !bytes.Equal(leaf.key, pk0) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk0, leaf.key)
	}

	leaf, err = trie.getLeaf(key2)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key2)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(leaf.value, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	} else if !bytes.Equal(leaf.key, nil) {
		t.Errorf("Fail to get correct partial key nil: got %x", leaf.key)
	}

	leaf, err = trie.getLeaf(key6)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key6)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key6, err.Error())
	} else if !bytes.Equal(leaf.value, value6) {
		t.Errorf("Fail to get key %x with value %x: got %x", key6, value6, val)
	} else if !bytes.Equal(leaf.key, nil) {
		t.Errorf("Fail to get correct partial key nil: got %x", leaf.key)
	}

	leaf, err = trie.getLeaf(key6)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key6)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key6, err.Error())
	} else if !bytes.Equal(leaf.value, value6) {
		t.Errorf("Fail to get key %x with value %x: got %x", key6, value6, val)
	} else if !bytes.Equal(leaf.key, nil) {
		t.Errorf("Fail to get correct partial key nil: got %x", leaf.key)
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err = trie.Get(key5)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key5, err.Error())
	} else if !bytes.Equal(val, value5) {
		t.Errorf("Fail to get key %x with value %x: got %x", key5, value5, val)
	}

	leaf, err = trie.getLeaf(key1)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key1)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(leaf.value, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	} else if !bytes.Equal(leaf.key, pk1) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk1, leaf.key)
	}

	leaf, err = trie.getLeaf(key2)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key2)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(leaf.value, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	} else if !bytes.Equal(leaf.key, nil) {
		t.Errorf("Fail to get correct partial key nil: got %x", leaf.key)
	}

	leaf, err = trie.getLeaf(key3)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key3)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(leaf.value, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	} else if !bytes.Equal(leaf.key, pk3) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk3, leaf.key)
	}

	leaf, err = trie.getLeaf(key4)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key4)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(leaf.value, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	} else if !bytes.Equal(leaf.key, pk4) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk4, leaf.key)
	}
}

func buildSmallTrie() *Trie {
	trie := newEmpty()

	key1 := []byte{0x01, 0x35}
	value1 := []byte("pen")
	key2 := []byte{0x01, 0x35, 0x79}
	value2 := []byte("penguin")
	key3 := []byte{0xf2}
	value3 := []byte("feather")
	key4 := []byte{0x09, 0xd3}
	value4 := []byte("noot")
	key5 := []byte{}
	value5 := []byte("floof")
	key6 := []byte{0x01, 0x35, 0x07}
	value6 := []byte("odd")

	err := trie.Put(key1, value1)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key5, value5)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	err = trie.Put(key3, value3)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	err = trie.Put(key6, value6)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key6, value6, err.Error())
	}

	return trie
}
func TestDeleteSmall(t *testing.T) {
	trie := buildSmallTrie()

	key1 := []byte{0x01, 0x35}
	value1 := []byte("pen")
	key2 := []byte{0x01, 0x35, 0x79}
	value2 := []byte("penguin")
	key3 := []byte{0xf2}
	value3 := []byte("feather")
	key4 := []byte{0x09, 0xd3}
	value4 := []byte("noot")
	key5 := []byte{}
	value5 := []byte("floof")
	key6 := []byte{0x01, 0x35, 0x07}
	value6 := []byte("odd")

	// key5 = nil
	err := trie.Delete(key5)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key5, err.Error())
	}

	val, err := trie.Get(key5)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key5, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key5, value5, val)
	}

	trie = buildSmallTrie()

	// key4 = 09d3
	err = trie.Delete(key4)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key4, err.Error())
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key4, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key4, value4, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(value2, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(value1, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	trie = buildSmallTrie()

	// key3 = f2
	err = trie.Delete(key3)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key3, err.Error())
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key3, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key3, value3, val)
	}

	trie = buildSmallTrie()

	// key5 = nil
	err = trie.Delete(key5)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key5, err.Error())
	}

	err = trie.Delete(key3)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key3, err.Error())
	}

	val, err = trie.Get(key5)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key5, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key5, value5, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(value2, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(value1, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	trie = buildSmallTrie()

	// key2 = 013579
	err = trie.Delete(key2)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key2, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key2, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(value1, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	trie = buildSmallTrie()

	// key2 = 0135
	err = trie.Delete(key1)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key1, err.Error())
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key1, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(value2, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	trie = buildSmallTrie()

	// key6 = 0135
	err = trie.Delete(key6)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key6, err.Error())
	}

	val, err = trie.Get(key6)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key6, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key6, value6, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(value2, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(value1, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}
}

func TestCombineBranch(t *testing.T) {
	trie := buildSmallTrie()

	key4 := []byte{0x09, 0xd3}
	value4 := []byte("noot")
	key6 := []byte{0x01, 0x35, 0x46}
	value6 := []byte("raccoon")
	key7 := []byte{0x01, 0x35, 0x46, 0x77}
	value7 := []byte("rat")


	err := trie.Put(key6, value6)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key6, value6, err.Error())
	}

	err = trie.Put(key7, value7)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key7, value7, err.Error())
	}

	err = trie.Delete(key4)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key4, err.Error())
		t.Errorf("Fail to delete key %x: %s", key4, err.Error())
	}

	val, err := trie.Get(key4)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key4, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key4, value4, val)
	}

	}

func TestDeleteOddKeyLengths(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x43, 0xc1}
	value1 := []byte("noot")
	key2 := []byte{0x49, 0x29}
	value2 := []byte("nootagain")
	key3 := []byte{0x43, 0x0c}
	value3 := []byte("odd")
	key4 := []byte{0x4f, 0x4d}
	value4 := []byte("stuff")
	key5 := []byte{0xf4, 0xbc}
	value5 := []byte("spaghetti")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	val, err := trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}

	err = trie.Put(key5, value5)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	val, err = trie.Get(key5)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key5, err.Error())
	} else if !bytes.Equal(val, value5) {
		t.Errorf("Fail to get key %x with value %x: got %x", key5, value5, val)
	}
	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}
}

func TestDelete(t *testing.T) {
	trie := newEmpty()

	rt := generateRandTest(1000)
	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	for _, test := range rt {
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
	}
}

func TestDeleteFromBranch(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x06, 0x15, 0xfc}
	value1 := []byte("noot")
	key2 := []byte{0x06, 0x2b, 0xa9}
	value2 := []byte("nootagain")
	key3 := []byte{0x06, 0xaf, 0xb1}
	value3 := []byte("odd")
	key4 := []byte{0x06, 0xa3, 0xff}
	value4 := []byte("stuff")
	key5 := []byte{0x43, 0x21}
	value5 := []byte("stuffagain")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	err = trie.Put(key5, value5)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	val, err := trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	err = trie.Delete(key1)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key1, err.Error())
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key1, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	err = trie.Delete(key3)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key3, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}

	err = trie.Delete(key4)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key4, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}
}