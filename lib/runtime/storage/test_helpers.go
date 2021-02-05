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
	// "encoding/binary"
	// "io/ioutil"
	// "math/rand"
	// "os"
	"testing"

	//"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

// NewTestTrieState returns an initialized TrieState
func NewTestTrieState(t *testing.T, tr *trie.Trie) *TrieState {
	// r := rand.Intn(1 << 16) //nolint
	// buf := make([]byte, 2)
	// binary.LittleEndian.PutUint16(buf, uint16(r))

	// testDatadirPath, _ := ioutil.TempDir(os.TempDir(), "test-datadir-*")

	// cfg := &chaindb.Config{
	// 	DataDir:  testDatadirPath,
	// 	InMemory: true,
	// }

	// db, err := chaindb.NewBadgerDB(cfg)
	// require.NoError(t, err)

	if tr == nil {
		tr = trie.NewEmptyTrie()
	}

	// err = tr.WriteDirty(db)
	// require.NoError(t, err)

	ts, err := NewTrieState(nil, tr)
	require.NoError(t, err)

	// t.Cleanup(func() {
	// 	_ = ts.db.Close()
	// 	_ = os.RemoveAll(ts.db.Path())
	// })

	return ts
}
