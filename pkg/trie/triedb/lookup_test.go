// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
	"github.com/stretchr/testify/assert"
)

func TestTrieDB_Lookup(t *testing.T) {
	t.Run("root_not_exists_in_db", func(t *testing.T) {
		db := newTestDB(t)
		empty := runtime.BlakeTwo256{}.Hash([]byte{0})
		lookup := NewTrieLookup[hash.H256, runtime.BlakeTwo256](db, empty, nil, nil)

		value, err := lookup.lookupValue([]byte("test"), nibbles.NewNibbles([]byte("test")))
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrIncompleteDB)
	})
}

// TODO: restore after implementing node level caching
// func Test_valueHash_CachedValue(t *testing.T) {
// 	var vh *valueHash[hash.H256]
// 	assert.Equal(t, NonExistingCachedValue{}, vh.CachedValue())

// 	vh = &valueHash[hash.H256]{
// 		Value: []byte("someValue"),
// 		Hash:  hash.NewRandomH256(),
// 	}
// 	assert.Equal(t, ExistingCachedValue[hash.H256]{vh.Hash, vh.Value}, vh.CachedValue())
// }
