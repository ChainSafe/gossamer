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

	//"github.com/ChainSafe/gossamer/lib/common"

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
	return db
}

func TestTrie_DatabaseStoreAndLoad(t *testing.T) {
	trie := &Trie{}

	tests := []Test{
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
		require.NoError(t, err)
	}

	db := newTestDB(t)
	err := trie.Store(db)
	require.NoError(t, err)

	res := NewEmptyTrie()
	err = res.Load(db, trie.MustHash())
	require.NoError(t, err)
}

func TestTrie_WriteDirty(t *testing.T) {
	trie := &Trie{}

	tests := []Test{
		{key: []byte{0x01, 0x35}, value: []byte("pen")},
		{key: []byte{0x01, 0x35, 0x79}, value: []byte("penguin")},
		{key: []byte{0x01, 0x35, 0x07}, value: []byte("ggg")},
		{key: []byte{0xf2}, value: []byte("feather")},
		{key: []byte{0xf2, 0x3}, value: []byte("fff")},
		{key: []byte{0x09, 0xd3}, value: []byte("noot")},
		{key: []byte{0x07}, value: []byte("ramen")},
		{key: []byte{0}, value: nil},
	}

	for _, test := range tests {
		err := trie.Put(test.key, test.value)
		require.NoError(t, err)
	}

	db := newTestDB(t)
	err := trie.Store(db)
	require.NoError(t, err)

	t.Log(trie)

	err = trie.Put([]byte{0x01, 0x35, 0x79}, []byte("notapenguin"))
	require.NoError(t, err)

	t.Log(trie)

}
