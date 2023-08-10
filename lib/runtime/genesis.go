// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
)

var (
	ErrGenesisTopNotFound = errors.New("genesis top not found")
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

	//TODO: check if we could get the trie version from genesis
	tr, err = trie.LoadFromMap(keyValues, trie.V0)
	if err != nil {
		return tr, fmt.Errorf("loading genesis top key values into trie: %w", err)
	}

	return tr, nil
}
