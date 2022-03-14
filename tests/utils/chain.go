// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// GetChainHead calls the endpoint chain_getHeader to get the latest chain head
func GetChainHead(ctx context.Context, rpcPort string) (header *types.Header, err error) {
	endpoint := NewEndpoint(rpcPort)
	const params = "[]"
	respBody, err := PostRPC(ctx, endpoint, ChainGetHeader, params)
	if err != nil {
		return nil, fmt.Errorf("cannot post RPC: %w", err)
	}

	var rpcHeader modules.ChainBlockHeaderResponse
	err = DecodeRPC(respBody, &rpcHeader)
	if err != nil {
		return nil, fmt.Errorf("cannot decode RPC response: %w", err)
	}

	header, err = headerResponseToHeader(rpcHeader)
	if err != nil {
		return nil, fmt.Errorf("malformed RPC header: %w", err)
	}

	return header, nil
}

// GetBlockHash calls the endpoint chain_getBlockHash to get the latest chain head.
// It will block until a response is received or the context gets canceled.
func GetBlockHash(ctx context.Context, rpcPort, num string) (hash common.Hash, err error) {
	endpoint := NewEndpoint(rpcPort)
	params := "[" + num + "]"
	respBody, err := PostRPC(ctx, endpoint, ChainGetBlockHash, params)
	if err != nil {
		return hash, fmt.Errorf("cannot post RPC: %w", err)
	}

	return hexStringBodyToHash(respBody)
}

// GetFinalizedHead calls the endpoint chain_getFinalizedHead to get the latest finalised head
func GetFinalizedHead(ctx context.Context, rpcPort string) (
	hash common.Hash, err error) {
	endpoint := NewEndpoint(rpcPort)
	method := ChainGetFinalizedHead
	const params = "[]"
	respBody, err := PostRPC(ctx, endpoint, method, params)
	if err != nil {
		return hash, fmt.Errorf("cannot post RPC: %w", err)
	}

	return hexStringBodyToHash(respBody)
}

// GetFinalizedHeadByRound calls the endpoint chain_getFinalizedHeadByRound to get the finalised head at a given round
// TODO: add setID, hard-coded at 1 for now
func GetFinalizedHeadByRound(ctx context.Context, rpcPort string, round uint64) (
	hash common.Hash, err error) {
	p := strconv.Itoa(int(round))
	endpoint := NewEndpoint(rpcPort)
	method := ChainGetFinalizedHeadByRound
	params := "[" + p + ",1]"
	respBody, err := PostRPC(ctx, endpoint, method, params)
	if err != nil {
		return hash, fmt.Errorf("cannot post RPC: %w", err)
	}

	return hexStringBodyToHash(respBody)
}

// GetBlock calls the endpoint chain_getBlock
func GetBlock(ctx context.Context, rpcPort string, hash common.Hash) (
	block *types.Block, err error) {
	endpoint := NewEndpoint(rpcPort)
	method := ChainGetBlock
	params := fmt.Sprintf(`["%s"]`, hash)
	respBody, err := PostRPC(ctx, endpoint, method, params)
	if err != nil {
		return nil, fmt.Errorf("cannot post RPC: %w", err)
	}

	rpcBlock := new(modules.ChainBlockResponse)
	err = DecodeRPC(respBody, rpcBlock)
	if err != nil {
		return nil, fmt.Errorf("cannot decode RPC response body: %w", err)
	}

	rpcHeader := rpcBlock.Block.Header
	header, err := headerResponseToHeader(rpcHeader)
	if err != nil {
		return nil, fmt.Errorf("malformed RPC header: %w", err)
	}

	body, err := types.NewBodyFromExtrinsicStrings(rpcBlock.Block.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot create body from RPC block blody: %w", err)
	}

	return &types.Block{
		Header: *header,
		Body:   *body,
	}, nil
}

func hexStringBodyToHash(body []byte) (hash common.Hash, err error) {
	var hexHashString string
	err = DecodeRPC(body, &hexHashString)
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot decode RPC: %w", err)
	}

	hash, err = common.HexToHash(hexHashString)
	if err != nil {
		return common.Hash{}, fmt.Errorf("malformed block hash hex string: %w", err)
	}

	return hash, nil
}
