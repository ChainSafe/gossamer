// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testKey = "key"

var testValue = []byte("value")

func TestStorageDiff_MainTrie(t *testing.T) {
	t.Parallel()
	t.Run("get", func(t *testing.T) {
		t.Parallel()
		t.Run("From empty", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			val, deleted := changes.get("test")

			require.False(t, deleted)
			require.Nil(t, val)
		})

		t.Run("Found", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			changes.upsert("test", []byte("test"))

			val, deleted := changes.get("test")

			require.False(t, deleted)
			require.Equal(t, []byte("test"), val)
		})

		t.Run("Not Found", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			changes.upsert("notfound", []byte("test"))

			val, deleted := changes.get("test")

			require.False(t, deleted)
			require.Nil(t, val)
		})
	})
	t.Run("upsert", func(t *testing.T) {
		t.Parallel()

		changes := newStorageDiff()

		changes.upsert(testKey, testValue)

		val, deleted := changes.get(testKey)
		require.False(t, deleted)
		require.Equal(t, testValue, val)
	})
	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		changes := newStorageDiff()
		changes.upsert(testKey, testValue)
		changes.delete(testKey)

		val, deleted := changes.get(testKey)
		require.True(t, deleted)
		require.Nil(t, val)
	})
	t.Run("clearPrefix", func(t *testing.T) {
		t.Parallel()

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

func TestStorageDiff_ChildTrie(t *testing.T) {
	t.Parallel()
	t.Run("getFromChild", func(t *testing.T) {
		t.Parallel()

		t.Run("Empty storage diff", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			val, deleted := changes.getFromChild("notFound", "testChildKey")

			require.False(t, deleted)
			require.Nil(t, val)
		})

		t.Run("Non existent child", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			changes.upsertChild("testChild", "testChildKey", []byte("test"))
			val, deleted := changes.getFromChild("notFound", "testChildKey")

			require.False(t, deleted)
			require.Nil(t, val)
		})

		t.Run("Not Found in child", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			changes.upsertChild("testChild", "testChildKey", []byte("test"))
			val, deleted := changes.getFromChild("testChild", "notFound")

			require.False(t, deleted)
			require.Nil(t, val)
		})

		t.Run("Found in child", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			changes.upsertChild("testChild", "testChildKey", []byte("test"))
			val, deleted := changes.getFromChild("testChild", "testChildKey")

			require.False(t, deleted)
			require.Equal(t, []byte("test"), val)
		})
	})

	t.Run("upsertChild", func(t *testing.T) {
		t.Parallel()

		changes := newStorageDiff()

		childkey := "testChild"
		changes.upsertChild(childkey, testKey, testValue)

		val, deleted := changes.getFromChild(childkey, testKey)
		require.False(t, deleted)
		require.Equal(t, testValue, val)
	})

	t.Run("deleteFromChild", func(t *testing.T) {
		t.Parallel()

		changes := newStorageDiff()

		childkey := "testChild"
		changes.upsertChild(childkey, testKey, testValue)
		changes.deleteFromChild(childkey, testKey)

		val, deleted := changes.getFromChild(childkey, testKey)
		require.True(t, deleted)
		require.Nil(t, val)
	})

	t.Run("deleteChildLimit", func(t *testing.T) {
		t.Parallel()

		testEntries := map[string][]byte{
			"key1": []byte("key1"),
			"key2": []byte("key2"),
			"key3": []byte("key3"),
		}

		cases := map[string]struct {
			limit            int
			currentChildKeys []string
			deleted          uint32
			allDelted        bool
		}{
			"empty_child_trie_limit_1": {
				limit:     1,
				deleted:   3, // Since keys during block exec does not count
				allDelted: true,
			},
			"empty_child_trie_limit_2": {
				limit:     2,
				deleted:   3, // Since keys during block exec does not count
				allDelted: true,
			},
			"empty_child_trie_same_limit_than_stored_keys": {
				limit:     3,
				deleted:   3,
				allDelted: true,
			},
			"empty_child_trie_no_limit": {
				limit:     -1,
				deleted:   3,
				allDelted: true,
			},
			"with_current_child_trie_1_entry_limit_1": {
				limit:            1,
				currentChildKeys: []string{"currentKey1"},
				deleted:          1, // Deletes currentKey1 only
				allDelted:        false,
			},
			"with_current_child_trie_1_entry_limit_2": {
				limit:            2,
				currentChildKeys: []string{"currentKey1"},
				deleted:          4, // Since keys during block exec does not count
				allDelted:        true,
			},
			"with_current_child_trie_1_entry_limit_3": {
				limit:            3,
				currentChildKeys: []string{"currentKey1"},
				deleted:          4,
				allDelted:        true,
			},
			"with_current_child_trie_with_no_limit": {
				limit:            -1,
				currentChildKeys: []string{"currentKey1"},
				deleted:          4,
				allDelted:        true,
			},
		}

		for tname, tt := range cases {
			tt := tt
			t.Run(tname, func(t *testing.T) {
				//t.Parallel()

				changes := newStorageDiff()

				for k, v := range testEntries {
					changes.upsertChild(testKey, k, v)
				}

				deleted, allDeleted := changes.deleteChildLimit(testKey, tt.currentChildKeys, tt.limit)
				require.Equal(t, tt.deleted, deleted)
				require.Equal(t, tt.allDelted, allDeleted)
			})
		}
	})
}
