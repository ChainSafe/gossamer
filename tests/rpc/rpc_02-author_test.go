// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v3/scale"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/retry"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/stretchr/testify/require"
)

func TestAuthorSubmitExtrinsic(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	tomlConfig.Core.BABELead = true

	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("http://localhost:%s", node.GetRPCPort()))
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

func TestAuthorRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	testCases := []*testCase{
		{ //TODO
			description: "test author_submitExtrinsic",
			method:      "author_submitExtrinsic",
			skip:        true,
		},
		{ //TODO
			description: "test author_pendingExtrinsics",
			method:      "author_pendingExtrinsics",
			skip:        true,
		},
		{ //TODO
			description: "test author_removeExtrinsic",
			method:      "author_removeExtrinsic",
			skip:        true,
		},
		{ //TODO
			description: "test author_insertKey",
			method:      "author_insertKey",
			skip:        true,
		},
		{ //TODO
			description: "test author_rotateKeys",
			method:      "author_rotateKeys",
			skip:        true,
		},
		{ //TODO
			description: "test author_hasSessionKeys",
			method:      "author_hasSessionKeys",
			skip:        true,
		},
		{ //TODO
			description: "test author_hasKey",
			method:      "author_hasKey",
			skip:        true,
		},
	}

	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	tomlConfig.Core.BABELead = true
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			if test.skip {
				t.SkipNow()
			}

			getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
			defer getResponseCancel()
			target := reflect.New(reflect.TypeOf(test.expected)).Interface()
			err := getResponse(getResponseCtx, test.method, test.params, target)
			require.NoError(t, err)
		})
	}
}
