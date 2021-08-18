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
	trie, entries := randomTrie(t, 500)
	root, err := trie.Hash()
	require.NoError(t, err)

	amount := make(chan struct{}, 50)
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

			_, err := trie.Prove(kv.k, 0, proof)
			require.NoError(t, err)
			v, err := VerifyProof(root, kv.k, proof)

			require.NoError(t, err)
			require.Equal(t, kv.v, v)
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

	_, err = trie.Prove(key, 0, proof)
	require.NoError(t, err)

	val, err := VerifyProof(rootHash, key, proof)
	require.NoError(t, err)

	require.Equal(t, []byte("V"), val)
}

func TestVerifyProof_BadProof(t *testing.T) {
	trie, entries := randomTrie(t, 200)
	rootHash, err := trie.Hash()
	require.NoError(t, err)

	amount := make(chan struct{}, 50)
	wg := new(sync.WaitGroup)

	for _, entry := range entries {
		wg.Add(1)

		go func(kv *kv) {
			defer func() {
				wg.Done()
				<-amount
			}()

			amount <- struct{}{}
			proof, cancel := inMemoryChainDB(t)
			defer cancel()

			nLen, err := trie.Prove(kv.k, 0, proof)
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

			_, err = VerifyProof(rootHash, kv.k, proof)
			require.Error(t, err)
		}(entry)
	}

	wg.Wait()
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
