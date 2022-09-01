// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

func Test_NewTrieFromGenesis(t *testing.T) {
	rawGenesis := genesis.Genesis{
		Genesis: genesis.Fields{
			Raw: map[string]map[string]string{
				"top": {"0x3a636f6465": "0x0102"},
			},
		},
	}

	expTrie := trie.NewEmptyTrie()
	expTrie.Put([]byte(`:code`), []byte{1, 2})

	trie, err := NewTrieFromGenesis(rawGenesis)
	require.NoError(t, err)

	require.Equal(t, expTrie, trie)
}
