// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

// StateGetReadProofRequest json fields
type StateGetReadProofRequest struct {
	Keys []string
	Hash common.Hash
}

// StateCallRequest holds json fields
type StateCallRequest struct {
	Method string       `json:"method"`
	Params string       `json:"params"`
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

// StateStorageQueryAtRequest holds json fields
type StateStorageQueryAtRequest struct {
	Keys []string    `json:"keys" validate:"required"`
	At   common.Hash `json:"at"`
}

type StateTrieAtRequest struct {
	At *common.Hash `json:"at"`
}

// StateStorageKeysQuery field to store storage keys
type StateStorageKeysQuery [][]byte

// StateCallResponse holds the result of the call
type StateCallResponse string

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

type StateTrieResponse []string

// StateStorageKeysResponse field for storage keys
type StateStorageKeysResponse []string

// StateMetadataResponse holds the metadata
type StateMetadataResponse string

// StateGetReadProofResponse holds the response format
type StateGetReadProofResponse struct {
	At    common.Hash `json:"at"`
	Proof []string    `json:"proof"`
}

// StorageChangeSetResponse is the struct that holds the block and changes
type StorageChangeSetResponse struct {
	Block *common.Hash `json:"block"`
	// Changes is a slice of arrays of string pointers instead of just strings
	// so that the JSON encoder can handle nil values as NULL instead of empty
	// strings.
	Changes [][2]*string `json:"changes"`
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

// NewStateRuntimeVersionResponse converts a runtime.Version to a
// StateRuntimeVersionResponse struct.
func NewStateRuntimeVersionResponse(runtimeVersion runtime.Version) (
	response StateRuntimeVersionResponse) {
	apisResponse := make([]interface{}, len(runtimeVersion.APIItems))
	for i, apiItem := range runtimeVersion.APIItems {
		hexItemName := hex.EncodeToString(apiItem.Name[:])
		apisResponse[i] = []interface{}{
			"0x" + hexItemName,
			apiItem.Ver,
		}
	}

	return StateRuntimeVersionResponse{
		SpecName:           string(runtimeVersion.SpecName),
		ImplName:           string(runtimeVersion.ImplName),
		AuthoringVersion:   runtimeVersion.AuthoringVersion,
		SpecVersion:        runtimeVersion.SpecVersion,
		ImplVersion:        runtimeVersion.ImplVersion,
		TransactionVersion: runtimeVersion.TransactionVersion,
		Apis:               apisResponse,
	}
}

// StateModule is an RPC module providing access to storage API points.
type StateModule struct {
	networkAPI NetworkAPI
	storageAPI StorageAPI
	coreAPI    CoreAPI
	blockAPI   BlockAPI
}

// NewStateModule creates a new State module.
func NewStateModule(net NetworkAPI, storage StorageAPI, core CoreAPI, blockAPI BlockAPI) *StateModule {
	return &StateModule{
		networkAPI: net,
		storageAPI: storage,
		coreAPI:    core,
		blockAPI:   blockAPI,
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

	reqBytes, err := common.HexToBytes(*req.Prefix)
	if err != nil {
		return fmt.Errorf("cannot convert hex prefix %s to bytes: %w", *req.Prefix, err)
	}
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

// Trie RPC method returns a list of scale encoded trie.Entry{Key byte, Value byte} representing
// all the entries in a trie for a block hash, if no block hash is given then it uses the best block hash
func (sm *StateModule) Trie(_ *http.Request, req *StateTrieAtRequest, res *StateTrieResponse) error {
	var blockHash common.Hash

	if req.At != nil {
		blockHash = *req.At
	} else {
		blockHash = sm.blockAPI.BestBlockHash()
	}

	blockHeader, err := sm.blockAPI.GetHeader(blockHash)
	if err != nil {
		return fmt.Errorf("getting header: %w", err)
	}

	entries, err := sm.storageAPI.Entries(&blockHeader.StateRoot)
	if err != nil {
		return fmt.Errorf("getting entries: %w", err)
	}

	entriesArr := make([]string, 0, len(entries))
	for key, value := range entries {
		entry := trie.Entry{
			Key:   []byte(key),
			Value: value,
		}

		encodedEntry, err := scale.Marshal(entry)
		if err != nil {
			return fmt.Errorf("scale encoding entry: %w", err)
		}

		entriesArr = append(entriesArr, common.BytesToHex(encodedEntry))
	}

	*res = entriesArr
	return nil
}

// Call makes a call to the runtime.
func (sm *StateModule) Call(_ *http.Request, req *StateCallRequest, res *StateCallResponse) error {
	var blockHash common.Hash
	if req.Block == nil {
		blockHash = sm.blockAPI.BestBlockHash()
	} else {
		blockHash = *req.Block
	}

	rt, err := sm.blockAPI.GetRuntime(blockHash)
	if err != nil {
		return fmt.Errorf("get runtime: %w", err)
	}

	request, err := common.HexToBytes(req.Params)
	if err != nil {
		return fmt.Errorf("convert hex to bytes: %w", err)
	}

	response, err := rt.Exec(req.Method, request)
	if err != nil {
		return fmt.Errorf("runtime exec: %w", err)
	}

	*res = StateCallResponse(common.BytesToHex(response))
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
	if err != nil {
		return fmt.Errorf("cannot get keys with prefix %s: %w", hPrefix, err)
	}
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
func (sm *StateModule) GetReadProof(
	_ *http.Request, req *StateGetReadProofRequest, res *StateGetReadProofResponse) error {
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
// If no block hash is provided, the latest version gets returned.
func (sm *StateModule) GetRuntimeVersion(
	_ *http.Request, req *StateRuntimeVersionRequest, res *StateRuntimeVersionResponse) error {
	rtVersion, err := sm.coreAPI.GetRuntimeVersion(req.Bhash)
	if err != nil {
		return err
	}

	*res = NewStateRuntimeVersionResponse(rtVersion)
	return nil
}

// GetStorage Returns a storage entry at a specific block's state.
// If not block hash is provided, the latest value is returned.
func (sm *StateModule) GetStorage(
	_ *http.Request, req *StateStorageRequest, res *StateStorageResponse) error {
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

// GetStorageHash returns the blake2b hash of a storage entry at a block's state.
// If no block hash is provided, the latest value is returned.
func (sm *StateModule) GetStorageHash(
	_ *http.Request, req *StateStorageHashRequest, res *StateStorageHashResponse) error {
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

	hash, err := common.Blake2bHash(item)
	if err != nil {
		return err
	}

	*res = StateStorageHashResponse(hash.String())
	return nil
}

// GetStorageSize returns the size of a storage entry at a block's state.
// If no block hash is provided, the latest value is used.
func (sm *StateModule) GetStorageSize(
	_ *http.Request, req *StateStorageSizeRequest, res *StateStorageSizeResponse) error {
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
		*res = StateStorageSizeResponse(uint64(len(item)))
	}

	return nil
}

// QueryStorage queries historical storage entries (by key) starting from a given request start block
// and until a given end block, or until the best block if the given end block is nil.
func (sm *StateModule) QueryStorage(
	_ *http.Request, req *StateStorageQueryRangeRequest, res *[]StorageChangeSetResponse) error {
	if req.StartBlock.IsEmpty() {
		return ErrStartBlockHashEmpty
	}

	startBlock, err := sm.blockAPI.GetBlockByHash(req.StartBlock)
	if err != nil {
		return err
	}

	startBlockNumber := startBlock.Header.Number

	endBlockHash := req.EndBlock
	if req.EndBlock.IsEmpty() {
		endBlockHash = sm.blockAPI.BestBlockHash()
	}
	endBlock, err := sm.blockAPI.GetBlockByHash(endBlockHash)
	if err != nil {
		return fmt.Errorf("getting block by hash: %w", err)
	}
	endBlockNumber := endBlock.Header.Number

	response := make([]StorageChangeSetResponse, 0, endBlockNumber-startBlockNumber)
	lastValue := make([]*string, len(req.Keys))

	for i := startBlockNumber; i <= endBlockNumber; i++ {
		blockHash, err := sm.blockAPI.GetHashByNumber(i)
		if err != nil {
			return fmt.Errorf("cannot get hash by number: %w", err)
		}
		changes := make([][2]*string, 0, len(req.Keys))

		for j, key := range req.Keys {
			value, err := sm.storageAPI.GetStorageByBlockHash(&blockHash, common.MustHexToBytes(key))
			if err != nil {
				return fmt.Errorf("getting value by block hash: %w", err)
			}
			var hexValue *string
			if len(value) > 0 {
				hexValue = stringPtr(common.BytesToHex(value))
			} else if value != nil { // empty byte slice value
				hexValue = stringPtr("0x")
			}

			differentValueEncountered := i == startBlockNumber ||
				lastValue[j] == nil && hexValue != nil ||
				lastValue[j] != nil && hexValue == nil ||
				lastValue[j] != nil && *lastValue[j] != *hexValue
			if differentValueEncountered {
				changes = append(changes, [2]*string{stringPtr(key), hexValue})
				lastValue[j] = hexValue
			}

		}

		response = append(response, StorageChangeSetResponse{
			Block:   &blockHash,
			Changes: changes,
		})
	}

	*res = response
	return nil
}

// QueryStorageAt queries historical storage entries (by key) at the block hash given or
// the best block if the given block hash is nil
func (sm *StateModule) QueryStorageAt(
	_ *http.Request, request *StateStorageQueryAtRequest, response *[]StorageChangeSetResponse) error {
	atBlockHash := request.At
	if atBlockHash.IsEmpty() {
		atBlockHash = sm.blockAPI.BestBlockHash()
	}

	changes := make([][2]*string, len(request.Keys))

	for i, key := range request.Keys {
		value, err := sm.storageAPI.GetStorageByBlockHash(&atBlockHash, common.MustHexToBytes(key))
		if err != nil {
			return fmt.Errorf("getting value by block hash: %w", err)
		}
		var hexValue *string
		if len(value) > 0 {
			hexValue = stringPtr(common.BytesToHex(value))
		} else if value != nil { // empty byte slice value
			hexValue = stringPtr("0x")
		}

		changes[i] = [2]*string{stringPtr(key), hexValue}
	}

	*response = []StorageChangeSetResponse{{
		Block:   &atBlockHash,
		Changes: changes,
	}}

	return nil
}

func stringPtr(s string) *string { return &s }

// SubscribeRuntimeVersion initialised a runtime version subscription and returns the current version
// See dot/rpc/subscription
func (sm *StateModule) SubscribeRuntimeVersion(
	r *http.Request, _ *StateStorageQueryRangeRequest, res *StateRuntimeVersionResponse) error {
	return sm.GetRuntimeVersion(r, nil, res)
}

// SubscribeStorage Storage subscription. If storage keys are specified, it creates a message for each block which
// changes the specified storage keys. If none are specified, then it creates a message for every block.
// This endpoint communicates over the Websocket protocol, but this func should remain here so it's
// added to rpc_methods list
func (*StateModule) SubscribeStorage(
	_ *http.Request, _ *StateStorageQueryRangeRequest, _ *StorageChangeSetResponse) error {
	return nil
}
