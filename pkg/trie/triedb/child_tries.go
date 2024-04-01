package triedb

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

func (t *TrieDB) GetChild(keyToChild []byte) (trie.Trie, error) {
	panic("implement me")
}

func (t *TrieDB) GetFromChild(keyToChild, key []byte) ([]byte, error) {
	panic("implement me")
}

func (t *TrieDB) GetChildTries() map[common.Hash]trie.Trie {
	panic("implement me")
}
