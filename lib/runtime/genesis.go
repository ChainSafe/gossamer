// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

var (
	ErrGenesisTopNotFound = errors.New("genesis top not found")
)

// NewTrieFromGenesis creates a new trie from the raw genesis data
func NewInMemoryTrieFromGenesis(gen genesis.Genesis) (tr *trie.InMemoryTrie, err error) {
	tr = trie.NewEmptyTrie()
	genesisFields := gen.GenesisFields()
	keyValues, ok := genesisFields.Raw["top"]
	if !ok {
		return tr, fmt.Errorf("%w: in genesis %s",
			ErrGenesisTopNotFound, gen.Name)
	}

	tr, err = trie.LoadFromMap(keyValues, trie.V0)
	if err != nil {
		return tr, fmt.Errorf("loading genesis top key values into trie: %w", err)
	}

	return tr, nil
}
