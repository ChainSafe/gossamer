// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

//StateGetReadProofRequest json fields
type StateGetReadProofRequest struct {
	Keys []string
	Hash common.Hash
}

// StateCallRequest holds json fields
type StateCallRequest struct {
	Method string       `json:"method"`
	Data   []byte       `json:"data"`
	Block  *common.Hash `json:"block"`
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
	Keys       []string    `json:"keys" validate:"required"`
	StartBlock common.Hash `json:"startBlock" validate:"required"`
	EndBlock   common.Hash `json:"block"`
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
type StateMetadataResponse string

//StateGetReadProofResponse holds the response format
type StateGetReadProofResponse struct {
	At    common.Hash `json:"at"`
	Proof []string    `json:"proof"`
}

// StorageChangeSetResponse is the struct that holds the block and changes
type StorageChangeSetResponse struct {
	Block   *common.Hash `json:"block"`
	Changes [][]string   `json:"changes"`
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
func (sm *StateModule) GetPairs(_ *http.Request, req *StatePairRequest, res *StatePairResponse) error {
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
			*res = append(*res, []string{common.BytesToHex([]byte(k)), common.BytesToHex(v)})
		}

		return nil
	}

	reqBytes, _ := common.HexToBytes(*req.Prefix)
	keys, err := sm.storageAPI.GetKeysWithPrefix(stateRootHash, reqBytes)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		*res = []interface{}{}
		return nil
	}

	*res = make([]interface{}, len(keys))
	for i, key := range keys {
		val, err := sm.storageAPI.GetStorage(stateRootHash, key)
		if err != nil {
			return err
		}

		(*res)[i] = []string{common.BytesToHex(key), common.BytesToHex(val)}
	}

	return nil
}

// Call isn't implemented properly yet.
func (sm *StateModule) Call(_ *http.Request, _ *StateCallRequest, _ *StateCallResponse) error {
	_ = sm.networkAPI
	_ = sm.storageAPI
	return nil
}

// GetKeysPaged Returns the keys with prefix with pagination support.
func (sm *StateModule) GetKeysPaged(_ *http.Request, req *StateStorageKeyRequest, res *StateStorageKeysResponse) error {
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
func (sm *StateModule) GetMetadata(_ *http.Request, req *StateRuntimeMetadataQuery, res *StateMetadataResponse) error {
	metadata, err := sm.coreAPI.GetMetadata(req.Bhash)
	if err != nil {
		return err
	}

	var decoded []byte
	err = scale.Unmarshal(metadata, &decoded)
	*res = StateMetadataResponse(common.BytesToHex(decoded))
	return err
}

// GetReadProof returns the proof to the received storage keys
func (sm *StateModule) GetReadProof(_ *http.Request, req *StateGetReadProofRequest, res *StateGetReadProofResponse) error {
	keys := make([][]byte, len(req.Keys))
	for i, hexKey := range req.Keys {
		bKey, err := common.HexToBytes(hexKey)
		if err != nil {
			return err
		}

		keys[i] = bKey
	}

	block, proofs, err := sm.coreAPI.GetReadProofAt(req.Hash, keys)
	if err != nil {
		return err
	}

	var decProof []string
	for _, p := range proofs {
		decProof = append(decProof, common.BytesToHex(p))
	}

	*res = StateGetReadProofResponse{
		At:    block,
		Proof: decProof,
	}

	return nil
}

// GetRuntimeVersion Get the runtime version at a given block.
//  If no block hash is provided, the latest version gets returned.
func (sm *StateModule) GetRuntimeVersion(_ *http.Request, req *StateRuntimeVersionRequest, res *StateRuntimeVersionResponse) error {
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
func (sm *StateModule) GetStorage(_ *http.Request, req *StateStorageRequest, res *StateStorageResponse) error {
	var (
		item []byte
		err  error
	)

	reqBytes, _ := common.HexToBytes(req.Key) // no need to catch error here

	if req.Bhash != nil {
		item, err = sm.storageAPI.GetStorageByBlockHash(req.Bhash, reqBytes)
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
func (sm *StateModule) GetStorageHash(_ *http.Request, req *StateStorageHashRequest, res *StateStorageHashResponse) error {
	var (
		item []byte
		err  error
	)

	reqBytes, _ := common.HexToBytes(req.Key)

	if req.Bhash != nil {
		item, err = sm.storageAPI.GetStorageByBlockHash(req.Bhash, reqBytes)
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
func (sm *StateModule) GetStorageSize(_ *http.Request, req *StateStorageSizeRequest, res *StateStorageSizeResponse) error {
	var (
		item []byte
		err  error
	)

	reqBytes, _ := common.HexToBytes(req.Key)

	if req.Bhash != nil {
		item, err = sm.storageAPI.GetStorageByBlockHash(req.Bhash, reqBytes)
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
func (sm *StateModule) QueryStorage(_ *http.Request, req *StateStorageQueryRangeRequest, res *[]StorageChangeSetResponse) error {
	if req.StartBlock.IsEmpty() {
		return errors.New("the start block hash cannot be an empty value")
	}

	changesByBlock, err := sm.coreAPI.QueryStorage(req.StartBlock, req.EndBlock, req.Keys...)
	if err != nil {
		return err
	}

	response := make([]StorageChangeSetResponse, 0, len(changesByBlock))

	for block, c := range changesByBlock {
		var changes [][]string

		for key, value := range c {
			changes = append(changes, []string{key, value})
		}

		response = append(response, StorageChangeSetResponse{
			Block:   &block,
			Changes: changes,
		})
	}

	*res = response
	return nil
}

// SubscribeRuntimeVersion initialised a runtime version subscription and returns the current version
// See dot/rpc/subscription
func (sm *StateModule) SubscribeRuntimeVersion(r *http.Request, _ *StateStorageQueryRangeRequest, res *StateRuntimeVersionResponse) error {
	return sm.GetRuntimeVersion(r, nil, res)
}

// SubscribeStorage Storage subscription. If storage keys are specified, it creates a message for each block which
//  changes the specified storage keys. If none are specified, then it creates a message for every block.
//  This endpoint communicates over the Websocket protocol, but this func should remain here so it's added to rpc_methods list
func (*StateModule) SubscribeStorage(_ *http.Request, _ *StateStorageQueryRangeRequest, _ *StorageChangeSetResponse) error {
	return nil
}

// ConvertAPIs runtime.APIItems to []interface
func ConvertAPIs(in []runtime.APIItem) []interface{} {
	ret := make([]interface{}, 0)
	for _, item := range in {
		encStr := hex.EncodeToString(item.Name[:])
		ret = append(ret, []interface{}{"0x" + encStr, item.Ver})
	}
	return ret
}
