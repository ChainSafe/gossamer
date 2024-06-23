// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/retry"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/stretchr/testify/require"
)

// TODO: add test against latest dev runtime
// See https://github.com/ChainSafe/gossamer/issues/2705
func TestAuthorSubmitExtrinsic(t *testing.T) {
	startTime := time.Now()
	t.Cleanup(func() {
		elapsedTime := time.Since(startTime)
		t.Logf("TestAuthorSubmitExtrinsic total test time: %v --------------------", elapsedTime)
	})
	genesisPath := libutils.GetWestendDevRawGenesisPath(t)
	tomlConfig := config.Default()
	tomlConfig.Account.Key = config.AliceKey
	tomlConfig.ChainSpec = genesisPath

	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("http://localhost:%s", node.RPCPort()))
	require.NoError(t, err)

	// Wait for the first block to be produced.
	const retryWait = time.Second
	err = retry.UntilOK(ctx, retryWait, func() (ok bool, err error) {
		block, err := api.RPC.Chain.GetBlockLatest()
		if err != nil {
			return false, err
		}
		return block.Block.Header.Number > 0, nil
	})
	require.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	require.NoError(t, err)

	c, err := types.NewCall(meta, "System.remark", []byte{0xab})
	require.NoError(t, err)

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	require.NoError(t, err)

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	require.NoError(t, err)

	key, err := types.CreateStorageKey(meta, "System", "Account", signature.TestKeyringPairAlice.PublicKey, nil)
	require.NoError(t, err)

	var accInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accInfo)
	require.NoError(t, err)
	require.True(t, ok)

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsImmortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	buffer := bytes.Buffer{}
	encoder := scale.NewEncoder(&buffer)
	ext.Encode(*encoder)

	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	require.NoError(t, err)
	require.NotEqual(t, types.Hash{}, hash)
}

func TestAuthorRPC(t *testing.T) { //nolint:tparallel
	startTime := time.Now()
	t.Cleanup(func() {
		elapsedTime := time.Since(startTime)
		t.Logf("TestAuthorRPC total test time: %v ----------------", elapsedTime)
	})
	genesisPath := libutils.GetWestendDevRawGenesisPath(t)
	tomlConfig := config.Default()
	tomlConfig.ChainSpec = genesisPath
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	t.Run("author_pendingExtrinsics", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO

		var target interface{} // TODO
		fetchWithTimeout(ctx, t, "author_pendingExtrinsics", "", target)
	})

	t.Run("author_submitExtrinsic", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO

		var target interface{} // TODO
		fetchWithTimeout(ctx, t, "author_submitExtrinsic", "", target)
	})

	t.Run("author_pendingExtrinsics", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO

		var target interface{} // TODO
		fetchWithTimeout(ctx, t, "author_pendingExtrinsics", "", target)
	})

	t.Run("author_removeExtrinsic", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO

		var target interface{} // TODO
		fetchWithTimeout(ctx, t, "author_removeExtrinsic", "", target)
	})

	t.Run("author_insertKey", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO

		var target interface{} // TODO
		fetchWithTimeout(ctx, t, "author_insertKey", "", target)
	})

	t.Run("author_rotateKeys", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO

		var target interface{} // TODO
		fetchWithTimeout(ctx, t, "author_rotateKeys", "", target)
	})

	t.Run("author_hasSessionKeys", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO

		var target interface{} // TODO
		fetchWithTimeout(ctx, t, "author_hasSessionKeys", "", target)
	})

	t.Run("author_hasKey", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO

		var target interface{} // TODO
		fetchWithTimeout(ctx, t, "author_hasKey", "", target)
	})
}
