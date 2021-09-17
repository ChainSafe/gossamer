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
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/require"
)

func TestGenerateProofWithRecorder(t *testing.T) {
	tmp, err := ioutil.TempDir("", "*-test-trie")
	require.NoError(t, err)

	memdb, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmp,
	})
	require.NoError(t, err)

	trie, entries := RandomTrieTest(t, 200)
	err = trie.Store(memdb)
	require.NoError(t, err)

	var otherKey *KV
	var lastEntryKey *KV

	i := 0
	for _, kv := range entries {
		if len(entries)-2 == i {
			otherKey = kv
		}
		lastEntryKey = kv
		i++
	}

	fmt.Printf("Test\n\tkey:0x%x\n\tvalue:0x%x\n", lastEntryKey.K, lastEntryKey.V)
	fmt.Printf("Test2\n\tkey:0x%x\n\tvalue2:0x%x\n", otherKey.K, otherKey.V)

	rootHash := trie.root.getHash()
	proof, err := GenerateProof(rootHash, [][]byte{lastEntryKey.K, otherKey.K}, memdb)
	require.NoError(t, err)

	fmt.Printf("\n\n")
	for _, p := range proof {
		fmt.Printf("generated -> 0x%x\n", p)
	}
}
