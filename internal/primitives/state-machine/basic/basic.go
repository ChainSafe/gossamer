// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package basic

import (
	"bytes"
	"iter"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/externalities"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/tidwall/btree"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "statemachine/basic"))

type BasicExternalities struct {
	overlay    overlayedchanges.OverlayedChanges[hash.H256, runtime.BlakeTwo256]
	extensions externalities.Extensions
}

func NewBasicExternalities() *BasicExternalities {
	return &BasicExternalities{
		overlay:    *overlayedchanges.NewOverlayedChanges[hash.H256, runtime.BlakeTwo256](),
		extensions: externalities.NewExtensions(),
	}
}

func (be *BasicExternalities) SetOffchainStorage(key []byte, value []byte) {}

func (be *BasicExternalities) Storage(key []byte) []byte {
	if value, ok := be.overlay.Storage(key); ok {
		return value
	}

	return nil
}

func (be *BasicExternalities) StorageHash(key []byte) []byte {
	value := be.Storage(key)
	if value == nil {
		return nil
	}

	return scale.MustMarshal(runtime.BlakeTwo256{}.Hash(value))
}

func (be *BasicExternalities) ChildStorageHash(childInfo storage.ChildInfo, key []byte) []byte {
	value := be.ChildStorage(childInfo, key)
	if value == nil {
		return nil
	}

	return scale.MustMarshal(runtime.BlakeTwo256{}.Hash(value))
}

func (be *BasicExternalities) ChildStorage(childInfo storage.ChildInfo, key []byte) []byte {
	if value, ok := be.overlay.ChildStorage(childInfo, key); ok {
		return value
	}

	return nil
}

func (be *BasicExternalities) SetStorage(key []byte, value []byte) {
	be.PlaceStorage(key, value)
}

func (be *BasicExternalities) SetChildStorage(childInfo storage.ChildInfo, key []byte, value []byte) {
	be.PlaceChildStorage(childInfo, key, value)
}

func (be *BasicExternalities) ClearStorage(key []byte) {
	be.PlaceStorage(key, nil)
}

func (be *BasicExternalities) ClearChildStorage(childInfo storage.ChildInfo, key []byte) {
	be.PlaceChildStorage(childInfo, key, nil)
}

func (be *BasicExternalities) ExistsStorage(key []byte) bool {
	return be.Storage(key) != nil
}

func (be *BasicExternalities) ExistsChildStorage(childInfo storage.ChildInfo, key []byte) bool {
	return be.ChildStorage(childInfo, key) != nil
}

func (be *BasicExternalities) NextStorageKey(key []byte) []byte {
	next, _ := iter.Pull2(be.overlay.IterAfter(key))
	if nextKey, _, has := next(); has {
		return nextKey
	}

	return nil
}

func (be *BasicExternalities) NextChildStorageKey(childInfo storage.ChildInfo, key []byte) []byte {
	next, _ := iter.Pull2(be.overlay.ChildIterAfter(overlayedchanges.StorageKey(childInfo.StorageKey()), key))
	if nextKey, _, has := next(); has {
		return nextKey
	}

	return nil
}

func (be *BasicExternalities) KillChildStorage(
	childInfo storage.ChildInfo,
	maybeLimit *uint32,
	maybeCursor []byte,
) externalities.MultiRemovalResults {
	count := be.overlay.ClearChildStorage(childInfo)
	return externalities.MultiRemovalResults{Cursor: nil, Backend: count, Unique: count, Loops: count}
}

func (be *BasicExternalities) PlaceStorage(key []byte, value []byte) {
	if keys.IsChildStorageKey(key) {
		logger.Warn("refuse to set child storage key via main storage")
		return
	}

	be.overlay.SetStorage(key, value)
}

func (be *BasicExternalities) PlaceChildStorage(
	childInfo storage.ChildInfo,
	key []byte,
	value []byte,
) {
	be.overlay.SetChildStorage(childInfo, key, value)
}

func (be *BasicExternalities) ClearPrefix(
	prefix []byte,
	limit *uint32,
	cursor []byte,
) externalities.MultiRemovalResults {
	if keys.IsChildStorageKey(prefix) {
		logger.Warn("refuse to clear prefix that is part of child storage key via main storage")
		return externalities.MultiRemovalResults{Cursor: prefix, Backend: 0, Unique: 0, Loops: 0}
	}

	count := be.overlay.ClearPrefix(prefix)
	return externalities.MultiRemovalResults{Cursor: nil, Backend: count, Unique: count, Loops: count}
}

func (be *BasicExternalities) ClearChildPrefix(
	childInfo storage.ChildInfo,
	prefix []byte,
	limit *uint32,
	cursor []byte,
) externalities.MultiRemovalResults {
	count := be.overlay.ClearChildPrefix(childInfo, prefix)
	return externalities.MultiRemovalResults{Cursor: nil, Backend: count, Unique: count, Loops: count}
}

func (be *BasicExternalities) StorageAppend(key []byte, element []byte) {
	be.overlay.AppendStorage(key, element, func() statemachine.StorageValue { return nil })
}

func (be *BasicExternalities) StorageRoot(stateVersion storage.StateVersion) []byte {
	top := btree.Map[string, statemachine.StorageValue]{}

	for k, v := range be.overlay.Changes() {
		if v.Value() != nil {
			top.Set(string(k), v.Value())
		}
	}

	// Single child trie implementation currently allows using the same child
	// empty root for all child trie. Using null storage key until multiple
	// type of child trie support.
	emptyHash := trie.EmptyChildTrieRoot[hash.H256, runtime.BlakeTwo256]()

	for _, childInfo := range be.overlay.Children() {
		childRoot := be.ChildStorageRoot(childInfo, stateVersion)
		if bytes.Equal(emptyHash.Bytes(), childRoot) {
			top.Delete(string(childInfo.PrefixedStorageKey()))
		} else {
			top.Set(string(childInfo.PrefixedStorageKey()), childRoot)
		}
	}

	iter := func(yield func(string, []byte) bool) {
		top.Scan(func(k string, v statemachine.StorageValue) bool {
			return yield(k, []byte(v))
		})
	}

	return stateVersion.TrieLayout().TrieRoot(iter).Bytes()
}

func (be *BasicExternalities) ChildStorageRoot(
	childInfo storage.ChildInfo,
	stateVersion storage.StateVersion,
) []byte {
	data, childInfo := be.overlay.ChildChanges(overlayedchanges.StorageKey(childInfo.StorageKey()))

	var rootHash hash.H256

	if childInfo != nil && data != nil {
		delta := make([]statemachine.Delta, 0)
		for k, v := range data {
			delta = append(delta, statemachine.Delta{Key: k, Value: v.Value()})
		}

		backend := statemachine.NewMemoryDBTrieBackend[hash.H256, runtime.BlakeTwo256]()
		backend.ChildStorageRoot(childInfo, delta, stateVersion)

		panic("not implemented")
	} else {
		rootHash = trie.EmptyChildTrieRoot[hash.H256, runtime.BlakeTwo256]()
	}

	return scale.MustMarshal(rootHash)
}

func (be *BasicExternalities) StorageStartTransaction() {
	be.overlay.StartTransaction()
}

func (be *BasicExternalities) StorageRollbackTransaction() error {
	return be.overlay.RollbackTransaction()
}

func (be *BasicExternalities) StorageCommitTransaction() error {
	return be.overlay.CommitTransaction()
}

func (be *BasicExternalities) StorageIndexTransaction(index uint32, hash []byte, size uint32) {
	panic("not implemented StorageIndexTransaction")
}

func (be *BasicExternalities) StorageRenewTransactionIndex(index uint32, hash []byte) {
	panic("not implemented StorageRenewTransactionIndex")
}
