// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

func (t *TrieDB[H, Hasher]) GetChild(keyToChild []byte) (trie.Trie, error) {
	panic("not implemented yet")
}

func (t *TrieDB[H, Hasher]) GetFromChild(keyToChild, key []byte) ([]byte, error) {
	panic("not implemented yet")
}

func (t *TrieDB[H, Hasher]) GetChildTries() map[common.Hash]trie.Trie {
	panic("not implemented yet")
}
