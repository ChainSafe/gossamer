package babe

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// newDevGenesisWithTrieAndHeader generates test dev genesis, genesis trie and genesis header
func newDevGenesisWithTrieAndHeader(t *testing.T) (gen *genesis.Genesis, genesisTrie *trie.Trie, header *types.Header) {
	genesisPath := utils.GetDevV3SubstrateGenesisPath(t)

	gen, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)

	genesisTrie, err = genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}),
		genesisTrie.MustHash(), trie.EmptyHash, 0, types.NewDigest())
	require.NoError(t, err)

	return gen, genesisTrie, genesisHeader
}
