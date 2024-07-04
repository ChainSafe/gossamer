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

	triedb := NewEmptyTrieDB(inmemoryDB, nil, nil)
	triedb.SetVersion(trie.V1)

	triedb.Put([]byte("aa"), []byte("avalue"))
	triedb.Put([]byte("aab"), []byte("aabvalue"))
	triedb.Put([]byte("aac"), make([]byte, 40))
	triedb.Put([]byte("aabb"), []byte("avalue"))

	root := triedb.MustHash()

	assert.NotNil(t, root)

	t.Run("Record `aa` access should record 1 node", func(t *testing.T) {
		recorder := NewRecorder()
		trie := NewTrieDB(root, inmemoryDB, nil, recorder)

		value := trie.Get([]byte("aa"))
		assert.True(t, bytes.Equal(value, []byte("avalue")))

		assert.Equal(t, len(recorder.nodes), 1)
		assert.Equal(t, recorder.recordedKeys.Len(), 1)
		assert.Equal(t, recorder.recordedKeys.Keys()[0], string(codec.KeyLEToNibbles([]byte("aa"))))
	})

	t.Run("Record `aab` access should record 2 nodes + 1 value", func(t *testing.T) {
		recorder := NewRecorder()
		trie := NewTrieDB(root, inmemoryDB, nil, recorder)

		value := trie.Get([]byte("aab"))
		t.Log(value)
		assert.True(t, bytes.Equal(value, []byte("aabvalue")))

		assert.Equal(t, len(recorder.nodes), 2)
		assert.Equal(t, recorder.recordedKeys.Keys()[0], string(codec.KeyLEToNibbles([]byte("aab"))))
	})

	t.Run("Record `aac` access should record 2 nodes", func(t *testing.T) {
		recorder := NewRecorder()
		trie := NewTrieDB(root, inmemoryDB, nil, recorder)

		trie.Get([]byte("aac"))

		nodes := recorder.Drain()

		assert.Equal(t, len(nodes), 3)
	})
}
