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

func TestProofGeneration(t *testing.T) {
	tmp, err := ioutil.TempDir("", "*-test-trie")
	require.NoError(t, err)

	memdb, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmp,
	})
	require.NoError(t, err)

	expectedValue := rand32Bytes()

	trie := NewEmptyTrie()
	trie.Put([]byte("cat"), rand32Bytes())
	trie.Put([]byte("catapulta"), rand32Bytes())
	trie.Put([]byte("catapora"), expectedValue)
	trie.Put([]byte("dog"), rand32Bytes())
	trie.Put([]byte("doguinho"), rand32Bytes())

	err = trie.Store(memdb)
	require.NoError(t, err)

	hash, err := trie.Hash()
	require.NoError(t, err)

	proof, err := GenerateProof(hash.ToBytes(), [][]byte{[]byte("catapulta"), []byte("catapora")}, memdb)
	require.NoError(t, err)

	// TODO: use the verify_proof function to assert the tests
	require.Equal(t, 5, len(proof))

	pl := []Pair{
		{Key: []byte("catapora"), Value: expectedValue},
	}

	v, err := VerifyProof(proof, hash.ToBytes(), pl)
	require.True(t, v)
	require.NoError(t, err)
}

func testGenerateProof(t *testing.T, entries []Pair, keys [][]byte) ([]byte, [][]byte, []Pair) {
	t.Helper()

	tmp, err := ioutil.TempDir("", "*-test-trie")
	require.NoError(t, err)

	memdb, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmp,
	})
	require.NoError(t, err)

	trie := NewEmptyTrie()
	for _, e := range entries {
		trie.Put(e.Key, e.Value)
	}

	err = trie.Store(memdb)
	require.NoError(t, err)

	root := trie.root.getHash()
	proof, err := GenerateProof(root, keys, memdb)
	require.NoError(t, err)

	items := make([]Pair, 0)
	for _, i := range keys {
		value := trie.Get(i)
		require.NotNil(t, value)

		itemFromDB := Pair{
			Key:   i,
			Value: value,
		}
		items = append(items, itemFromDB)
	}

	return root, proof, items
}

func TestVerifyProofCorrectly(t *testing.T) {
	entries := []Pair{
		{Key: []byte("alpha"), Value: make([]byte, 32)},
		{Key: []byte("bravo"), Value: []byte("bravo")},
		{Key: []byte("do"), Value: []byte("verb")},
		{Key: []byte("dog"), Value: []byte("puppy")},
		{Key: []byte("doge"), Value: make([]byte, 32)},
		{Key: []byte("horse"), Value: []byte("stallion")},
		{Key: []byte("house"), Value: []byte("building")},
	}

	keys := [][]byte{
		[]byte("do"),
		[]byte("dog"),
		[]byte("doge"),
	}

	root, proof, _ := testGenerateProof(t, entries, keys)

	fmt.Printf("ROOT: 0x%x\n", root)
	fmt.Println("PROOF")
	for _, p := range proof {
		fmt.Printf("0x%x\n", p)
	}

	pl := []Pair{
		{Key: []byte("do"), Value: []byte("verb")},
		{Key: []byte("dog"), Value: []byte("puppy")},
		{Key: []byte("doge"), Value: make([]byte, 32)},
	}

	v, err := VerifyProof(proof, root, pl)
	require.True(t, v)
	require.NoError(t, err)
}
