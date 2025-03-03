// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package recorder

import (
	"testing"

	memorydb "github.com/ChainSafe/gossamer/internal/memory-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	ptrie "github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func makeValue(i uint8) []byte {
	val := make([]byte, 64)
	for j := 0; j < len(val); j++ {
		val[j] = i
	}
	return val
}

var testData []ptrie.KeyValue = []ptrie.KeyValue{
	{
		Key:   []byte("key1"),
		Value: makeValue(1),
	},
	{
		Key:   []byte("key2"),
		Value: makeValue(2),
	},
	{
		Key:   []byte("key3"),
		Value: makeValue(3),
	},
	{
		Key:   []byte("key4"),
		Value: makeValue(4),
	},
}

type MemoryDB = memorydb.MemoryDB[
	hash.H256, runtime.BlakeTwo256, hash.H256, memorydb.HashKey[hash.H256],
]

func newMemoryDB() *MemoryDB {
	mdb := memorydb.NewMemoryDB[
		hash.H256, runtime.BlakeTwo256, hash.H256, memorydb.HashKey[hash.H256],
	]([]byte{0})
	return &mdb
}

func createTrie(t *testing.T) (db *MemoryDB, root hash.H256) {
	t.Helper()
	db = newMemoryDB()
	trieDB := triedb.NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](db)
	trieDB.SetVersion(trie.V1)

	for _, td := range testData {
		err := trieDB.Set(td.Key, td.Value)
		require.NoError(t, err)
	}

	root, err := trieDB.Hash()
	require.NoError(t, err)

	return db, root
}
func TestRecorder(t *testing.T) {
	db, root := createTrie(t)
	rec := Recorder[hash.H256]{inner: newRecorderInner[hash.H256]()}

	{
		trieRecorder := rec.TrieRecorder(root)
		trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
		trieDB.SetVersion(trie.V1)
		val, err := trieDB.Get(testData[0].Key)
		require.NoError(t, err)
		require.Equal(t, testData[0].Value, val)
	}

	storageProof := rec.DrainStorageProof()
	memDB := ptrie.NewMemoryDBFromStorageProof[hash.H256, runtime.BlakeTwo256](storageProof)

	// Check that we recorded the required data
	trieDB := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](root, memDB)
	trieDB.SetVersion(trie.V1)
	val, err := trieDB.Get(testData[0].Key)
	require.NoError(t, err)
	require.Equal(t, testData[0].Value, val)
}

type recorderStats struct {
	AccessedNodes int
	RecordedKeys  int
	EstimatedSize int
}

func newRecorderStats(recorder *Recorder[hash.H256]) (rs recorderStats) {
	recorder.innerMtx.Lock()
	defer recorder.innerMtx.Unlock()

	var recordedKeys int
	allKeys := maps.Values(recorder.inner.recordedKeys)
	for _, keys := range allKeys {
		recordedKeys = recordedKeys + len(maps.Keys(keys))
	}

	rs.RecordedKeys = recordedKeys
	rs.AccessedNodes = len(recorder.inner.accessedNodes)
	rs.EstimatedSize = int(recorder.EstimateEncodedSize())
	return
}

func TestRecorder_TransactionsRollback(t *testing.T) {
	db, root := createTrie(t)
	rec := Recorder[hash.H256]{inner: newRecorderInner[hash.H256]()}
	stats := make([]recorderStats, 0)
	stats = append(stats, recorderStats{})

	for i := 0; i < 4; i++ {
		rec.StartTransaction()
		{
			trieRecorder := rec.TrieRecorder(root)
			trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
			trieDB.SetVersion(trie.V1)
			val, err := trieDB.Get(testData[i].Key)
			require.NoError(t, err)
			require.Equal(t, testData[i].Value, val)
		}
		stats = append(stats, newRecorderStats(&rec))
	}

	require.Equal(t, 4, len(rec.inner.transactions))

	for i := 0; i < 5; i++ {
		assert.Equal(t, stats[4-i], newRecorderStats(&rec))

		storageProof := rec.StorageProof()
		memDB := ptrie.NewMemoryDBFromStorageProof[hash.H256, runtime.BlakeTwo256](storageProof)

		trieDB := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](root, memDB)
		trieDB.SetVersion(trie.V1)

		// Check that the required data is still present.
		for a := 0; a < 4; a++ {
			if a < 4-i {
				val, err := trieDB.Get(testData[a].Key)
				require.NoError(t, err)
				assert.Equal(t, testData[a].Value, val)
			} else {
				// All the data that we already rolled back, should be gone!
				val, err := trieDB.Get(testData[a].Key)
				require.Error(t, err)
				assert.Nil(t, val)
			}
		}

		if i < 4 {
			err := rec.RollBackTransaction()
			assert.NoError(t, err)
		}
	}
	assert.Equal(t, 0, len(rec.inner.transactions))
}

func TestRecorder_TransactionsCommit(t *testing.T) {
	db, root := createTrie(t)
	rec := Recorder[hash.H256]{inner: newRecorderInner[hash.H256]()}

	for i := 0; i < 4; i++ {
		rec.StartTransaction()
		{
			trieRecorder := rec.TrieRecorder(root)
			trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
			trieDB.SetVersion(trie.V1)
			val, err := trieDB.Get(testData[i].Key)
			assert.NoError(t, err)
			assert.Equal(t, testData[i].Value, val)
		}
	}

	stats := newRecorderStats(&rec)
	assert.Equal(t, 4, len(rec.inner.transactions))

	for i := 0; i < 4; i++ {
		err := rec.CommitTransaction()
		assert.NoError(t, err)
	}
	assert.Equal(t, 0, len(rec.inner.transactions))
	assert.Equal(t, stats, newRecorderStats(&rec))

	storageProof := rec.StorageProof()
	memDB := ptrie.NewMemoryDBFromStorageProof[hash.H256, runtime.BlakeTwo256](storageProof)

	// Check that we recorded the required data
	trieDB := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](root, memDB)
	trieDB.SetVersion(trie.V1)

	// Check that the required data is still present.
	for i := 0; i < 4; i++ {
		val, err := trieDB.Get(testData[i].Key)
		assert.NoError(t, err)
		assert.Equal(t, testData[i].Value, val)
	}
}

func TestRecorder_TransactionsCommitAndRollback(t *testing.T) {
	db, root := createTrie(t)
	rec := Recorder[hash.H256]{inner: newRecorderInner[hash.H256]()}

	for i := 0; i < 2; i++ {
		rec.StartTransaction()
		{
			trieRecorder := rec.TrieRecorder(root)
			trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
			trieDB.SetVersion(trie.V1)
			val, err := trieDB.Get(testData[i].Key)
			assert.NoError(t, err)
			assert.Equal(t, testData[i].Value, val)
		}
	}

	err := rec.RollBackTransaction()
	assert.NoError(t, err)

	for i := 2; i < 4; i++ {
		rec.StartTransaction()
		{
			trieRecorder := rec.TrieRecorder(root)
			trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
			trieDB.SetVersion(trie.V1)
			val, err := trieDB.Get(testData[i].Key)
			assert.NoError(t, err)
			assert.Equal(t, testData[i].Value, val)
		}
	}

	err = rec.RollBackTransaction()
	assert.NoError(t, err)

	assert.Equal(t, 2, len(rec.inner.transactions))

	for i := 0; i < 2; i++ {
		err := rec.CommitTransaction()
		assert.NoError(t, err)
	}

	assert.Equal(t, 0, len(rec.inner.transactions))

	storageProof := rec.StorageProof()
	memDB := ptrie.NewMemoryDBFromStorageProof[hash.H256, runtime.BlakeTwo256](storageProof)

	// Check that we recorded the required data
	trieDB := triedb.NewTrieDB[hash.H256, runtime.BlakeTwo256](root, memDB)
	trieDB.SetVersion(trie.V1)

	// Check that the required data is still present.
	for i := 0; i < 4; i++ {
		if i%2 == 0 {
			val, err := trieDB.Get(testData[i].Key)
			assert.NoError(t, err)
			assert.Equal(t, testData[i].Value, val)
		} else {
			val, err := trieDB.Get(testData[i].Key)
			assert.Error(t, err)
			assert.Nil(t, val)
		}
	}
}

func TestRecorder_TransactionAccessedKeys(t *testing.T) {
	key := testData[0].Key
	db, root := createTrie(t)
	rec := Recorder[hash.H256]{inner: newRecorderInner[hash.H256]()}

	{
		trieRecorder := rec.TrieRecorder(root)
		assert.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedNone)
	}

	rec.StartTransaction()
	{
		trieRecorder := rec.TrieRecorder(root)
		trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
		trieDB.SetVersion(trie.V1)

		hash, err := trieDB.GetHash(key)
		require.NoError(t, err)
		require.NotNil(t, hash)
		require.Equal(t, runtime.BlakeTwo256{}.Hash(testData[0].Value), *hash)

		assert.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedHash)
	}

	rec.StartTransaction()
	{
		trieRecorder := rec.TrieRecorder(root)
		trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
		trieDB.SetVersion(trie.V1)

		val, err := triedb.GetWith(trieDB, testData[0].Key, func(data []byte) []byte { return data })
		require.NoError(t, err)
		require.NotNil(t, val)
		require.Equal(t, testData[0].Value, *val)

		require.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedValue)
	}

	err := rec.RollBackTransaction()
	require.NoError(t, err)
	{
		trieRecorder := rec.TrieRecorder(root)
		require.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedHash)
	}

	err = rec.RollBackTransaction()
	require.NoError(t, err)
	{
		trieRecorder := rec.TrieRecorder(root)
		require.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedNone)
	}

	rec.StartTransaction()
	{
		trieRecorder := rec.TrieRecorder(root)
		trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
		trieDB.SetVersion(trie.V1)

		val, err := triedb.GetWith(trieDB, testData[0].Key, func(data []byte) []byte { return data })
		require.NoError(t, err)
		require.NotNil(t, val)
		require.Equal(t, testData[0].Value, *val)

		require.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedValue)
	}

	rec.StartTransaction()
	{
		trieRecorder := rec.TrieRecorder(root)
		trieDB := triedb.NewTrieDB(root, db, triedb.WithRecorder[hash.H256, runtime.BlakeTwo256](trieRecorder))
		trieDB.SetVersion(trie.V1)

		hash, err := trieDB.GetHash(testData[0].Key)
		require.NoError(t, err)
		require.NotNil(t, hash)
		require.Equal(t, runtime.BlakeTwo256{}.Hash(testData[0].Value), *hash)

		assert.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedValue)
	}

	err = rec.RollBackTransaction()
	require.NoError(t, err)
	{
		trieRecorder := rec.TrieRecorder(root)
		require.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedValue)
	}

	err = rec.RollBackTransaction()
	require.NoError(t, err)
	{
		trieRecorder := rec.TrieRecorder(root)
		require.Equal(t, trieRecorder.TrieNodesRecordedForKey(key), triedb.RecordedNone)
	}
}
