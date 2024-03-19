// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageDiff_MainTrie(t *testing.T) {
	t.Run("Upsert", func(t *testing.T) {
		changes := newStorageDiff()

		key := "key"
		value := []byte("value")
		changes.upsert(key, value)

		val, deleted := changes.get(key)
		require.False(t, deleted)
		require.Equal(t, value, val)
	})

	t.Run("Delete", func(t *testing.T) {
		changes := newStorageDiff()

		key := "key"
		value := []byte("value")
		changes.upsert(key, value)
		changes.delete(key)

		val, deleted := changes.get(key)
		require.True(t, deleted)
		require.Nil(t, val)
	})

	t.Run("clearPrefix", func(t *testing.T) {
		testEntries := map[string][]byte{
			"pre":        []byte("pre"),
			"predict":    []byte("predict"),
			"prediction": []byte("prediction"),
		}

		commonPrefix := []byte("pre")

		cases := map[string]struct {
			prefix    []byte
			limit     int
			trieKeys  []string
			deleted   uint32
			allDelted bool
		}{
			"empty_trie_limit_1": {
				prefix:    commonPrefix,
				limit:     1,
				deleted:   3, // Since keys during block exec does not count
				allDelted: true,
			},
			"empty_trie_limit_2": {
				prefix:    commonPrefix,
				limit:     2,
				deleted:   3, // Since keys during block exec does not count
				allDelted: true,
			},
			"empty_trie_same_limit_than_stored_keys": {
				prefix:    commonPrefix,
				limit:     3,
				deleted:   3,
				allDelted: true,
			},
			"empty_trie_no_limit": {
				prefix:    commonPrefix,
				limit:     -1,
				deleted:   3,
				allDelted: true,
			},
			"with_previous_state_not_sharing_prefix_limit_1": {
				prefix:    commonPrefix,
				limit:     1,
				trieKeys:  []string{"bio"},
				deleted:   3, // Since keys during block exec does not count
				allDelted: false,
			},
			"with_previous_state_not_sharing_prefix_limit_2": {
				prefix:    commonPrefix,
				limit:     2,
				trieKeys:  []string{"bio"},
				deleted:   3, // Since keys during block exec does not count
				allDelted: false,
			},
			"with_previous_state_not_sharing_prefix_limit_3": {
				prefix:    commonPrefix,
				limit:     3,
				trieKeys:  []string{"bio"},
				deleted:   3,
				allDelted: false,
			},
			"with_previous_state_not_sharing_prefix_with_no_limit": {
				prefix:    commonPrefix,
				limit:     -1,
				trieKeys:  []string{"bio"},
				deleted:   3,
				allDelted: false,
			},
			"with_previous_state_sharing_prefix_limit_1": {
				prefix:    []byte("p"),
				limit:     1,
				trieKeys:  []string{"p"},
				deleted:   1, // the "p" key only
				allDelted: false,
			},
			"with_previous_state_sharing_prefix_limit_2": {
				prefix:    []byte("p"),
				limit:     2,
				trieKeys:  []string{"p"},
				deleted:   4, // Since keys during block exec does not count
				allDelted: true,
			},
			"with_previous_state_sharing_prefix_limit_3": {
				prefix:    []byte("p"),
				limit:     3,
				trieKeys:  []string{"p"},
				deleted:   4,
				allDelted: true,
			},
			"with_previous_state_sharing_prefix_with_no_limit": {
				prefix:    []byte("p"),
				limit:     -1,
				trieKeys:  []string{"p"},
				deleted:   4,
				allDelted: true,
			},
		}

		for tname, tt := range cases {
			tt := tt
			t.Run(tname, func(t *testing.T) {
				t.Parallel()

				changes := newStorageDiff()

				for k, v := range testEntries {
					changes.upsert(k, v)
				}

				deleted, allDeleted := changes.clearPrefix(tt.prefix, tt.trieKeys, tt.limit)
				require.Equal(t, tt.deleted, deleted)
				require.Equal(t, tt.allDelted, allDeleted)
			})
		}
	})
}
