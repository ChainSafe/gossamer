// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// GetChainHead calls the endpoint chain_getHeader to get the latest chain head
func GetChainHead(t *testing.T, node *Node) *types.Header {
	respBody, err := PostRPC(ChainGetHeader, NewEndpoint(node.RPCPort), "[]")
	require.NoError(t, err)

	header := new(modules.ChainBlockHeaderResponse)
	err = DecodeRPC(t, respBody, header)
	require.NoError(t, err)

	return headerResponseToHeader(t, header)
}

// GetChainHeadWithError calls the endpoint chain_getHeader to get the latest chain head
func GetChainHeadWithError(t *testing.T, node *Node) (*types.Header, error) {
	respBody, err := PostRPC(ChainGetHeader, NewEndpoint(node.RPCPort), "[]")
	require.NoError(t, err)

	header := new(modules.ChainBlockHeaderResponse)
	err = DecodeRPC(t, respBody, header)
	if err != nil {
		return nil, err
	}

	return headerResponseToHeader(t, header), nil
}

// GetBlockHash calls the endpoint chain_getBlockHash to get the latest chain head
func GetBlockHash(t *testing.T, node *Node, num string) (common.Hash, error) {
	respBody, err := PostRPCWithRetry(ChainGetBlockHash, NewEndpoint(node.RPCPort), "["+num+"]", 5)
	if err != nil {
		return common.Hash{}, err
	}

	var hash string
	err = DecodeRPC(t, respBody, &hash)
	if err != nil {
		return common.Hash{}, err
	}
	return common.MustHexToHash(hash), nil
}

// GetFinalizedHead calls the endpoint chain_getFinalizedHead to get the latest finalised head
func GetFinalizedHead(t *testing.T, node *Node) common.Hash {
	respBody, err := PostRPC(ChainGetFinalizedHead, NewEndpoint(node.RPCPort), "[]")
	require.NoError(t, err)

	var hash string
	err = DecodeRPC(t, respBody, &hash)
	require.NoError(t, err)
	return common.MustHexToHash(hash)
}

// GetFinalizedHeadByRound calls the endpoint chain_getFinalizedHeadByRound to get the finalised head at a given round
// TODO: add setID, hard-coded at 1 for now
func GetFinalizedHeadByRound(t *testing.T, node *Node, round uint64) (common.Hash, error) {
	p := strconv.Itoa(int(round))
	respBody, err := PostRPC(ChainGetFinalizedHeadByRound, NewEndpoint(node.RPCPort), "["+p+",1]")
	require.NoError(t, err)

	var hash string
	err = DecodeRPC(t, respBody, &hash)
	if err != nil {
		return common.Hash{}, err
	}

	return common.MustHexToHash(hash), nil
}

// GetBlock calls the endpoint chain_getBlock
func GetBlock(t *testing.T, node *Node, hash common.Hash) *types.Block {
	respBody, err := PostRPC(ChainGetBlock, NewEndpoint(node.RPCPort), "[\""+hash.String()+"\"]")
	require.NoError(t, err)

	block := new(modules.ChainBlockResponse)
	err = DecodeRPC(t, respBody, block)
	if err != nil {
		return nil
	}

	header := block.Block.Header

	parentHash, err := common.HexToHash(header.ParentHash)
	require.NoError(t, err)

	nb, err := common.HexToBytes(header.Number)
	require.NoError(t, err)
	number := big.NewInt(0).SetBytes(nb)

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
