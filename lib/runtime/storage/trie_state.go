package storage

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

type TrieState interface {
	SetVersion(v trie.TrieLayout)
	StartTransaction()
	RollbackTransaction()
	CommitTransaction()
	Put(key, value []byte) error
	Get(key []byte) []byte
	Root() (common.Hash, error)
	Trie() trie.Trie
	Has(key []byte) bool
	Delete(key []byte) error
	NextKey(key []byte) []byte
	ClearPrefix(prefix []byte) error
	ClearPrefixLimit(prefix []byte, limit uint32) (loops uint32, deleted uint32, allDeleted bool, err error)
	TrieEntries() map[string][]byte
	SetChildStorage(keyToChild, key, value []byte) error
	GetChildRoot(keyToChild []byte) (common.Hash, error)
	GetChildStorage(keyToChild, key []byte) ([]byte, error)
	DeleteChild(keyToChild []byte) error
	DeleteChildLimit(key []byte, limit *[]byte) (deleted uint32, allDeleted bool, err error)
	ClearChildStorage(keyToChild, key []byte) error
	ClearPrefixInChild(keyToChild, prefix []byte) error
	ClearPrefixInChildWithLimit(keyToChild, prefix []byte, limit uint32) (uint32, uint32, bool, error)
	GetChildNextKey(keyToChild, key []byte) ([]byte, error)
	GetKeysWithPrefixFromChild(keyToChild, prefix []byte) ([][]byte, error)
	LoadCode() []byte
	LoadCodeHash() (common.Hash, error)
	GetChangedNodeHashes() (inserted, deleted map[common.Hash]struct{}, err error)
}
