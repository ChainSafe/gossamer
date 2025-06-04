package trie

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hasher"
	"github.com/ChainSafe/gossamer/internal/primitives/kv"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
)

type KeyValue = kv.KeyValue

func BlakeTwo256Root(input []KeyValue, version storage.StateVersion) hash.H256 {
	switch version {
	case storage.StateVersionV0:
		return trie.LayoutV0[hasher.Blake2Hasher, hash.H256]{}.TrieRoot(input)
	case storage.StateVersionV1:
		return trie.LayoutV1[hasher.Blake2Hasher, hash.H256]{}.TrieRoot(input)
	default:
		panic("unreachable")
	}
}
