package proof

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
)

func Test_GenerateProofForLeaf(t *testing.T) {
	testCases := map[string]struct {
		entries        []triedb.Entry
		storageVersion trie.TrieLayout
		keys           []string
		expectedProof  [][]byte
	}{
		"leaf": {
			entries: []triedb.Entry{
				{
					Key:   []byte("a"),
					Value: []byte("a"),
				},
			},
			keys: []string{"a"},
			expectedProof: [][]byte{
				{66, 97, 0}, // 'a' node without value
			},
		},
		"branch_and_leaf": {
			entries: []triedb.Entry{
				{
					Key:   []byte("a"),
					Value: []byte("a"),
				},
				{
					Key:   []byte("ab"),
					Value: []byte("ab"),
				},
			},
			keys: []string{"ab"},
			expectedProof: [][]byte{
				{194, 97, 64, 0, 4, 97, 12, 65, 2, 0},
			},
		},
		"complex_trie": {
			entries: []triedb.Entry{
				{
					Key:   []byte("pol"),
					Value: []byte("pol"),
				},
				{
					Key:   []byte("polka"),
					Value: []byte("polka"),
				},
				{
					Key:   []byte("polkadot"),
					Value: []byte("polkadot"),
				},
				{
					Key:   []byte("go"),
					Value: []byte("go"),
				},
				{
					Key:   []byte("golang"),
					Value: []byte("golang"),
				},
				{
					Key:   []byte("gossamer"),
					Value: []byte("gossamer"),
				},
			},
			keys: []string{"go"},
			expectedProof: [][]byte{
				{
					128, 192, 0, 0, 128, 114, 166, 121, 79, 225, 146, 229,
					34, 68, 211, 54, 148, 205, 192, 58, 131, 95, 46, 239,
					201, 206, 94, 116, 179, 122, 33, 19, 156, 225, 190, 57, 57,
				},
				{
					131, 7, 111, 192, 0, 48, 71, 12, 97, 110, 103, 24, 103,
					111, 108, 97, 110, 103, 64, 75, 3, 115, 97, 109, 101,
					114, 32, 103, 111, 115, 115, 97, 109, 101, 114,
				},
			},
		},
		"complex_trie_multiple_keys": {
			entries: []triedb.Entry{
				{
					Key:   []byte("pol"),
					Value: []byte("pol"),
				},
				{
					Key:   []byte("polka"),
					Value: []byte("polka"),
				},
				{
					Key:   []byte("polkadot"),
					Value: []byte("polkadot"),
				},
				{
					Key:   []byte("go"),
					Value: []byte("go"),
				},
				{
					Key:   []byte("golang"),
					Value: []byte("golang"),
				},
				{
					Key:   []byte("gossamer"),
					Value: []byte("gossamer"),
				},
			},
			keys: []string{"go", "polkadot"},
			expectedProof: [][]byte{
				{
					128, 192, 0, 0, 0,
				},
				{
					131, 7, 111, 192, 0, 48, 71, 12, 97, 110, 103, 24,
					103, 111, 108, 97, 110, 103, 64, 75, 3, 115, 97,
					109, 101, 114, 32, 103, 111, 115, 115, 97, 109,
					101, 114,
				},
				{
					197, 0, 111, 108, 64, 0, 12, 112, 111, 108, 68,
					195, 11, 97, 64, 0, 20, 112, 111, 108, 107, 97,
					20, 69, 4, 111, 116, 0,
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			// Build trie
			inmemoryDB := NewMemoryDB(triedb.EmptyNode)
			triedb := triedb.NewEmptyTrieDB(inmemoryDB)

			for _, entry := range testCase.entries {
				triedb.Put(entry.Key, entry.Value)
			}

			root := triedb.MustHash()

			// Generate proof
			proof, err := GenerateProof(inmemoryDB, testCase.storageVersion, root, testCase.keys)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedProof, proof)
		})
	}
}
