// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrieDB_Lookup(t *testing.T) {
	t.Run("root_not_exists_in_db", func(t *testing.T) {
		db := newTestDB(t)
		empty := runtime.BlakeTwo256{}.Hash([]byte{0})
		lookup := NewTrieLookup[hash.H256, runtime.BlakeTwo256, []byte](db, empty, nil, nil, nil)

		value, err := lookup.Lookup([]byte("test"))
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrInvalidStateRoot)
	})
}

type trieCacheImpl struct{}

func (trieCacheImpl) GetValue(key []byte) CachedValue[hash.H256]         { return nil }
func (*trieCacheImpl) SetValue(key []byte, value CachedValue[hash.H256]) {}
func (*trieCacheImpl) GetOrInsertNode(
	hash hash.H256, fetchNode func() (NodeOwned[hash.H256], error),
) (NodeOwned[hash.H256], error) {
	return fetchNode()
}
func (*trieCacheImpl) GetNode(hash hash.H256) NodeOwned[hash.H256] { return nil }

func Test_TrieLookup_lookupValueWithCache(t *testing.T) {
	cache := &trieCacheImpl{}
	inmemoryDB := NewMemoryDB[hash.H256, runtime.BlakeTwo256](EmptyNode)
	trieDB := NewEmptyTrieDB[hash.H256, runtime.BlakeTwo256](
		inmemoryDB,
		WithCache[hash.H256, runtime.BlakeTwo256](cache),
	)
	trieDB.SetVersion(trie.V1)

	entries := map[string][]byte{
		"no":           make([]byte, 1),
		"noot":         make([]byte, 2),
		"not":          make([]byte, 3),
		"notable":      make([]byte, 4),
		"notification": make([]byte, 33),
		"test":         make([]byte, 6),
		"dimartiro":    make([]byte, 7),
	}

	for k, v := range entries {
		require.NoError(t, trieDB.Put([]byte(k), v))
	}

	err := trieDB.commit()
	require.NoError(t, err)

	lookup := NewTrieLookup[hash.H256, runtime.BlakeTwo256](
		inmemoryDB,
		trieDB.rootHash,
		cache,
		nil,
		func(data []byte) []byte {
			return data
		},
	)

	for k, v := range entries {
		bytes, err := lookup.lookupWithCache([]byte(k), nibbles.NewNibbles([]byte(k)))
		require.NoError(t, err)
		require.NotNil(t, bytes)
		require.Equal(t, []byte(v), *bytes)
	}
}

func Test_valueHash_CachedValue(t *testing.T) {
	var vh *valueHash[hash.H256]
	assert.Equal(t, NonExistingCachedValue[hash.H256]{}, vh.CachedValue())

	vh = &valueHash[hash.H256]{
		Value: []byte("someValue"),
		Hash:  hash.NewRandomH256(),
	}
	assert.Equal(t, ExistingCachedValue[hash.H256]{vh.Hash, vh.Value}, vh.CachedValue())
}
