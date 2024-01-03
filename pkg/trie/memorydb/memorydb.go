package memorydb

import "github.com/ChainSafe/gossamer/pkg/trie/hashdb"

type MemoryDBValue[T any] struct {
	value T
	rc    int32
}

type MemoryDB[Hash hashdb.HasherOut, Hasher hashdb.Hasher[Hash], KF KeyFunction[Hash, Hasher], T any] struct {
	data           map[Hash]MemoryDBValue[T]
	hashedNullNode Hash
	nullNodeData   T
	keyFunction    KF
}

type KeyFunction[Hash hashdb.HasherOut, H hashdb.Hasher[Hash]] interface {
	Key(hash Hash, prefix hashdb.Prefix) Hash
}
