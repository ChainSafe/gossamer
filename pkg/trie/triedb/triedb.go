package triedb

import "github.com/ChainSafe/gossamer/pkg/trie/hashdb"

type TrieDB[Out HashOut] struct {
	db       hashdb.HashDB[Out, DBValue]
	root     Out
	cache    TrieCache[Out]
	recorder TrieRecorder[Out]
}

func NewTrieDB[H HashOut](
	db hashdb.HashDB[H, DBValue],
	root H,
	cache TrieCache[H],
	recorder TrieRecorder[H],
) *TrieDB[H] {
	return &TrieDB[H]{
		db:       db,
		root:     root,
		cache:    cache,
		recorder: recorder,
	}
}
