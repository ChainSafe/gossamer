// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
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
	blockAPI BlockAPI
}

// NewPaymentModule returns a pointer to PaymentModule
func NewPaymentModule(blockAPI BlockAPI) *PaymentModule {
	return &PaymentModule{
		blockAPI: blockAPI,
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
	if err != nil {
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
