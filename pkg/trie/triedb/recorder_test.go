// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/karlseguin/ccache/v3/assert"
)

func TestRecorder(t *testing.T) {
	inmemoryDB := NewMemoryDB(emptyNode)

	triedb := NewEmptyTrieDB(inmemoryDB)
	triedb.SetVersion(trie.V1)

	triedb.Put([]byte("aa"), []byte("aavalue"))
	triedb.Put([]byte("aab"), []byte("aabvalue"))
	triedb.Put([]byte("aac"), make([]byte, 40))
	triedb.Put([]byte("aabb"), []byte("aabbvalue"))

	// Commit and get root
	root := triedb.MustHash()

	assert.NotNil(t, root)

	t.Run("Record_aa_access_should_record_1_node", func(t *testing.T) {
		recorder := NewRecorder()
		trie := NewTrieDB(root, inmemoryDB, WithRecorder(recorder))

		value := trie.Get([]byte("aa"))
		assert.True(t, bytes.Equal(value, []byte("aavalue")))

		assert.Equal(t, len(recorder.nodes), 1)
		assert.Equal(t, recorder.recordedKeys.Len(), 1)
		assert.Equal(t, recorder.recordedKeys.Keys()[0], string(codec.KeyLEToNibbles([]byte("aa"))))
	})

	t.Run("Record_aab_access_should_record_2_nodes_and_1_value", func(t *testing.T) {
		recorder := NewRecorder()
		trie := NewTrieDB(root, inmemoryDB, WithRecorder(recorder))

		value := trie.Get([]byte("aab"))
		assert.True(t, bytes.Equal(value, []byte("aabvalue")))

		assert.Equal(t, len(recorder.nodes), 2)
		assert.Equal(t, recorder.recordedKeys.Keys()[0], string(codec.KeyLEToNibbles([]byte("aab"))))
	})

	t.Run("Record_aabb_access_should_record_2_nodes", func(t *testing.T) {
		recorder := NewRecorder()
		trie := NewTrieDB(root, inmemoryDB, WithRecorder(recorder))

		value := trie.Get([]byte("aabb"))
		assert.True(t, bytes.Equal(value, []byte("aabbvalue")))

		assert.Equal(t, len(recorder.nodes), 2)
		assert.Equal(t, recorder.recordedKeys.Keys()[0], string(codec.KeyLEToNibbles([]byte("aabb"))))
	})
}
