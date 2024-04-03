package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

func (t *TrieDB) GetChild(keyToChild []byte) (trie.Trie, error) {
	panic("not implemented yet")
}

func (t *TrieDB) GetFromChild(keyToChild, key []byte) ([]byte, error) {
	panic("not implemented yet")
}

func (t *TrieDB) GetChildTries() map[common.Hash]trie.Trie {
	panic("not implemented yet")
}
