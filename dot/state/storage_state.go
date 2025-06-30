// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

type StorageState interface {
	TrieState(bhash *common.Hash) (rtstorage.TrieState, error)
	StoreTrie(rtstorage.TrieState, *types.Header) error

	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	GenerateTrieProof(stateRoot common.Hash, keys [][]byte) ([][]byte, error)
	GetStorage(root *common.Hash, key []byte) ([]byte, error)
	GetStorageByBlockHash(bhash *common.Hash, key []byte) ([]byte, error)
	StorageRoot() (common.Hash, error)
	Entries(root *common.Hash) (map[string][]byte, error) // should be overhauled to iterate
	GetKeysWithPrefix(root *common.Hash, prefix []byte) ([][]byte, error)
	GetStorageChild(root *common.Hash, keyToChild []byte) (trie.Trie, error)
	GetStorageFromChild(root *common.Hash, keyToChild, key []byte) ([]byte, error)

	LoadCode(hash *common.Hash) ([]byte, error)
	LoadCodeHash(hash *common.Hash) (common.Hash, error)

	RegisterStorageObserver(o Observer)
	UnregisterStorageObserver(o Observer)

	sync.Locker
}
