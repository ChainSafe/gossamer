package modules

import (
	"github.com/ChainSafe/gossamer/common"
)

type StateCallRequest struct {
	Method string      `json:"method"`
	Data   []byte      `json:"data"`
	Block  common.Hash `json:"block"`
}

type StateChildStorageRequest struct {
	ChildStorageKey []byte      `json:"childStorageKey"`
	Key             []byte      `json:"key"`
	Block           common.Hash `json:"block"`
}

type StateStorageQueryRequest struct {
	Key   []byte      `json:"key"`
	Block common.Hash `json:"block"`
}

type StateBlockHashQuery common.Hash

type StateStorageQueryRangeRequest struct {
	Keys []byte `json:"keys"`
	StartBlock common.Hash `json:"startBlock"`
	EndBlock common.Hash `json:"block"`
}

type StateCallResponse []byte

type StateChildKeysResponse [][]byte

type StateGetStorageResponse []byte

type StateStorageHashResponse common.Hash

type StateStorageSizeResonse uint64

type StateStorageKeysReponse [][]byte

// TODO: Determine actual type
type StateMetadataResponse []byte

type StateRuntimeVersionResponse string

// TODO: SubscribeRuntimeVersion and SubscribeStorage