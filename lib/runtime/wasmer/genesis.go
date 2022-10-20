// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
)

var (
	ErrGenesisTopNotFound = fmt.Errorf("genesis top not found")
)

// NewTrieFromGenesis creates a new trie from the raw genesis data
func NewTrieFromGenesis(gen genesis.Genesis) (tr trie.Trie, err error) {
	triePtr := trie.NewEmptyTrie()
	tr = *triePtr
	genesisFields := gen.GenesisFields()
	keyValues, ok := genesisFields.Raw["top"]
	if !ok {
		return tr, fmt.Errorf("%w: in genesis %s",
			ErrGenesisTopNotFound, gen.Name)
	}

	tr, err = trie.LoadFromMap(keyValues)
	if err != nil {
		return tr, fmt.Errorf("loading genesis top key values into trie: %w", err)
	}

	return tr, nil
}
