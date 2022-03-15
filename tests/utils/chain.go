// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// GetChainHead calls the endpoint chain_getHeader to get the latest chain head
func GetChainHead(ctx context.Context, t *testing.T, rpcPort string) *types.Header {
	endpoint := NewEndpoint(rpcPort)
	const params = "[]"
	respBody, err := PostRPC(ctx, endpoint, ChainGetHeader, params)
	require.NoError(t, err)

	header := new(modules.ChainBlockHeaderResponse)
	err = DecodeRPC(respBody, header)
	require.NoError(t, err)

	return headerResponseToHeader(t, header)
}

// GetChainHeadWithError calls the endpoint chain_getHeader to get the latest chain head
func GetChainHeadWithError(ctx context.Context, t *testing.T, rpcPort string) (*types.Header, error) {
	endpoint := NewEndpoint(rpcPort)
	const params = "[]"
	respBody, err := PostRPC(ctx, endpoint, ChainGetHeader, params)
	require.NoError(t, err)

	header := new(modules.ChainBlockHeaderResponse)
	err = DecodeRPC(respBody, header)
	if err != nil {
		return nil, fmt.Errorf("cannot decode RPC response: %w", err)
	}

	return headerResponseToHeader(t, header), nil
}

// GetBlockHash calls the endpoint chain_getBlockHash to get the latest chain head.
// It will block until a response is received or the context gets canceled.
func GetBlockHash(ctx context.Context, t *testing.T, rpcPort, num string) (common.Hash, error) {
	endpoint := NewEndpoint(rpcPort)
	params := "[" + num + "]"
	const requestWait = time.Second
	respBody, err := PostRPCWithRetry(ctx, endpoint, ChainGetBlockHash, params, requestWait)
	require.NoError(t, err)

	var hash string
	err = DecodeRPC(respBody, &hash)
	if err != nil {
		return common.Hash{}, err
	}
	return common.MustHexToHash(hash), nil
}

// GetFinalizedHead calls the endpoint chain_getFinalizedHead to get the latest finalised head
func GetFinalizedHead(ctx context.Context, t *testing.T, rpcPort string) common.Hash {
	endpoint := NewEndpoint(rpcPort)
	method := ChainGetFinalizedHead
	const params = "[]"
	respBody, err := PostRPC(ctx, endpoint, method, params)
	require.NoError(t, err)

	var hash string
	err = DecodeRPC(respBody, &hash)
	require.NoError(t, err)
	return common.MustHexToHash(hash)
}

// GetFinalizedHeadByRound calls the endpoint chain_getFinalizedHeadByRound to get the finalised head at a given round
// TODO: add setID, hard-coded at 1 for now
func GetFinalizedHeadByRound(ctx context.Context, t *testing.T, rpcPort string, round uint64) (common.Hash, error) {
	p := strconv.Itoa(int(round))
	endpoint := NewEndpoint(rpcPort)
	method := ChainGetFinalizedHeadByRound
	params := "[" + p + ",1]"
	respBody, err := PostRPC(ctx, endpoint, method, params)
	require.NoError(t, err)

	var hash string
	err = DecodeRPC(respBody, &hash)
	if err != nil {
		return common.Hash{}, err
	}

	return common.MustHexToHash(hash), nil
}

// GetBlock calls the endpoint chain_getBlock
func GetBlock(ctx context.Context, t *testing.T, rpcPort string, hash common.Hash) *types.Block {
	endpoint := NewEndpoint(rpcPort)
	method := ChainGetBlock
	params := fmt.Sprintf(`["%s"]`, hash)
	respBody, err := PostRPC(ctx, endpoint, method, params)
	require.NoError(t, err)

	block := new(modules.ChainBlockResponse)
	err = DecodeRPC(respBody, block)
	if err != nil {
		return nil
	}

	header := block.Block.Header

	parentHash, err := common.HexToHash(header.ParentHash)
	require.NoError(t, err)

	nb, err := common.HexToBytes(header.Number)
	require.NoError(t, err)
	number := common.BytesToUint(nb)

	stateRoot, err := common.HexToHash(header.StateRoot)
	require.NoError(t, err)

	extrinsicsRoot, err := common.HexToHash(header.ExtrinsicsRoot)
	require.NoError(t, err)

	h, err := types.NewHeader(parentHash, stateRoot, extrinsicsRoot, number, rpcLogsToDigest(t, header.Digest.Logs))
	require.NoError(t, err)

	b, err := types.NewBodyFromExtrinsicStrings(block.Block.Body)
	require.NoError(t, err, fmt.Sprintf("%v", block.Block.Body))

	return &types.Block{
		Header: *h,
		Body:   *b,
	}
}
