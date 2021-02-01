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
	"io/ioutil"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) chaindb.Database {
	// TODO: dynamically get os.TMPDIR
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")

	cfg := &chaindb.Config{
		DataDir:  testDatadirPath,
		InMemory: true,
	}

	// TODO: don't initialize new DB but pass it in
	db, err := chaindb.NewBadgerDB(cfg)
	require.NoError(t, err)
	return chaindb.NewTable(db, "trie")
}

func TestTrie_DatabaseStoreAndLoad(t *testing.T) {
	trie := &Trie{}

	cases := [][]Test{
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
			{key: []byte{0}, value: nil},
		},
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x30}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
	}

	for _, testCase := range cases {
		for _, test := range testCase {
			err := trie.Put(test.key, test.value)
			require.NoError(t, err)
		}

		db := newTestDB(t)
		err := trie.Store(db)
		require.NoError(t, err)

		res := NewEmptyTrie()
		err = res.Load(db, trie.MustHash())
		require.NoError(t, err)
		require.Equal(t, trie.MustHash(), res.MustHash())
	}
}

func TestTrie_WriteDirty_Put(t *testing.T) {
	trie := &Trie{}

	cases := [][]Test{
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
			{key: []byte{0}, value: nil},
		},
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x30}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
	}

	for _, testCase := range cases {
		for _, test := range testCase {
			err := trie.Put(test.key, test.value)
			require.NoError(t, err)
		}

		db := newTestDB(t)
		err := trie.Store(db)
		require.NoError(t, err)

		err = trie.PutInDB(db, []byte{0x01, 0x35, 0x79}, []byte("notapenguin"))
		require.NoError(t, err)

		res := NewEmptyTrie()
		err = res.Load(db, trie.MustHash())
		require.NoError(t, err)
		require.Equal(t, trie.MustHash(), res.MustHash())
	}
}

func TestTrie_WriteDirty_Delete(t *testing.T) {
	trie := &Trie{}

	cases := [][]Test{
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
			{key: []byte{0}, value: nil},
		},
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x30}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
	}

	for _, testCase := range cases {
		for _, test := range testCase {
			err := trie.Put(test.key, test.value)
			require.NoError(t, err)
		}

		db := newTestDB(t)
		err := trie.Store(db)
		require.NoError(t, err)

		err = trie.DeleteFromDB(db, []byte{0x01, 0x35, 0x79})
		require.NoError(t, err)

		res := NewEmptyTrie()
		err = res.Load(db, trie.MustHash())
		require.NoError(t, err)
		require.Equal(t, trie.MustHash(), res.MustHash())
	}
}

func TestTrie_WriteDirty_ClearPrefix(t *testing.T) {
	trie := &Trie{}

	cases := [][]Test{
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
			{key: []byte{0}, value: nil},
		},
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x30}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
	}

	for _, testCase := range cases {
		for _, test := range testCase {
			err := trie.Put(test.key, test.value)
			require.NoError(t, err)
		}

		db := newTestDB(t)
		err := trie.Store(db)
		require.NoError(t, err)

		err = trie.ClearPrefixFromDB(db, []byte{0x01, 0x35})
		require.NoError(t, err)

		res := NewEmptyTrie()
		err = res.Load(db, trie.MustHash())
		require.NoError(t, err)

		require.Equal(t, trie.MustHash(), res.MustHash())
	}
}

func TestGetFromDB(t *testing.T) {
	trie := &Trie{}

	cases := [][]Test{
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x3}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
			{key: []byte{0}, value: nil},
		},
		[]Test{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x70}, value: []byte("g")},
			{key: []byte{0xf2}, value: []byte("feather")},
			{key: []byte{0xf2, 0x30}, value: []byte("f")},
			{key: []byte{0x09, 0xd3}, value: []byte("noot")},
			{key: []byte{0x07}, value: []byte("ramen")},
		},
	}

	for _, testCase := range cases {
		for _, test := range testCase {
			err := trie.Put(test.key, test.value)
			require.NoError(t, err)
		}

		db := newTestDB(t)
		err := trie.Store(db)
		require.NoError(t, err)

		root := trie.MustHash()

		val, err := GetFromDB(db, root, []byte{0x01, 0x35, 0x79})
		require.NoError(t, err)
		require.Equal(t, []byte("penguin"), val)
	}
}
