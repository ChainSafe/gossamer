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
	crand "crypto/rand"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func inMemoryChainDB(t *testing.T) (*chaindb.BadgerDB, func()) {
	t.Helper()

	tmpdir, err := ioutil.TempDir("", "trie-chaindb-*")
	require.NoError(t, err)

	db, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmpdir,
	})
	require.NoError(t, err)

	clear := func() {
		err = db.Close()
		require.NoError(t, err)

		err = os.RemoveAll(tmpdir)
		require.NoError(t, err)
	}

	return db, clear
}

func TestVerifyProof(t *testing.T) {
	trie, entries := randomTrie(t, 200)
	root, err := trie.Hash()
	require.NoError(t, err)

	amount := make(chan struct{}, 15)
	wg := new(sync.WaitGroup)

	for _, entry := range entries {
		wg.Add(1)
		go func(kv *kv) {
			defer func() {
				wg.Done()
				<-amount
			}()

			amount <- struct{}{}

			proof, clear := inMemoryChainDB(t)
			defer clear()

			_, err := trie.GenerateProof(kv.k, proof)
			require.NoError(t, err)
			v, err := VerifyProof(root, kv.k, proof)

			require.NoError(t, err)
			require.True(t, v)
		}(entry)
	}

	wg.Wait()
}

func TestVerifyProofOneElement(t *testing.T) {
	trie := NewEmptyTrie()
	key := randBytes(32)
	trie.Put(key, []byte("V"))

	rootHash, err := trie.Hash()
	require.NoError(t, err)

	proof, clear := inMemoryChainDB(t)
	defer clear()

	_, err = trie.GenerateProof(key, proof)
	require.NoError(t, err)

	val, err := VerifyProof(rootHash, key, proof)
	require.NoError(t, err)

	require.True(t, val)
}

func TestVerifyProof_BadProof(t *testing.T) {
	trie, entries := randomTrie(t, 200)
	rootHash, err := trie.Hash()
	require.NoError(t, err)

	amount := make(chan struct{}, 15)
	wg := new(sync.WaitGroup)

	for _, entry := range entries {
		wg.Add(1)

		go func(kv *kv) {
			defer func() {
				wg.Done()
				<-amount
			}()

			amount <- struct{}{}
			proof, clear := inMemoryChainDB(t)
			defer clear()

			nLen, err := trie.GenerateProof(kv.k, proof)
			require.Greater(t, nLen, 0)
			require.NoError(t, err)

			it := proof.NewIterator()
			for i, d := 0, rand.Intn(nLen); i <= d; i++ {
				it.Next()
			}
			key := it.Key()
			val, _ := proof.Get(key)
			proof.Del(key)
			it.Release()

			newhash, err := common.Keccak256(val)
			require.NoError(t, err)
			proof.Put(newhash.ToBytes(), val)

			v, err := VerifyProof(rootHash, kv.k, proof)
			require.NoError(t, err)
			require.False(t, v)
		}(entry)
	}

	wg.Wait()
}

func TestGenerateProofMissingKey(t *testing.T) {
	trie := NewEmptyTrie()

	parentKey, parentVal := randBytes(32), randBytes(20)
	chieldKey, chieldValue := modifyLastBytes(parentKey), modifyLastBytes(parentVal)
	gransonKey, gransonValue := modifyLastBytes(chieldKey), modifyLastBytes(chieldValue)

	trie.Put(parentKey, parentVal)
	trie.Put(chieldKey, chieldValue)
	trie.Put(gransonKey, gransonValue)

	proof, clear := inMemoryChainDB(t)
	defer clear()

	searchfor := make([]byte, len(gransonKey))
	copy(searchfor[:], gransonKey[:])

	// keep the path til the key but modify the last element
	searchfor[len(searchfor)-1] = searchfor[len(searchfor)-1] + byte(0xff)

	_, err := trie.GenerateProof(searchfor, proof)
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

	proof, clear := inMemoryChainDB(t)
	defer clear()

	searchfor := make([]byte, len(parentKey))
	copy(searchfor[:], parentKey[:])

	// the keys are equals until the byte number 20 so we modify the byte number 20 to another
	// value and the branch node will no be able to found the right slot
	searchfor[20] = searchfor[20] + byte(0xff)

	_, err := trie.GenerateProof(searchfor, proof)
	require.Error(t, err, "no more paths to follow")
}

type kv struct {
	k []byte
	v []byte
}

func randomTrie(t *testing.T, n int) (*Trie, map[string]*kv) {
	t.Helper()

	trie := NewEmptyTrie()
	vals := make(map[string]*kv)

	for i := 0; i < n; i++ {
		v := &kv{randBytes(32), randBytes(20)}
		trie.Put(v.k, v.v)
		vals[string(v.k)] = v
	}

	return trie, vals
}

func randBytes(n int) []byte {
	r := make([]byte, n)
	crand.Read(r)
	return r
}

func modifyLastBytes(b []byte) []byte {
	newB := make([]byte, len(b))
	copy(newB[:], b)

	rb := randBytes(12)
	copy(newB[20:], rb)

	return newB
}
