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
	"math/rand"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestVerifyProof(t *testing.T) {
	trie, entries := RandomTrieTest(t, 200)
	root, err := trie.Hash()
	require.NoError(t, err)

	for _, entry := range entries {
		proof, err := trie.GenerateProof([][]byte{entry.K})
		require.NoError(t, err)

		v, err := VerifyProof(root, entry.K, proof)
		require.NoError(t, err)
		require.True(t, v)
	}
}

func TestVerifyProofOneElement(t *testing.T) {
	trie := NewEmptyTrie()
	key := randBytes(32)
	trie.Put(key, []byte("V"))

	rootHash, err := trie.Hash()
	require.NoError(t, err)

	proof, err := trie.GenerateProof([][]byte{key})
	fmt.Println(proof)
	require.NoError(t, err)

	val, err := VerifyProof(rootHash, key, proof)
	require.NoError(t, err)

	require.True(t, val)
}

func TestVerifyProof_BadProof(t *testing.T) {
	trie, entries := RandomTrieTest(t, 200)
	rootHash, err := trie.Hash()
	require.NoError(t, err)

	for _, entry := range entries {
		proof, err := trie.GenerateProof([][]byte{entry.K})
		require.Greater(t, len(proof), 0)
		require.NoError(t, err)

		i := 0
		d := rand.Intn(len(proof))

		var toTamper string
		for k := range proof {
			if i < d {
				i++
				continue
			}

			toTamper = k
			break
		}

		val := proof[toTamper]
		delete(proof, toTamper)

		newhash, err := common.Keccak256(val)
		require.NoError(t, err)
		proof[common.BytesToHex(newhash.ToBytes())] = val

		v, err := VerifyProof(rootHash, entry.K, proof)
		require.NoError(t, err)
		require.False(t, v)
	}
}

func TestGenerateProofMissingKey(t *testing.T) {
	trie := NewEmptyTrie()

	parentKey, parentVal := randBytes(32), randBytes(20)
	chieldKey, chieldValue := modifyLastBytes(parentKey), modifyLastBytes(parentVal)
	gransonKey, gransonValue := modifyLastBytes(chieldKey), modifyLastBytes(chieldValue)

	trie.Put(parentKey, parentVal)
	trie.Put(chieldKey, chieldValue)
	trie.Put(gransonKey, gransonValue)

	searchfor := make([]byte, len(gransonKey))
	copy(searchfor[:], gransonKey[:])

	// keep the path til the key but modify the last element
	searchfor[len(searchfor)-1] = searchfor[len(searchfor)-1] + byte(0xff)

	_, err := trie.GenerateProof([][]byte{searchfor})
	require.Error(t, err, "leaf node doest not match the key")
}

func TestGenerateProofNoMorePathToFollow(t *testing.T) {
	trie := NewEmptyTrie()

	parentKey, parentVal := randBytes(32), randBytes(20)
	chieldKey, chieldValue := modifyLastBytes(parentKey), modifyLastBytes(parentVal)
	gransonKey, gransonValue := modifyLastBytes(chieldKey), modifyLastBytes(chieldValue)

	trie.Put(parentKey, parentVal)
	trie.Put(chieldKey, chieldValue)
	trie.Put(gransonKey, gransonValue)

	searchfor := make([]byte, len(parentKey))
	copy(searchfor[:], parentKey[:])

	// the keys are equals until the byte number 20 so we modify the byte number 20 to another
	// value and the branch node will no be able to found the right slot
	searchfor[20] = searchfor[20] + byte(0xff)

	_, err := trie.GenerateProof([][]byte{searchfor})
	require.Error(t, err, "no more paths to follow")
}

func modifyLastBytes(b []byte) []byte {
	newB := make([]byte, len(b))
	copy(newB[:], b)

	rb := randBytes(12)
	copy(newB[20:], rb)

	return newB
}
