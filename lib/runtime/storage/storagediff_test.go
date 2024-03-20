// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/require"
)

const testKey = "key"

var testValue = []byte("value")

func Test_MainTrie(t *testing.T) {
	t.Parallel()
	t.Run("get", func(t *testing.T) {
		t.Parallel()
		t.Run("from_empty", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			val, deleted := changes.get("test")

			require.False(t, deleted)
			require.Nil(t, val)
		})

		t.Run("found", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			changes.upsert("test", []byte("test"))

			val, deleted := changes.get("test")

			require.False(t, deleted)
			require.Equal(t, []byte("test"), val)
		})

		t.Run("not_found", func(t *testing.T) {
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

func Test_ChildTrie(t *testing.T) {
	t.Parallel()
	t.Run("getFromChild", func(t *testing.T) {
		t.Parallel()

		t.Run("empty_storage_diff", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			val, deleted := changes.getFromChild("notFound", "testChildKey")

			require.False(t, deleted)
			require.Nil(t, val)
		})

		t.Run("non_existent_child", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			changes.upsertChild("testChild", "testChildKey", []byte("test"))
			val, deleted := changes.getFromChild("notFound", "testChildKey")

			require.False(t, deleted)
			require.Nil(t, val)
		})

		t.Run("not_found_in_child", func(t *testing.T) {
			t.Parallel()

			changes := newStorageDiff()
			changes.upsertChild("testChild", "testChildKey", []byte("test"))
			val, deleted := changes.getFromChild("testChild", "notFound")

			require.False(t, deleted)
			require.Nil(t, val)
		})

		t.Run("found_in_child", func(t *testing.T) {
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

	t.Run("clearPrefixInChild", func(t *testing.T) {
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
					changes.upsertChild("child", k, v)
				}

				deleted, allDeleted := changes.clearPrefixInChild("child", tt.prefix, tt.trieKeys, tt.limit)
				require.Equal(t, tt.deleted, deleted)
				require.Equal(t, tt.allDelted, allDeleted)
			})
		}
	})
}

func Test_Snapshot(t *testing.T) {
	t.Parallel()

	changes := newStorageDiff()

	changes.upsert("key1", []byte("value1"))
	changes.upsert("key2", []byte("value2"))
	changes.delete("key2")
	changes.upsertChild("childKey", "key1", []byte("value1"))
	changes.upsertChild("childKey", "key2", []byte("value2"))

	snapshot := changes.snapshot()

	require.Equal(t, changes, snapshot)
}

func Test_ApplyToTrie(t *testing.T) {
	t.Parallel()

	t.Run("add_entries_in_main_trie", func(t *testing.T) {
		t.Parallel()

		state := trie.NewEmptyTrie()

		key := "key1"
		value := []byte("value1")

		diff := newStorageDiff()
		diff.upsert(key, value)

		expected := trie.NewEmptyTrie()
		expected.Put([]byte(key), value)

		diff.applyToTrie(state)
		require.Equal(t, expected, state)
	})

	t.Run("delete_entries_from_main_trie", func(t *testing.T) {
		t.Parallel()

		state := trie.NewEmptyTrie()

		key := "key1"
		value := []byte("value1")

		state.Put([]byte(key), value)

		diff := newStorageDiff()
		diff.delete(key)

		expected := trie.NewEmptyTrie()

		diff.applyToTrie(state)
		require.Equal(t, expected, state)
	})
}
