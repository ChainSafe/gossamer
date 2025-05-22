// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"encoding/binary"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

// ExtBackedTrieState is a wrapper around the Ext struct that provides a TrieState interface
// Note: Since the Ext struct is using a read-only backend, all actions are done on the overlayed changes
// and no changes are reflected on the backend
type ExtBackedTrieState[H runtime.Hash, Hasher runtime.Hasher[H], B statemachine.Backend[H, Hasher]] struct {
	ext         *overlayedchanges.Ext[H, Hasher, B]
	trieVersion storage.StateVersion
}

func NewExtBackedTrieState[H runtime.Hash, Hasher runtime.Hasher[H], B statemachine.Backend[H, Hasher]](
	ext *overlayedchanges.Ext[H, Hasher, B],
) *ExtBackedTrieState[H, Hasher, B] {
	return &ExtBackedTrieState[H, Hasher, B]{ext: ext}
}

func (t *ExtBackedTrieState[H, Hasher, B]) SetVersion(v trie.TrieLayout) {
	// TODO: when the migration is done,modify the interface to not set the version
	// instead receive it as parameter in the required methods
	t.trieVersion = storage.StateVersion(v)
}

func (t *ExtBackedTrieState[H, Hasher, B]) StartTransaction() {
	t.ext.StorageStartTransaction()
}

func (t *ExtBackedTrieState[H, Hasher, B]) RollbackTransaction() error {
	if err := t.ext.StorageRollbackTransaction(); err != nil {
		return ErrNoTransactionsToRollback
	}
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) CommitTransaction() error {
	if err := t.ext.StorageCommitTransaction(); err != nil {
		return ErrNoTransactionsToCommit
	}
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) Put(key, value []byte) error {
	t.ext.SetStorage(key, value)
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) Get(key []byte) []byte {
	return t.ext.Storage(key)
}

func (t *ExtBackedTrieState[H, Hasher, B]) Root() (common.Hash, error) {
	root := t.ext.StorageRoot(t.trieVersion)
	return common.NewHash(root), nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) Has(key []byte) bool {
	return t.ext.ExistsStorage(key)
}

func (t *ExtBackedTrieState[H, Hasher, B]) Delete(key []byte) error {
	t.ext.ClearStorage(key)
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) NextKey(key []byte) []byte {
	return t.ext.NextStorageKey(key)
}

func (t *ExtBackedTrieState[H, Hasher, B]) ClearPrefix(prefix []byte) error {
	_ = t.ext.ClearPrefix(prefix, nil, nil)
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) ClearPrefixLimit(
	prefix []byte,
	limit uint32,
) (loops uint32, deleted uint32, allDeleted bool, err error) {
	results := t.ext.ClearPrefix(prefix, &limit, nil)
	return results.Loops, results.Unique, results.Cursor == nil, nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) SetChildStorage(keyToChild, key, value []byte) error {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	t.ext.SetChildStorage(childInfo, key, value)
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) GetChildRoot(keyToChild []byte) (common.Hash, error) {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	root := t.ext.ChildStorageRoot(childInfo, t.trieVersion)
	return common.NewHash(root), nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) GetChildStorage(keyToChild, key []byte) ([]byte, error) {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	value := t.ext.ChildStorage(childInfo, key)
	return value, nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) DeleteChild(keyToChild []byte) error {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	t.ext.KillChildStorage(childInfo, nil, nil)
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) DeleteChildLimit(
	keyToChild []byte,
	limit *[]byte,
) (deleted uint32, allDeleted bool, err error) {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	var deleteLimit *uint32
	if limit != nil {
		leLimit := binary.LittleEndian.Uint32(*limit)
		deleteLimit = &leLimit
	}
	results := t.ext.KillChildStorage(childInfo, deleteLimit, nil)
	return results.Loops, results.Cursor == nil, nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) ClearChildStorage(keyToChild, key []byte) error {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	t.ext.ClearChildStorage(childInfo, key)
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) ClearPrefixInChild(keyToChild, prefix []byte) error {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	t.ext.ClearChildPrefix(childInfo, prefix, nil, nil)
	return nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) ClearPrefixInChildWithLimit(
	keyToChild,
	prefix []byte,
	limit uint32,
) (uint32, uint32, bool, error) {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	results := t.ext.ClearChildPrefix(childInfo, prefix, &limit, nil)
	return results.Loops, results.Unique, results.Cursor == nil, nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) GetChildNextKey(keyToChild, key []byte) ([]byte, error) {
	childInfo := storage.NewDefaultChildInfo(keyToChild)
	next := t.ext.NextChildStorageKey(childInfo, key)
	return next, nil
}

func (t *ExtBackedTrieState[H, Hasher, B]) LoadCode() []byte {
	return t.ext.Storage(common.CodeKey)
}

func (t *ExtBackedTrieState[H, Hasher, B]) LoadCodeHash() (common.Hash, error) {
	code := t.LoadCode()
	return common.Blake2bHash(code)
}

func (t *ExtBackedTrieState[H, Hasher, B]) Trie() trie.Trie {
	// TODO: remove this from the interface
	panic("not implemented")
}

func (t *ExtBackedTrieState[H, Hasher, B]) TrieEntries() map[string][]byte {
	// TODO: remove this from the interface
	panic("not implemented")
}

func (t *ExtBackedTrieState[H, Hasher, B]) GetChangedNodeHashes() (
	inserted,
	deleted map[common.Hash]struct{},
	err error,
) {
	panic("not implemented")
}
