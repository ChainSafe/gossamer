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
	"bytes"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

// newTestTrieState returns an initialized TrieState
func newTestTrieState(t *testing.T) *TrieState {
	ts, err := NewTrieState(nil)
	require.NoError(t, err)
	return ts
}

var testCases = []string{
	"asdf",
	"ghjk",
	"qwerty",
	"uiopl",
	"zxcv",
	"bnm",
}

func TestTrieState_SetGet(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Set([]byte(tc), []byte(tc))
		}

		for _, tc := range testCases {
			res := ts.Get([]byte(tc))
			require.Equal(t, []byte(tc), res)
		}
	}

	ts := newTestTrieState(t)
	testFunc(ts)
}

func TestTrieState_Delete(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Set([]byte(tc), []byte(tc))
		}

		ts.Delete([]byte(testCases[0]))
		has := ts.Has([]byte(testCases[0]))
		require.False(t, has)
	}

	ts := newTestTrieState(t)
	testFunc(ts)
}

func TestTrieState_Root(t *testing.T) {
	testFunc := func(ts *TrieState) {
		for _, tc := range testCases {
			ts.Set([]byte(tc), []byte(tc))
		}

		expected := ts.MustRoot()
		require.Equal(t, expected, ts.MustRoot())
	}

	ts := newTestTrieState(t)
	testFunc(ts)
}

func TestTrieState_ClearPrefix(t *testing.T) {
	ts := newTestTrieState(t)

	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	for i, key := range keys {
		ts.Set([]byte(key), []byte{byte(i)})
	}

	ts.ClearPrefix([]byte("noo"))

	for i, key := range keys {
		val := ts.Get([]byte(key))
		if i < 2 {
			require.Nil(t, val)
		} else {
			require.NotNil(t, val)
		}
	}
}

func TestTrieState_ClearPrefixInChild(t *testing.T) {
	ts := newTestTrieState(t)
	child := trie.NewEmptyTrie()

	keys := []string{
		"noot",
		"noodle",
		"other",
	}

	for i, key := range keys {
		child.Put([]byte(key), []byte{byte(i)})
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

func TestTrieState_NextKey(t *testing.T) {
	ts := newTestTrieState(t)

	for _, tc := range testCases {
		ts.Set([]byte(tc), []byte(tc))
	}

	sort.Slice(testCases, func(i, j int) bool {
		return bytes.Compare([]byte(testCases[i]), []byte(testCases[j])) == -1
	})

	for i, tc := range testCases {
		next := ts.NextKey([]byte(tc))
		if i == len(testCases)-1 {
			require.Nil(t, next)
		} else {
			require.Equal(t, []byte(testCases[i+1]), next, common.BytesToHex([]byte(tc)))
		}
	}
}

func TestTrieState_CommitStorageTransaction(t *testing.T) {
	ts := newTestTrieState(t)

	for _, tc := range testCases {
		ts.Set([]byte(tc), []byte(tc))
	}

	ts.BeginStorageTransaction()
	testValue := []byte("noot")
	ts.Set([]byte(testCases[0]), testValue)
	ts.CommitStorageTransaction()

	val := ts.Get([]byte(testCases[0]))
	require.Equal(t, testValue, val)
}

func TestTrieState_RollbackStorageTransaction(t *testing.T) {
	ts := newTestTrieState(t)

	for _, tc := range testCases {
		ts.Set([]byte(tc), []byte(tc))
	}

	ts.BeginStorageTransaction()
	testValue := []byte("noot")
	ts.Set([]byte(testCases[0]), testValue)
	ts.RollbackStorageTransaction()

	val := ts.Get([]byte(testCases[0]))
	require.Equal(t, []byte(testCases[0]), val)
}
