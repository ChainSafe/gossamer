// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/runtime"

	"github.com/ChainSafe/gossamer/lib/common"
	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	"github.com/stretchr/testify/require"
)

func TestStateRPCResponseValidation(t *testing.T) { //nolint:tparallel
	genesisPath := libutils.GetWestendDevRawGenesisPath(t)
	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	getBlockHashCtx, getBlockHashCancel := context.WithTimeout(ctx, time.Second)
	blockHash, err := rpc.GetBlockHash(getBlockHashCtx, node.RPCPort(), "")
	getBlockHashCancel()
	require.NoError(t, err)

	t.Run("state_call", func(t *testing.T) {
		t.Parallel()

		const params = `["Core_version", "0x"]`
		var response runtime.Version

		rpcCtx, rpcCancel := context.WithTimeout(ctx, 10*time.Second)
		defer rpcCancel()
		endpoint := rpc.NewEndpoint(node.RPCPort())
		data, err := rpc.Post(rpcCtx, endpoint, "state_call", params)
		require.NoError(t, err)

		err = rpc.DecodeScale(data, &response)
		require.NoError(t, err)

		// TODO assert stateCallResponse
	})

	t.Run("state_getKeysPaged", func(t *testing.T) {
		t.Parallel()
		t.SkipNow()

		var response struct{} // TODO
		fetchWithTimeout(ctx, t, "state_getKeysPaged", "", &response)

		// TODO assert response
	})

	t.Run("state_queryStorage", func(t *testing.T) {
		t.Parallel()
		t.SkipNow() // TODO disable skip

		params := fmt.Sprintf(
			`[["0xf2794c22e353e9a839f12faab03a911bf68967d635641a7087e53f2bff1ecad3c6756fee45ec79ead60347fffb770bcdf0ec74da701ab3d6495986fe1ecc3027"], "%s", null]`, //nolint:lll
			blockHash)
		var response modules.StorageChangeSetResponse

		fetchWithTimeout(ctx, t, "state_queryStorage", params, &response)

		// TODO assert response
	})

	t.Run("state_getRuntimeVersion", func(t *testing.T) {
		t.Parallel()

		params := fmt.Sprintf(`[%q]`, blockHash)
		var response modules.StateRuntimeVersionResponse

		fetchWithTimeout(ctx, t, "state_getRuntimeVersion", params, &response)

		// TODO assert response
	})

	t.Run("valid block hash state_getPairs", func(t *testing.T) {
		t.Parallel()

		params := fmt.Sprintf(`["0x", "%s"]`, blockHash)
		var response modules.StatePairResponse

		fetchWithTimeout(ctx, t, "state_getPairs", params, &response)

		// TODO assert response
	})

	t.Run("valid block hash state_getMetadata", func(t *testing.T) {
		t.Parallel()

		params := fmt.Sprintf(`["%s"]`, blockHash)
		var response modules.StateMetadataResponse

		fetchWithTimeout(ctx, t, "state_getMetadata", params, &response)

		// TODO assert response
	})

	t.Run("valid block hash state_getRuntimeVersion", func(t *testing.T) {
		t.Parallel()

		var response modules.StateRuntimeVersionResponse

		fetchWithTimeout(ctx, t, "state_getRuntimeVersion", "[]", &response)

		// TODO assert response
	})

	t.Run("optional params hash state_getPairs", func(t *testing.T) {
		t.Parallel()

		var response modules.StatePairResponse

		fetchWithTimeout(ctx, t, "state_getPairs", `["0x"]`, &response)

		// TODO assert response
	})

	t.Run("optional param hash state_getMetadata", func(t *testing.T) {
		t.Parallel()

		var response modules.StateMetadataResponse

		fetchWithTimeout(ctx, t, "state_getMetadata", "[]", &response)

		// TODO assert response
	})

	t.Run("optional param value as null state_getRuntimeVersion", func(t *testing.T) {
		t.Parallel()

		var response modules.StateRuntimeVersionResponse

		fetchWithTimeout(ctx, t, "state_getRuntimeVersion", "[null]", &response)

		// TODO assert response
	})

	t.Run("optional param value as null state_getMetadata", func(t *testing.T) {
		t.Parallel()

		var response modules.StateMetadataResponse

		fetchWithTimeout(ctx, t, "state_getMetadata", "[null]", &response)

		// TODO assert response
	})

	t.Run("optional param value as null state_getPairs", func(t *testing.T) {
		t.Parallel()

		var response modules.StatePairResponse

		fetchWithTimeout(ctx, t, "state_getPairs", `["0x", null]`, &response)

		// TODO assert response
	})
}

func TestStateRPCAPI(t *testing.T) {
	genesisPath := libutils.GetWestendLocalRawGenesisPath(t)
	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	time.Sleep(5 * time.Second) // Wait for block production

	getBlockHashCtx, getBlockHashCancel := context.WithTimeout(ctx, time.Second)
	blockHash, err := rpc.GetBlockHash(getBlockHashCtx, node.RPCPort(), "")
	getBlockHashCancel()
	require.NoError(t, err)

	const (
		randomHash        = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
		ErrKeyNotFound    = "Key not found"
		InvalidHashFormat = "invalid hash format"
		// `:grandpa_authorities` key
		GrandpaAuthorityKey   = "0x3a6772616e6470615f617574686f726974696573"
		GrandpaAuthorityValue = "0x010c439660b36c6c03afafca027b910b4fecf99801834c62a5e6006f27d978de2" +
			"34f010000000000000088dc3417d5058ec4b4503e0c12ea1a0a89be200fe98922423d4334014fa6b0ee0100000" +
			"000000000d17c2d7823ebf260fd138f2d7e27d114c0145d968b5ff5006125f2414fadae690100000000000000"
		StorageSizeGrandpaAuthorityKey = "122"
	)
	hash := common.MustBlake2bHash(common.MustHexToBytes(GrandpaAuthorityValue))
	storageHashGrandpaAuthorityKey := common.BytesToHex(hash[:])

	testCases := []*testCase{
		{
			description: "Test valid block hash state_getStorage",
			method:      "state_getStorage",
			params:      fmt.Sprintf(`["%s", "%s"]`, GrandpaAuthorityKey, blockHash.String()),
			expected:    GrandpaAuthorityValue,
		},
		{
			description: "Test valid block hash state_getStorageHash",
			method:      "state_getStorageHash",
			params:      fmt.Sprintf(`["%s","%s"]`, GrandpaAuthorityKey, blockHash.String()),
			expected:    storageHashGrandpaAuthorityKey,
		},
		{
			description: "Test valid block hash state_getStorageSize",
			method:      "state_getStorageSize",
			params:      fmt.Sprintf(`["%s", "%s"]`, GrandpaAuthorityKey, blockHash.String()),
			expected:    StorageSizeGrandpaAuthorityKey,
		},
		{
			description: "Test empty value state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			params:      `[""]`,
			expected:    InvalidHashFormat,
		},
		{
			description: "Test empty value hash state_getPairs",
			method:      "state_getPairs",
			params:      `["0x", ""]`,
			expected:    InvalidHashFormat,
		},
		{
			description: "Test empty value hash state_getMetadata",
			method:      "state_getMetadata",
			params:      `[""]`,
			expected:    InvalidHashFormat,
		},
		{
			description: "Test empty value hash state_getStorage",
			method:      "state_getStorage",
			params:      fmt.Sprintf(`["%s", ""]`, GrandpaAuthorityKey),
			expected:    InvalidHashFormat,
		},
		{
			description: "Test empty value hash state_getStorageHash",
			method:      "state_getStorageHash",
			params:      fmt.Sprintf(`["%s",""]`, GrandpaAuthorityKey),
			expected:    InvalidHashFormat,
		},
		{
			description: "Test empty value hash state_getStorageSize",
			method:      "state_getStorageSize",
			params:      fmt.Sprintf(`["%s", ""]`, GrandpaAuthorityKey),
			expected:    InvalidHashFormat,
		},
		{
			description: "Test optional params hash state_getStorage",
			method:      "state_getStorage",
			params:      fmt.Sprintf(`["%s"]`, GrandpaAuthorityKey),
			expected:    GrandpaAuthorityValue,
		},
		{
			description: "Test optional params hash state_getStorageHash",
			method:      "state_getStorageHash",
			params:      fmt.Sprintf(`["%s"]`, GrandpaAuthorityKey),
			expected:    storageHashGrandpaAuthorityKey,
		},
		{
			description: "Test optional params hash state_getStorageSize",
			method:      "state_getStorageSize",
			params:      fmt.Sprintf(`["%s"]`, GrandpaAuthorityKey),
			expected:    StorageSizeGrandpaAuthorityKey,
		},
		{
			description: "Test invalid block hash state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			params:      fmt.Sprintf(`["%s"]`, randomHash),
			expected:    ErrKeyNotFound,
		},
		{
			description: "Test invalid block hash state_getPairs",
			method:      "state_getPairs",
			params:      fmt.Sprintf(`["0x", "%s"]`, randomHash),
			expected:    ErrKeyNotFound,
		},
		{
			description: "Test invalid block hash state_getMetadata",
			method:      "state_getMetadata",
			params:      fmt.Sprintf(`["%s"]`, randomHash),
			expected:    ErrKeyNotFound,
		},
		{
			description: "Test invalid block hash  state_getStorage",
			method:      "state_getStorage",
			params:      fmt.Sprintf(`["%s", "%s"]`, GrandpaAuthorityKey, randomHash),
			expected:    ErrKeyNotFound,
		},
		{
			description: "Test invalid block hash state_getStorageHash",
			method:      "state_getStorageHash",
			params:      fmt.Sprintf(`["%s","%s"]`, GrandpaAuthorityKey, randomHash),
			expected:    ErrKeyNotFound,
		},
		{
			description: "Test invalid block hash state_getStorageSize",
			method:      "state_getStorageSize",
			params:      fmt.Sprintf(`["%s","%s"]`, GrandpaAuthorityKey, randomHash),
			expected:    ErrKeyNotFound,
		},
		{
			description: "Test required param missing key state_getPairs",
			method:      "state_getPairs",
			params:      `[]`,
			expected:    "Field validation for 'Prefix' failed on the 'required' tag",
		},
		{
			description: "Test required param missing key state_getStorage",
			method:      "state_getStorage",
			params:      `[]`,
			expected:    "Field validation for 'Key' failed on the 'required' tag",
		},
		{
			description: "Test required param missing key state_getStorageSize",
			method:      "state_getStorageSize",
			params:      `[]`,
			expected:    "Field validation for 'Key' failed on the 'required' tag",
		},
		{
			description: "Test required param missing key state_getStorageHash",
			method:      "state_getStorageHash",
			params:      `[]`,
			expected:    "Field validation for 'Key' failed on the 'required' tag",
		},
		{
			description: "Test required param null state_getPairs",
			method:      "state_getPairs",
			params:      `[null]`,
			expected:    "Field validation for 'Prefix' failed on the 'required' tag",
		},
		{
			description: "Test required param as null state_getStorage",
			method:      "state_getStorage",
			params:      `[null]`,
			expected:    "Field validation for 'Key' failed on the 'required' tag",
		},
		{
			description: "Test required param as null state_getStorageSize",
			method:      "state_getStorageSize",
			params:      `[null]`,
			expected:    "Field validation for 'Key' failed on the 'required' tag",
		},
		{
			description: "Test required param as null state_getStorageHash",
			method:      "state_getStorageHash",
			params:      `[null]`,
			expected:    "Field validation for 'Key' failed on the 'required' tag",
		},
	}

	// Cases for valid block hash in RPC params
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			postRPCCtx, cancel := context.WithTimeout(ctx, time.Second)
			endpoint := rpc.NewEndpoint(node.RPCPort())
			respBody, err := rpc.Post(postRPCCtx, endpoint, test.method, test.params)
			cancel()
			require.NoError(t, err)

			require.Contains(t, string(respBody), test.expected)
		})
	}
}

func TestRPCStructParamUnmarshal(t *testing.T) {
	genesisPath := libutils.GetWestendDevRawGenesisPath(t)
	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	time.Sleep(2 * time.Second) // Wait for block production

	test := testCase{
		description: "Test valid read request in local json2",
		method:      "state_queryStorage",
		params:      `[["0xf2794c22e353e9a839f12faab03a911bf68967d635641a7087e53f2bff1ecad3c6756fee45ec79ead60347fffb770bcdf0ec74da701ab3d6495986fe1ecc3027"],"0xa32c60dee8647b07435ae7583eb35cee606209a595718562dd4a486a07b6de15", null]`, //nolint:lll
	}
	t.Run(test.description, func(t *testing.T) {
		postRPCCtx, postRPCCancel := context.WithTimeout(ctx, time.Second)
		endpoint := rpc.NewEndpoint(node.RPCPort())
		respBody, err := rpc.Post(postRPCCtx, endpoint, test.method, test.params)
		postRPCCancel()
		require.NoError(t, err)
		require.NotContains(t, string(respBody), "json: cannot unmarshal")
		fmt.Println(string(respBody))
	})
}
