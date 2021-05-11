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
	//"bytes"
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
	if req.Prefix == "" {
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

	// if bytes.Equal(reqBytes, common.MustHexToBytes("0xcec5070d609dd3497f72bde07fc96ba0e0cdd062e6eaf24295ad4ccfc41d4609")) {
	// 	*res = StateStorageResponse("0x183887050ecff59f58658b3df63a16d03a00f92890f1517f48c2f6ccd215e5450edea6f4a727d3b2399275d6ee8817881f10597471dc1d27f144295ad6fb933c7afa3437b10f6e7af8f31362df3a179b991a8c56313d1bcd6307a4d0c734c1ae316a103df5c5131813fa77ba4f8be88b2d2b4a47323d2011c9d987615f067e9e7856f0bb1f6307e043be568014eb4062a9bca4a255f39ed0be9205ee97c93b4b6ee0a3e2de329a70e2763438a1a757bf6dab945dcaededc7455a7fcfae83def07bdc9974cdb3cebfac4d31333c30865ff66c35c1bf898df5c5dd2924d3280e720148b623941c2a4d41cf25ef495408690fc853f777192498c0922eab1e9df4f061d2419bc8835493ac89eb09d5985281f5dff4bc6c7a7ea988fd23af05f301580a3ccde029459535c8bda2aa6fc2d97af3880409010bbc05a15f8d42bce8f0176d3c7d33a7ca6e152bcceb20a75bf67dca553cfe1fa0546decfdab25177765ae078acc4f2aa64faa0c97ea1f8702fbdf1843694734eee4d7c65c5605c2f81271485809fd84af6483070acbb92378e3498dbc02fb47f8e97f006bb83f60d7b2b15df72daf2e560e4f0f22fb5cbb04ad1d7fee850aab238fd014c178769e7e3a9b84ccb6bef60defc30724545d57440394ed1c71ea7ee6d880ed0e79871a05b5e4065e5ab03e0bc62a8fd3fded0b09ac04c6192796873b38abceffdbd1548f35f61aa25cc78808d9ffb966aaa53c3c399cff7ea0b409dc8b42908b9f2da6d34c352514f13a09505d4014b468c1d3e394002832d9edc35dbbae1a7a6dc96025d47d5b88ee494d719d68a18aade04903839ea37b6be99552ceceb530674b237afa91661c151c11cb72334d26d70769e3af7bbff3801a4e2dca2b09b7cce0af8dd813075e67b64cf07d4d258a47df63835121423551712844f5b67de68e36bb9a21e1276c694dbad86b8de9c1c9947e536b3391b77caaca86a23195a2b111b24b0d516450e91d8b60377c58f1e8dfb6236dece92917f1b4ee67d2787ab090c5f8d2200f74b919094e1fca66ed767766aa0a91025b6a8b955bb970912900ad4e413ea93682104c22c383925323bf209d771dec6e1388285abe22c22d50de968467e0bb6c680d278213f908658a49a1025a7f466c197e8fb6fabb5e62220a7bd75f860cab6236877b05370265640c133fec07e64d7ca823db1dc56f2d3584b3d7c0f161583cfc25dae5d649a0d4f2775656419f2c9a4318584694bb60c66e8d0c8b96f5029ab54fd64223ac5dd0c547efbf0015944d1bcf8f4ca721716d8922fc940c9a610a7d2ed5da6a62c32ef4477bef2a1ba05c5feea57ebd44516a8257dcf9a3b67be240d12c7ad07bb0e7785ee6837095ddeebb7aef84d6ed7ea87da197805b343a8e59368700ea89e2bf8922cc9e4b86d6651d1c689a0d57813f9768dbaadecf716c52d02d95c30aa567fda284acf25025ca7470f0b0c516ddf94475a1807c4d25b09bbdce34c5bff2f9f212118c05296db12854ecd09ed0eb0dc7714c9337ce29cec7e2d5e28925ae9f906e5ebf1c81adcc7e524751273a73a278f472d863f5324c0831fc73ca4ae4d46cf82e74ad01549973d132795c579d40eed490cbb01524")
	// }
	fmt.Printf("GetStorage key=%x\t val=%s\n", reqBytes, *res)

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
