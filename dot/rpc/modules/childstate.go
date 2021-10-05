// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
func (cs *ChildStateModule) GetStorage(r *http.Request, req *ChildStateStorageRequest, res *StateStorageResponse) error {
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

// GetChildKeys isn't implemented properly yet.
func (cs *ChildStateModule) GetChildKeys(_ *http.Request, _ *ChildStateStorageRequest, _ *StateKeysResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return nil
}

// GetChildStorageHash isn't implemented properly yet.
func (cs *ChildStateModule) GetChildStorageHash(_ *http.Request, _ *ChildStateStorageRequest, _ *StateChildStorageResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return nil
}

// GetChildStorageSize isn't implemented properly yet.
func (cs *ChildStateModule) GetChildStorageSize(_ *http.Request, _ *ChildStateStorageRequest, _ *StateChildStorageSizeResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return nil
}
