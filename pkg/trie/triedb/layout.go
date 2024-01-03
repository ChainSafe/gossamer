package triedb

import "github.com/ChainSafe/gossamer/pkg/trie/hashdb"

type TrieLayout[Out HashOut] interface {
	UseExtension() bool
	AllowEmpty() bool
	MaxInlineValue() *uint
	Hasher() hashdb.Hasher[Out]
	Codec() NodeCodec[Out]
}
