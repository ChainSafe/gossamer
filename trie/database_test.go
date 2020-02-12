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
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/db"
)

func newTrie() (*Trie, error) {
	hasher, err := NewHasher()
	if err != nil {
		return nil, err
	}

	stateDB, err := db.NewBadgerDB("./test_data/state")
	if err != nil {
		return nil, err
	}

	trie := &Trie{
		db: &Database{
			DB:     stateDB,
			Hasher: hasher,
		},
		root: nil,
	}

	trie.db.Batch = trie.db.DB.NewBatch()

	return trie, nil
}

func (t *Trie) closeDb() {
	t.db.DB.Close()
	if err := os.RemoveAll("./test_data"); err != nil {
		fmt.Println("removal of temp directory gossamer_data failed")
	}
}

func TestStoreAndLoadFromDB(t *testing.T) {
	trie, err := newTrie()
	if err != nil {
		t.Fatal(err)
	}

	defer trie.closeDb()

	rt := generateRandomTests(1000)
	var val []byte
	for _, test := range rt {
		err = trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}

		val, err = trie.Get(test.key)
		if err != nil {
			t.Errorf("Fail to get key %x: %s", test.key, err.Error())
		} else if !bytes.Equal(val, test.value) {
			t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
		}
	}

	err = trie.StoreInDB()
	if err != nil {
		t.Fatalf("Fail: could not write trie to DB: %s", err)
	}

	encroot, err := trie.Hash()
	if err != nil {
		t.Fatal(err)
	}

	expected := &Trie{root: trie.root}

	trie.root = nil
	err = trie.LoadFromDB(encroot)
	if err != nil {
		t.Errorf("Fail: could not load trie from DB: %s", err)
	}

	if strings.Compare(expected.String(), trie.String()) != 0 {
		t.Errorf("Fail: got\n %s expected\n %s", expected.String(), trie.String())
	}

	if !reflect.DeepEqual(expected.root, trie.root) {
		t.Errorf("Fail: got\n %s expected\n %s", expected.String(), trie.String())
	}
}

func TestEncodeAndDecodeFromDB(t *testing.T) {
	trie := &Trie{}

	tests := []trieTest{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x3}, value: []byte("f")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
		{key: []byte{0}, value: nil},
	}

	for _, test := range tests {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Fatal(err)
		}
	}

	enc, err := trie.encode()
	if err != nil {
		t.Fatal(err)
	}

	testTrie := &Trie{}
	err = testTrie.decode(enc)
	if err != nil {
		testTrie.Print()
		t.Fatal(err)
	}

	if strings.Compare(testTrie.String(), trie.String()) != 0 {
		t.Errorf("Fail: got\n %s expected\n %s", testTrie.String(), trie.String())
	}

	if !reflect.DeepEqual(testTrie.root, trie.root) {
		t.Errorf("Fail: got\n %s expected\n %s", testTrie.String(), trie.String())
	}
}

func TestStoreAndLoadHash(t *testing.T) {
	trie, err := newTrie()
	if err != nil {
		t.Fatal(err)
	}

	defer trie.closeDb()

	tests := []trieTest{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x3}, value: []byte("f")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
		{key: []byte{0}, value: nil},
	}

	for _, test := range tests {
		err = trie.Put(test.key, test.value)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = trie.StoreHash()
	if err != nil {
		t.Fatal(err)
	}

	hash, err := trie.LoadHash()
	if err != nil {
		t.Fatal(err)
	}

	expected, err := trie.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if hash != expected {
		t.Fatalf("Fail: got %x expected %x", hash, expected)
	}
}

func TestStoreAndLoadGenesisData(t *testing.T) {
	trie, err := newTrie()
	if err != nil {
		t.Fatal(err)
	}

	defer trie.closeDb()

	bootnodes := common.StringArrayToBytes([]string{
		"/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu",
		"/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu",
	})

	expected := &genesis.Data{
		Name:       "gossamer",
		ID:         "gossamer",
		Bootnodes:  bootnodes,
		ProtocolID: "/gossamer/test/0",
	}

	err = trie.db.StoreGenesisData(expected)
	if err != nil {
		t.Fatal(err)
	}

	gen, err := trie.db.LoadGenesisData()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gen, expected) {
		t.Fatalf("Fail: got %v expected %v", gen, expected)
	}
}
