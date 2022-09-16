// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

// PaymentQueryInfoRequest represents the request to get the fee of an extrinsic in a given block
type PaymentQueryInfoRequest struct {
	// hex SCALE encoded extrinsic
	Ext string
	// hex optional block hash indicating the state
	Hash *common.Hash
}

// PaymentQueryInfoResponse holds the response fields to the query info RPC method
type PaymentQueryInfoResponse struct {
	Weight     uint64 `json:"weight"`
	Class      int    `json:"class"`
	PartialFee string `json:"partialFee"`
}

// PaymentModule holds all the RPC implementation of polkadot payment rpc api
type PaymentModule struct {
	blockAPI   BlockAPI
	storageAPI StorageAPI
}

// NewPaymentModule returns a pointer to PaymentModule
func NewPaymentModule(blockAPI BlockAPI, storageAPI StorageAPI) *PaymentModule {
	return &PaymentModule{
		blockAPI:   blockAPI,
		storageAPI: storageAPI,
	}
}

// QueryInfo query the known data about the fee of an extrinsic at the given block
func (p *PaymentModule) QueryInfo(_ *http.Request, req *PaymentQueryInfoRequest, res *PaymentQueryInfoResponse) error {
	var hash common.Hash
	if req.Hash == nil {
		hash = p.blockAPI.BestBlockHash()
	} else {
		hash = *req.Hash
	}

	r, err := p.blockAPI.GetRuntime(&hash)
	if errors.Is(err, blocktree.ErrFailedToGetRuntime) {
		r, err = p.getRuntimeFromDB(&hash)
		if err != nil {
			return fmt.Errorf("getting runtime from database: %w", err)
		}
		defer r.Stop()
	} else if err != nil {
		return err
	}
	ext, err := common.HexToBytes(req.Ext)
	if err != nil {
		return err
	}

	encQueryInfo, err := r.PaymentQueryInfo(ext)
	if err != nil {
		return err
	}

	if encQueryInfo != nil {
		*res = PaymentQueryInfoResponse{
			Weight:     encQueryInfo.Weight,
			Class:      encQueryInfo.Class,
			PartialFee: encQueryInfo.PartialFee.String(),
		}
	}

	return nil
}

// getRuntimeFromDB gets the runtime for the corresponding block hash from storageState
func (p *PaymentModule) getRuntimeFromDB(blockHash *common.Hash) (instance runtime.Instance, err error) {
	var stateRootHash *common.Hash
	if blockHash != nil {
		stateRootHash, err = p.storageAPI.GetStateRootFromBlock(blockHash)
		if err != nil {
			return nil, fmt.Errorf("getting state root from block hash: %w", err)
		}
	}

	trieState, err := p.storageAPI.TrieState(stateRootHash)
	if err != nil {
		return nil, fmt.Errorf("getting trie state: %w", err)
	}

	code := trieState.LoadCode()
	config := wasmer.Config{
		LogLvl: log.DoNotChange,
	}
	instance, err = wasmer.NewInstance(code, config)
	if err != nil {
		return nil, fmt.Errorf("creating runtime instance: %w", err)
	}

	return instance, nil
}
