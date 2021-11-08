// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

// GetKeysRequest represents the request to retrieve the keys of a child storage
type GetKeysRequest struct {
	Key    []byte
	Prefix []byte
	Hash   *common.Hash
}

// ChildStateStorageRequest holds json fields
type ChildStateStorageRequest struct {
	ChildStorageKey []byte       `json:"childStorageKey"`
	Key             []byte       `json:"key"`
	Hash            *common.Hash `json:"block"`
}

// GetStorageHash the request to get the entry child storage hash
type GetStorageHash struct {
	KeyChild []byte
	EntryKey []byte
	Hash     *common.Hash
}

// GetChildStorageRequest the request to get the entry child storage hash
type GetChildStorageRequest struct {
	KeyChild []byte
	EntryKey []byte
	Hash     *common.Hash
}

// ChildStateModule is the module responsible to implement all the childstate RPC calls
type ChildStateModule struct {
	storageAPI StorageAPI
	blockAPI   BlockAPI
}

// NewChildStateModule returns a new ChildStateModule
func NewChildStateModule(s StorageAPI, b BlockAPI) *ChildStateModule {
	return &ChildStateModule{
		storageAPI: s,
		blockAPI:   b,
	}
}

// GetKeys returns the keys from the specified child storage. The keys can also be filtered based on a prefix.
func (cs *ChildStateModule) GetKeys(_ *http.Request, req *GetKeysRequest, res *[]string) error {
	var hash common.Hash

	if req.Hash == nil {
		hash = cs.blockAPI.BestBlockHash()
	} else {
		hash = *req.Hash
	}

	stateRoot, err := cs.storageAPI.GetStateRootFromBlock(&hash)
	if err != nil {
		return err
	}

	trie, err := cs.storageAPI.GetStorageChild(stateRoot, req.Key)
	if err != nil {
		return err
	}

	keys := trie.GetKeysWithPrefix(req.Prefix)
	hexKeys := make([]string, len(keys))
	for idx, k := range keys {
		hexKeys[idx] = common.BytesToHex(k)
	}

	*res = hexKeys
	return nil
}

// GetStorageSize returns the size of a child storage entry.
func (cs *ChildStateModule) GetStorageSize(_ *http.Request, req *GetChildStorageRequest, res *uint64) error {
	var hash common.Hash

	if req.Hash == nil {
		hash = cs.blockAPI.BestBlockHash()
	} else {
		hash = *req.Hash
	}

	stateRoot, err := cs.storageAPI.GetStateRootFromBlock(&hash)
	if err != nil {
		return err
	}

	item, err := cs.storageAPI.GetStorageFromChild(stateRoot, req.KeyChild, req.EntryKey)
	if err != nil {
		return err
	}

	if item != nil {
		*res = uint64(len(item))
	}

	return nil
}

// GetStorageHash returns the hash of a child storage entry
func (cs *ChildStateModule) GetStorageHash(_ *http.Request, req *GetStorageHash, res *string) error {
	var hash common.Hash

	if req.Hash == nil {
		hash = cs.blockAPI.BestBlockHash()
	} else {
		hash = *req.Hash
	}

	stateRoot, err := cs.storageAPI.GetStateRootFromBlock(&hash)
	if err != nil {
		return err
	}

	item, err := cs.storageAPI.GetStorageFromChild(stateRoot, req.KeyChild, req.EntryKey)
	if err != nil {
		return err
	}

	if item != nil {
		*res = common.BytesToHash(item).String()
	}

	return nil
}

// GetStorage returns a child storage entry.
func (cs *ChildStateModule) GetStorage(_ *http.Request, req *ChildStateStorageRequest, res *StateStorageResponse) error {
	var (
		item []byte
		err  error
		hash common.Hash
	)

	if req.Hash == nil {
		hash = cs.blockAPI.BestBlockHash()
	} else {
		hash = *req.Hash
	}

	stateRoot, err := cs.storageAPI.GetStateRootFromBlock(&hash)
	if err != nil {
		return err
	}

	item, err = cs.storageAPI.GetStorageFromChild(stateRoot, req.ChildStorageKey, req.Key)
	if err != nil {
		return err
	}

	if len(item) > 0 {
		*res = StateStorageResponse(common.BytesToHex(item))
	}

	return nil
}
