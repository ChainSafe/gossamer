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
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// StateCallRequest holds json fields
type StateCallRequest struct {
	Method string       `json:"method"`
	Data   []byte       `json:"data"`
	Block  *common.Hash `json:"block"`
}

// StateChildStorageRequest holds json fields
type StateChildStorageRequest struct {
	ChildStorageKey []byte       `json:"childStorageKey"`
	Key             []byte       `json:"key"`
	Block           *common.Hash `json:"block"`
}

// StateStorageKeyRequest holds json fields
type StateStorageKeyRequest struct {
	Prefix   string       `json:"prefix"`
	Qty      uint32       `json:"qty"`
	AfterKey string       `json:"afterKey"`
	Block    *common.Hash `json:"block"`
}

// StateRuntimeMetadataQuery is a hash value
type StateRuntimeMetadataQuery struct {
	Bhash *common.Hash
}

// StateRuntimeVersionRequest is hash value
type StateRuntimeVersionRequest struct {
	Bhash *common.Hash
}

// StatePairRequest holds json field
type StatePairRequest struct {
	Prefix *string `validate:"required"`
	Bhash  *common.Hash
}

// StateStorageSizeRequest holds json field
type StateStorageSizeRequest struct {
	Key   string `validate:"required"`
	Bhash *common.Hash
}

// StateStorageHashRequest holds json field
type StateStorageHashRequest struct {
	Key   string `validate:"required"`
	Bhash *common.Hash
}

// StateStorageRequest holds json field
type StateStorageRequest struct {
	Key   string `validate:"required"`
	Bhash *common.Hash
}

// StateStorageQueryRangeRequest holds json fields
type StateStorageQueryRangeRequest struct {
	Keys       []*common.Hash `json:"keys" validate:"required"`
	StartBlock *common.Hash   `json:"startBlock" validate:"required"`
	Block      *common.Hash   `json:"block"`
}

// StateStorageKeysQuery field to store storage keys
type StateStorageKeysQuery [][]byte

// StateCallResponse holds json fields
type StateCallResponse []byte

// StateKeysResponse field to store the state keys
type StateKeysResponse [][]byte

// StateStorageDataResponse field to store data response
type StateStorageDataResponse string

// StateStorageHashResponse is a hash value
type StateStorageHashResponse string

// StateChildStorageResponse is a hash value
type StateChildStorageResponse string

// StateChildStorageSizeResponse is a unint value
type StateChildStorageSizeResponse uint64

// StateStorageSizeResponse the default size for response
type StateStorageSizeResponse uint64

// StateStorageResponse storage hash value
type StateStorageResponse string

// StatePairResponse is a key values
type StatePairResponse []interface{}

// StateStorageKeysResponse field for storage keys
type StateStorageKeysResponse []string

// StateMetadataResponse holds the metadata
//TODO: Determine actual type
type StateMetadataResponse string

// StorageChangeSetResponse is the struct that holds the block and changes
type StorageChangeSetResponse struct {
	Block   *common.Hash
	Changes []KeyValueOption
}

// KeyValueOption struct holds json fields
type KeyValueOption []byte

// StorageKey is the key for the storage
type StorageKey []byte

// StateRuntimeVersionResponse is the runtime version response
type StateRuntimeVersionResponse struct {
	SpecName           string        `json:"specName"`
	ImplName           string        `json:"implName"`
	AuthoringVersion   uint32        `json:"authoringVersion"`
	SpecVersion        uint32        `json:"specVersion"`
	ImplVersion        uint32        `json:"implVersion"`
	TransactionVersion uint32        `json:"transactionVersion"`
	Apis               []interface{} `json:"apis"`
}

// StateModule is an RPC module providing access to storage API points.
type StateModule struct {
	networkAPI NetworkAPI
	storageAPI StorageAPI
	coreAPI    CoreAPI
}

// NewStateModule creates a new State module.
func NewStateModule(net NetworkAPI, storage StorageAPI, core CoreAPI) *StateModule {
	return &StateModule{
		networkAPI: net,
		storageAPI: storage,
		coreAPI:    core,
	}
}

// GetPairs returns the keys with prefix, leave empty to get all the keys.
func (sm *StateModule) GetPairs(r *http.Request, req *StatePairRequest, res *StatePairResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	var (
		stateRootHash *common.Hash
		err           error
	)

	if req.Bhash != nil {
		stateRootHash, err = sm.storageAPI.GetStateRootFromBlock(req.Bhash)
		if err != nil {
			return err
		}
	}

	if req.Prefix == nil || *req.Prefix == "" || *req.Prefix == "0x" {
		pairs, err := sm.storageAPI.Entries(stateRootHash)
		if err != nil {
			return err
		}
		for k, v := range pairs {
			*res = append(*res, []string{"0x" + hex.EncodeToString([]byte(k)), "0x" + hex.EncodeToString(v)})
		}
	} else {
		// TODO this should return all keys with same prefix, currently only returning
		//  matches.  Implement when #837 is done.
		reqBytes, _ := common.HexToBytes(*req.Prefix)
		resI, err := sm.storageAPI.GetStorage(stateRootHash, reqBytes)
		if err != nil {
			return err
		}
		if resI != nil {
			*res = append(*res, []string{"0x" + hex.EncodeToString(reqBytes), "0x" + hex.EncodeToString(resI)})
		} else {
			*res = []interface{}{}
		}
	}

	return nil
}

// Call isn't implemented properly yet.
func (sm *StateModule) Call(r *http.Request, req *StateCallRequest, res *StateCallResponse) error {
	_ = sm.networkAPI
	_ = sm.storageAPI
	return nil
}

// GetChildKeys isn't implemented properly yet.
func (sm *StateModule) GetChildKeys(r *http.Request, req *StateChildStorageRequest, res *StateKeysResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return nil
}

// GetChildStorage isn't implemented properly yet.
func (sm *StateModule) GetChildStorage(r *http.Request, req *StateChildStorageRequest, res *StateStorageDataResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return nil
}

// GetChildStorageHash isn't implemented properly yet.
func (sm *StateModule) GetChildStorageHash(r *http.Request, req *StateChildStorageRequest, res *StateChildStorageResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return nil
}

// GetChildStorageSize isn't implemented properly yet.
func (sm *StateModule) GetChildStorageSize(r *http.Request, req *StateChildStorageRequest, res *StateChildStorageSizeResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return nil
}

// GetKeysPaged Returns the keys with prefix with pagination support.
func (sm *StateModule) GetKeysPaged(r *http.Request, req *StateStorageKeyRequest, res *StateStorageKeysResponse) error {
	if len(req.Prefix) == 0 {
		req.Prefix = "0x"
	}
	hPrefix, err := common.HexToBytes(req.Prefix)
	if err != nil {
		return err
	}
	keys, err := sm.storageAPI.GetKeysWithPrefix(req.Block, hPrefix)
	resCount := uint32(0)
	for _, k := range keys {
		fKey := fmt.Sprintf("0x%x", k)
		if strings.Compare(fKey, req.AfterKey) == 1 {
			// sm.storageAPI.Keys sorts keys in lexicographical order, so we know that keys where strings.Compare = 1
			//  are after the requested after key.
			if resCount >= req.Qty {
				break
			}
			*res = append(*res, fKey)
			resCount++
		}
	}
	return err
}

// GetMetadata calls runtime Metadata_metadata function
func (sm *StateModule) GetMetadata(r *http.Request, req *StateRuntimeMetadataQuery, res *StateMetadataResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	metadata, err := sm.coreAPI.GetMetadata(req.Bhash)
	if err != nil {
		return err
	}

	decoded, err := scale.Decode(metadata, []byte{})
	*res = StateMetadataResponse(common.BytesToHex(decoded.([]byte)))
	return err
}

// GetRuntimeVersion Get the runtime version at a given block.
//  If no block hash is provided, the latest version gets returned.
// TODO currently only returns latest version, add functionality to lookup runtime by block hash (see issue #834)
func (sm *StateModule) GetRuntimeVersion(r *http.Request, req *StateRuntimeVersionRequest, res *StateRuntimeVersionResponse) error {
	rtVersion, err := sm.coreAPI.GetRuntimeVersion(req.Bhash)
	if err != nil {
		return err
	}

	res.SpecName = string(rtVersion.SpecName())
	res.ImplName = string(rtVersion.ImplName())
	res.AuthoringVersion = rtVersion.AuthoringVersion()
	res.SpecVersion = rtVersion.SpecVersion()
	res.ImplVersion = rtVersion.ImplVersion()
	res.TransactionVersion = rtVersion.TransactionVersion()
	res.Apis = ConvertAPIs(rtVersion.APIItems())

	return nil
}

// GetStorage Returns a storage entry at a specific block's state. If not block hash is provided, the latest value is returned.
func (sm *StateModule) GetStorage(r *http.Request, req *StateStorageRequest, res *StateStorageResponse) error {
	var (
		item []byte
		err  error
	)

	reqBytes, _ := common.HexToBytes(req.Key) // no need to catch error here
	if req.Bhash != nil {
		item, err = sm.storageAPI.GetStorageByBlockHash(*req.Bhash, reqBytes)
		if err != nil {
			return err
		}
	} else {
		item, err = sm.storageAPI.GetStorage(nil, reqBytes)
		if err != nil {
			return err
		}
	}

	if len(item) > 0 {
		*res = StateStorageResponse(common.BytesToHex(item))
	}

	return nil
}

// GetStorageHash returns the hash of a storage entry at a block's state.
//  If no block hash is provided, the latest value is returned.
//  TODO implement change storage trie so that block hash parameter works (See issue #834)
func (sm *StateModule) GetStorageHash(r *http.Request, req *StateStorageHashRequest, res *StateStorageHashResponse) error {
	var (
		item []byte
		err  error
	)

	reqBytes, _ := common.HexToBytes(req.Key)

	if req.Bhash != nil {
		item, err = sm.storageAPI.GetStorageByBlockHash(*req.Bhash, reqBytes)
		if err != nil {
			return err
		}
	} else {
		item, err = sm.storageAPI.GetStorage(nil, reqBytes)
		if err != nil {
			return err
		}
	}

	if len(item) > 0 {
		*res = StateStorageHashResponse(common.BytesToHash(item).String())
	}

	return nil
}

// GetStorageSize returns the size of a storage entry at a block's state.
//  If no block hash is provided, the latest value is used.
// TODO implement change storage trie so that block hash parameter works (See issue #834)
func (sm *StateModule) GetStorageSize(r *http.Request, req *StateStorageSizeRequest, res *StateStorageSizeResponse) error {
	var (
		item []byte
		err  error
	)

	reqBytes, _ := common.HexToBytes(req.Key)

	if req.Bhash != nil {
		item, err = sm.storageAPI.GetStorageByBlockHash(*req.Bhash, reqBytes)
		if err != nil {
			return err
		}
	} else {
		item, err = sm.storageAPI.GetStorage(nil, reqBytes)
		if err != nil {
			return err
		}
	}

	if len(item) > 0 {
		*res = StateStorageSizeResponse((uint64)(len(item)))
	}

	return nil
}

// QueryStorage isn't implemented properly yet.
func (sm *StateModule) QueryStorage(r *http.Request, req *StateStorageQueryRangeRequest, res *StorageChangeSetResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return nil
}

// SubscribeRuntimeVersion isn't implemented properly yet.
// TODO make this actually a subscription that pushes data
func (sm *StateModule) SubscribeRuntimeVersion(r *http.Request, req *StateStorageQueryRangeRequest, res *StateRuntimeVersionResponse) error {
	// TODO implement change storage trie so that block hash parameter works (See issue #834)
	return sm.GetRuntimeVersion(r, nil, res)
}

// SubscribeStorage Storage subscription. If storage keys are specified, it creates a message for each block which
//  changes the specified storage keys. If none are specified, then it creates a message for every block.
//  This endpoint communicates over the Websocket protocol, but this func should remain here so it's added to rpc_methods list
func (sm *StateModule) SubscribeStorage(r *http.Request, req *StateStorageQueryRangeRequest, res *StorageChangeSetResponse) error {
	return nil
}

// ConvertAPIs runtime.APIItems to []interface
func ConvertAPIs(in []*runtime.APIItem) []interface{} {
	ret := make([]interface{}, 0)
	for _, item := range in {
		encStr := hex.EncodeToString(item.Name[:])
		ret = append(ret, []interface{}{"0x" + encStr, item.Ver})
	}
	return ret
}
