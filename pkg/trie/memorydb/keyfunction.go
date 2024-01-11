package memorydb

import (
	"github.com/ChainSafe/gossamer/pkg/trie/hashdb"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibble"
)

type KeyFunction[Hash hashdb.HasherOut, H hashdb.Hasher[Hash]] interface {
	Key(hash Hash, prefix nibble.Prefix) Hash
}

type (
	HashKey[H hashdb.HasherOut] struct{}
	PrefixKey[H []byte]         struct{}
)

func (h HashKey[H]) Key(key H, prefix nibble.Prefix) H {
	return key
}

func (h PrefixKey[H]) Key(key H, prefix nibble.Prefix) H {
	newLen := len(key) + len(prefix.PartialKey) + 1
	prefixedKey := make([]byte, newLen)

	prefixedKey = append(prefixedKey, prefix.PartialKey...)
	if prefix.PaddedByte != nil {
		prefixedKey = append(prefixedKey, *prefix.PaddedByte)
	}
	prefixedKey = append(prefixedKey, key...)

	return H(prefixedKey)
}


