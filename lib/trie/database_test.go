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
	cases := [][]Test{
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
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
		trie := NewEmptyTrie()

		for _, test := range testCase {
			trie.Put(test.key, test.value)
		}

		db := newTestDB(t)
		err := trie.Store(db)
		require.NoError(t, err)

		res := NewEmptyTrie()
		err = res.Load(db, trie.MustHash())
		require.NoError(t, err)
		require.Equal(t, trie.MustHash(), res.MustHash())

		for _, test := range testCase {
			val, err := GetFromDB(db, trie.MustHash(), test.key)
			require.NoError(t, err)
			require.Equal(t, test.value, val)
		}
	}
}

func TestTrie_WriteDirty_Put(t *testing.T) {
	cases := [][]Test{
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
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
		trie := NewEmptyTrie()
		db := newTestDB(t)

		for i, test := range testCase {
			trie.Put(test.key, test.value)
			err := trie.WriteDirty(db)
			require.NoError(t, err)

			for j, kv := range testCase {
				if j > i {
					break
				}

				val, err := GetFromDB(db, trie.MustHash(), kv.key)
				require.NoError(t, err)
				require.Equal(t, kv.value, val, fmt.Sprintf("key=%x", kv.key))
			}
		}

		err := trie.Store(db)
		require.NoError(t, err)

		trie.Put([]byte("asdf"), []byte("notapenguin"))
		err = trie.WriteDirty(db)
		require.NoError(t, err)

		res := NewEmptyTrie()
		err = res.Load(db, trie.MustHash())
		require.NoError(t, err)
		require.Equal(t, trie.MustHash(), res.MustHash())

		for _, test := range testCase {
			val, err := GetFromDB(db, trie.MustHash(), test.key) //nolint
			require.NoError(t, err)
			if bytes.Equal(test.key, []byte("asdf")) {
				continue
			}
			require.Equal(t, test.value, val)
		}

		val, err := GetFromDB(db, trie.MustHash(), []byte("asdf"))
		require.NoError(t, err)
		require.Equal(t, []byte("notapenguin"), val)
	}
}

func TestTrie_WriteDirty_PutReplace(t *testing.T) {
	cases := [][]Test{
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
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
		trie := NewEmptyTrie()
		db := newTestDB(t)

		for _, test := range testCase {
			trie.Put(test.key, test.value)

			err := trie.WriteDirty(db)
			require.NoError(t, err)
		}

		for _, test := range testCase {
			// overwrite existing values
			trie.Put(test.key, test.key)

			err := trie.WriteDirty(db)
			require.NoError(t, err)
		}

		res := NewEmptyTrie()
		err := res.Load(db, trie.MustHash())
		require.NoError(t, err)
		require.Equal(t, trie.MustHash(), res.MustHash())

		for _, test := range testCase {
			val, err := GetFromDB(db, trie.MustHash(), test.key)
			require.NoError(t, err)
			require.Equal(t, test.key, val)
		}
	}
}

func TestTrie_WriteDirty_Delete(t *testing.T) {
	cases := [][]Test{
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
			{key: []byte{0x01, 0x35, 0x99}, value: []byte("g")},
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
		for _, curr := range testCase {
			trie := NewEmptyTrie()

			for _, test := range testCase {
				trie.Put(test.key, test.value)
			}

			db := newTestDB(t)
			err := trie.Store(db)
			require.NoError(t, err)

			err = trie.DeleteFromDB(db, curr.key)
			require.NoError(t, err)

			res := NewEmptyTrie()
			err = res.Load(db, trie.MustHash())
			require.NoError(t, err)
			require.Equal(t, trie.MustHash(), res.MustHash())

			for _, kv := range testCase {
				val, err := GetFromDB(db, trie.MustHash(), kv.key)
				require.NoError(t, err)

				if bytes.Equal(kv.key, curr.key) {
					require.Nil(t, val, fmt.Sprintf("key=%x", kv.key))
					continue
				}

				require.Equal(t, kv.value, val)
			}
		}
	}
}

func TestTrie_WriteDirty_ClearPrefix(t *testing.T) {
	cases := [][]Test{
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
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
		trie := NewEmptyTrie()

		for _, test := range testCase {
			trie.Put(test.key, test.value)
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

func TestTrie_GetFromDB(t *testing.T) {
	cases := [][]Test{
		{
			{key: []byte{0x01, 0x35}, value: []byte("pen")},
			{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
			{key: []byte{0x01, 0x35, 0x7}, value: []byte("g")},
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
		trie := NewEmptyTrie()

		for _, test := range testCase {
			trie.Put(test.key, test.value)
		}

		db := newTestDB(t)
		err := trie.Store(db)
		require.NoError(t, err)

		root := trie.MustHash()

		for _, test := range testCase {
			val, err := GetFromDB(db, root, test.key)
			require.NoError(t, err)
			require.Equal(t, test.value, val)
		}
	}
}
