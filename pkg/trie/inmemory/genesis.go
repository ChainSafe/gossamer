// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

// GenesisBlock creates a genesis block from the trie.
func (t *InMemoryTrie) GenesisBlock() (genesisHeader types.Header, err error) {
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
