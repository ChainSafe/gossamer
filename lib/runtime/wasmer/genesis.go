// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

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
func NewTrieFromGenesis(gen genesis.Genesis) (tr *trie.Trie, err error) {
	tr = trie.NewEmptyTrie()
	genesisFields := gen.GenesisFields()
	keyValues, ok := genesisFields.Raw["top"]
	if !ok {
		return nil, fmt.Errorf("%w: in mapping %v",
			ErrGenesisTopNotFound, genesisFields.Raw)
	}

	err = tr.LoadFromMap(keyValues)
	if err != nil {
		return nil, fmt.Errorf("loading genesis top key values into trie: %w", err)
	}

	return tr, nil
}
