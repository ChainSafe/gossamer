// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/assert"
)

func TestTrieDB_Lookup(t *testing.T) {
	t.Run("root_not_exists_in_db", func(t *testing.T) {
		db := newTestDB(t)
		lookup := NewTrieLookup(db, trie.EmptyHash, nil)

		value, err := lookup.lookupValue([]byte("test"))
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrIncompleteDB)
	})
}
