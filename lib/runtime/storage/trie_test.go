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

package storage

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

// newTestTrieState returns an initialized TrieState
func newTestTrieState(t *testing.T, tr *trie.Trie) *TrieState {
	r := rand.Intn(1 << 16) //nolint
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(r))

	// TODO: dynamically get os.TMPDIR
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")

	cfg := &chaindb.Config{
		DataDir:  testDatadirPath,
		InMemory: true,
	}

	// TODO: don't initialize new DB but pass it in
	db, err := chaindb.NewBadgerDB(cfg)
	require.NoError(t, err)

	if tr == nil {
		tr = trie.NewEmptyTrie()
	}

	ts, err := NewTrieState(db, tr)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ts.db.Close()
		_ = os.RemoveAll(ts.db.Path())
	})

	return ts
}

func TestNewTrieState(t *testing.T) {
	testFunc := func(ts *TrieState) {
		entries := ts.t.Entries()
		iter := ts.db.NewIterator()
		dbEntries := make(map[string][]byte)

		for iter.Next() {
			key := iter.Key()
			dbEntries[string(key)] = iter.Value()
		}

		require.Equal(t, entries, dbEntries)
	}

	ts := newTestTrieState(t, nil)
	testFunc(ts)
}

var testCases = []string{
	"asdf",
	"ghjk",
	"qwerty",
	"uiopl",
	"zxcv",
	"bnm",
}

func TestTrieState_Commit(t *testing.T) {
	testFunc := func(ts *TrieState) {
		expected := make(map[string][]byte)

		for _, tc := range testCases {
			err := ts.Set([]byte(tc), []byte(tc))
			require.NoError(t, err)
			expected[tc] = []byte(tc)
		}

		require.Equal(t, expected, ts.t.Entries())
	}

	ts := newTestTrieState(t, nil)
	testFunc(ts)
}

func TestTrieState_SetGet(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			err := ts.Set([]byte(tc), []byte(tc))
			require.NoError(t, err)
		}

		err := ts.t.Store(ts.db)
		require.NoError(t, err)

		// change a trie value to simulate runtime corruption
		err = ts.t.Put([]byte(testCases[0]), []byte("noot"))
		require.NoError(t, err)

		for _, tc := range testCases {
			res, err := ts.Get([]byte(tc))
			require.NoError(t, err, fmt.Sprintf("failed to get key %s", tc))
			require.Equal(t, []byte(tc), res)
		}
	}

	ts := newTestTrieState(t, nil)
	testFunc(ts)
}

func TestTrieState_Delete(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			err := ts.Set([]byte(tc), []byte(tc))
			require.NoError(t, err)
		}

		err := ts.t.Store(ts.db)
		require.NoError(t, err)

		err = ts.Delete([]byte(testCases[0]))
		require.NoError(t, err)

		has, err := ts.Has([]byte(testCases[0]))
		require.NoError(t, err)
		require.False(t, has)
	}

	ts := newTestTrieState(t, nil)
	testFunc(ts)
}

func TestTrieState_Root(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			err := ts.Set([]byte(tc), []byte(tc))
			require.NoError(t, err)
		}

		expected := ts.MustRoot()

		// change a trie value to simulate runtime corruption
		err := ts.t.Put([]byte(testCases[0]), []byte("noot"))
		require.NoError(t, err)

		require.Equal(t, expected, ts.MustRoot())
	}

	ts := newTestTrieState(t, nil)
	testFunc(ts)
}

func TestTrieState_ClearPrefix(t *testing.T) {
	ts := newTestTrieState(t, nil)

	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	for i, key := range keys {
		err := ts.Set([]byte(key), []byte{byte(i)})
		require.NoError(t, err)
	}

	// TODO: the trie structure is incorrect after using trie.ClearPrefix. should be 1 leaf,
	// instead is 2 branches leading to leaf.
	ts.ClearPrefix([]byte("noo"))

	for i, key := range keys {
		val, err := ts.Get([]byte(key))
		require.NoError(t, err)
		if i < 2 {
			require.Nil(t, val)
		} else {
			require.NotNil(t, val)
		}
	}
}

func TestTrieState_ClearPrefixInChild(t *testing.T) {
	ts := newTestTrieState(t, nil)
	child := trie.NewEmptyTrie()

	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	for i, key := range keys {
		err := child.Put([]byte(key), []byte{byte(i)})
		require.NoError(t, err)
	}

	keyToChild := []byte("keytochild")

	err := ts.SetChild(keyToChild, child)
	require.NoError(t, err)

	err = ts.ClearPrefixInChild(keyToChild, []byte("noo"))
	require.NoError(t, err)

	for i, key := range keys {
		val, err := ts.GetChildStorage(keyToChild, []byte(key))
		require.NoError(t, err)
		if i < 2 {
			require.Nil(t, val)
		} else {
			require.NotNil(t, val)
		}
	}
}
