package proof

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
)

func Test_GenerateProofForLeaf(t *testing.T) {
	inmemoryDB := NewMemoryDB(triedb.EmptyNode)

	// Trie with only one leaf
	triedb := triedb.NewEmptyTrieDB(inmemoryDB)
	triedb.Put([]byte("a"), []byte("a"))
	root := triedb.MustHash()

	// Keys to generate the proof
	keys := []string{"a"}

	proof, err := GenerateProof(inmemoryDB, trie.V0, root, keys)

	// Leaf node without value
	expectedNodes := [][]byte{
		{
			66, 97, 0,
		},
	}

	require.NoError(t, err)
	require.Equal(t, proof, expectedNodes)
}
