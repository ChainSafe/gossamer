package triedb

import "github.com/ChainSafe/gossamer/pkg/trie/hashdb"

type DBValue = []byte

type TrieDBBuilder[Out HashOut] struct {
	db       hashdb.HashDBReadOnly[Out, DBValue]
	root     Out
	cache    TrieCache[Out]
	recorder TrieRecorder[Out]
}
