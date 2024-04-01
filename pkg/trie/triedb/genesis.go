package triedb

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

func (t *TrieDB) GenesisBlock() (genesisHeader types.Header, err error) {
	rootHash, err := t.Hash()
	if err != nil {
		return genesisHeader, fmt.Errorf("root hashing trie: %w", err)
	}

	parentHash := common.Hash{0}
	extrinsicRoot := trie.EmptyHash
	const blockNumber = 0
	digest := types.NewDigest()
	genesisHeader = *types.NewHeader(parentHash, rootHash, extrinsicRoot, blockNumber, digest)
	return genesisHeader, nil
}
