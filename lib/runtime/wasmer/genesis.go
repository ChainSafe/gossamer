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

// NewTrieFromGenesis creates a new trie from the raw genesis data.
// TODO inject state version perhaps? Check usages
func NewTrieFromGenesis(gen genesis.Genesis) (tr trie.Trie, err error) {
	stateVersion, err := StateVersionFromGenesis(gen)
	if err != nil {
		return tr, fmt.Errorf("getting genesis state version: %w", err)
	}

	triePtr := trie.NewEmptyTrie()
	tr = *triePtr
	genesisFields := gen.GenesisFields()
	keyValues, ok := genesisFields.Raw["top"]
	if !ok {
		return tr, fmt.Errorf("%w: in genesis %s",
			ErrGenesisTopNotFound, gen.Name)
	}

	err = tr.LoadFromMap(keyValues, stateVersion)
	if err != nil {
		return tr, fmt.Errorf("loading genesis top key values into trie: %w", err)
	}

	return tr, nil
}

func StateVersionFromGenesis(genesis genesis.Genesis) (
	stateVersion trie.Version, err error) {
	runtimeCode, err := genesis.RuntimeCode()
	if err != nil {
		return stateVersion, fmt.Errorf("getting genesis runtime code: %w", err)
	}

	stateVersion, err = GetRuntimeStateVersion(runtimeCode)
	if err != nil {
		return stateVersion, fmt.Errorf("getting genesis runtime state version: %w", err)
	}

	return stateVersion, nil
}
