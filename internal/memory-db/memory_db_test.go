package memorydb

import (
	"testing"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hashing"
	"github.com/stretchr/testify/assert"
)

var (
	_ KeyFunction[hash.H256, hash.H256] = HashKey[hash.H256]{}
	_ KeyFunction[hash.H256, []byte]    = PrefixedKey[hash.H256]{}
)

// / Blake2-256 Hash implementation.
type Keccak256 struct{}

// / Produce the hash of some byte-slice.
func (k256 Keccak256) Hash(s []byte) hash.H256 {
	h := hashing.Keccak256(s)
	return hash.H256(h[:])
}

func TestMemoryDB_RemoveAndPurge(t *testing.T) {
	helloBytes := []byte("Hello world!")
	helloKey := Keccak256{}.Hash(helloBytes)

	m := NewMemoryDB[hash.H256, Keccak256, hash.H256, HashKey[hash.H256], []byte]([]byte{0})
	m.Remove(helloKey, hashdb.EmptyPrefix)
	assert.Equal(t, int32(-1), m.raw(helloKey, hashdb.EmptyPrefix).RC)
	m.Purge()
	assert.Equal(t, int32(-1), m.raw(helloKey, hashdb.EmptyPrefix).RC)
	m.Insert(hashdb.EmptyPrefix, helloBytes)
	assert.Equal(t, int32(0), m.raw(helloKey, hashdb.EmptyPrefix).RC)
	m.Purge()
	assert.Nil(t, m.raw(helloKey, hashdb.EmptyPrefix))

	m = NewMemoryDB[hash.H256, Keccak256, hash.H256, HashKey[hash.H256], []byte]([]byte{0})
	assert.Nil(t, m.removeAndPurge(helloKey, hashdb.EmptyPrefix))
	assert.Equal(t, int32(-1), m.raw(helloKey, hashdb.EmptyPrefix).RC)
	m.Insert(hashdb.EmptyPrefix, helloBytes)
	m.Insert(hashdb.EmptyPrefix, helloBytes)
	assert.Equal(t, int32(1), m.raw(helloKey, hashdb.EmptyPrefix).RC)
	assert.Equal(t, &helloBytes, m.removeAndPurge(helloKey, hashdb.EmptyPrefix))
	assert.Nil(t, m.raw(helloKey, hashdb.EmptyPrefix))
	assert.Nil(t, m.removeAndPurge(helloKey, hashdb.EmptyPrefix))
}

func TestMemoryDB_Consolidate(t *testing.T) {
	main := NewMemoryDB[hash.H256, Keccak256, hash.H256, HashKey[hash.H256], []byte]([]byte{0})
	other := NewMemoryDB[hash.H256, Keccak256, hash.H256, HashKey[hash.H256], []byte]([]byte{0})
	removeKey := other.Insert(hashdb.EmptyPrefix, []byte("doggo"))
	main.Remove(removeKey, hashdb.EmptyPrefix)

	insertKey := other.Insert(hashdb.EmptyPrefix, []byte("arf"))
	main.Emplace(insertKey, hashdb.EmptyPrefix, []byte("arf"))

	negativeRemoveKey := other.Insert(hashdb.EmptyPrefix, []byte("negative"))
	other.Remove(negativeRemoveKey, hashdb.EmptyPrefix)
	other.Remove(negativeRemoveKey, hashdb.EmptyPrefix)
	main.Remove(negativeRemoveKey, hashdb.EmptyPrefix)

	main.Consolidate(&other)

	assert.Equal(t, &dataRC[[]byte]{[]byte("doggo"), 0}, main.raw(removeKey, hashdb.EmptyPrefix))
	assert.Equal(t, &dataRC[[]byte]{[]byte("arf"), 2}, main.raw(insertKey, hashdb.EmptyPrefix))
	assert.Equal(t, &dataRC[[]byte]{[]byte("negative"), -2}, main.raw(negativeRemoveKey, hashdb.EmptyPrefix))
}

func TestMemoryDB_DefaultWorks(t *testing.T) {
	db := NewMemoryDB[hash.H256, Keccak256, hash.H256, HashKey[hash.H256], []byte]([]byte{0})
	hashedNullNode := Keccak256{}.Hash([]byte{0})
	assert.Equal(t, hashedNullNode, db.Insert(hashdb.EmptyPrefix, []byte{0}))

	db2 := NewMemoryDB[hash.H256, Keccak256, hash.H256, HashKey[hash.H256], []byte]([]byte{0})
	root := db2.hashedNullNode
	assert.True(t, db2.Contains(root, hashdb.EmptyPrefix))
	assert.True(t, db.Contains(root, hashdb.EmptyPrefix))
}
