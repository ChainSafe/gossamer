// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	"github.com/stretchr/testify/require"
)

func TestStateRPCResponseValidation(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	config := config.CreateDefault(t)
	node := node.New(t, node.SetBabeLead(true),
		node.SetGenesis(genesisPath), node.SetConfig(config))
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	time.Sleep(time.Second) // give server a second to start

	getBlockHashCtx, getBlockHashCancel := context.WithTimeout(ctx, time.Second)
	blockHash, err := rpc.GetBlockHash(getBlockHashCtx, node.GetRPCPort(), "")
	getBlockHashCancel()
	require.NoError(t, err)

	testCases := []*testCase{
		{
			description: "Test state_call",
			method:      "state_call",
			params:      `["", "","0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"]`,
			expected:    modules.StateCallResponse{},
		},
		{ //TODO disable skip when implemented
			description: "Test state_getKeysPaged",
			method:      "state_getKeysPaged",
			skip:        true,
		},
		{
			description: "Test state_queryStorage",
			method:      "state_queryStorage",
			params: fmt.Sprintf(
				`[["0xf2794c22e353e9a839f12faab03a911bf68967d635641a7087e53f2bff1ecad3c6756fee45ec79ead60347fffb770bcdf0ec74da701ab3d6495986fe1ecc3027"], "%s", null]`, //nolint:lll
				blockHash),
			expected: modules.StorageChangeSetResponse{
				Block:   &blockHash,
				Changes: [][]string{},
			},
			skip: true,
		},
		{
			description: "Test valid block hash state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			params:      fmt.Sprintf(`["%s"]`, blockHash.String()),
			expected:    modules.StateRuntimeVersionResponse{},
		},
		{
			description: "Test valid block hash state_getPairs",
			method:      "state_getPairs",
			params:      fmt.Sprintf(`["0x", "%s"]`, blockHash.String()),
			expected:    modules.StatePairResponse{},
		},
		{
			description: "Test valid block hash state_getMetadata",
			method:      "state_getMetadata",
			params:      fmt.Sprintf(`["%s"]`, blockHash.String()),
			expected:    modules.StateMetadataResponse(""),
		},
		{
			description: "Test optional param state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			params:      `[]`,
			expected:    modules.StateRuntimeVersionResponse{},
		},
		{
			description: "Test optional params hash state_getPairs",
			method:      "state_getPairs",
			params:      `["0x"]`,
			expected:    modules.StatePairResponse{},
		},
		{
			description: "Test optional param hash state_getMetadata",
			method:      "state_getMetadata",
			params:      `[]`,
			expected:    modules.StateMetadataResponse(""),
		},
		{
			description: "Test optional param value as null state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			params:      `[null]`,
			expected:    modules.StateRuntimeVersionResponse{},
		},
		{
			description: "Test optional param value as null state_getMetadata",
			method:      "state_getMetadata",
			params:      `[null]`,
			expected:    modules.StateMetadataResponse(""),
		},
		{
			description: "Test optional param value as null state_getPairs",
			method:      "state_getPairs",
			params:      `["0x", null]`,
			expected:    modules.StatePairResponse{},
		},
	}

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

func TestStateRPCAPI(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	config := config.CreateDefault(t)
	node := node.New(t, node.SetBabeLead(true),
		node.SetGenesis(genesisPath), node.SetConfig(config))
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	time.Sleep(5 * time.Second) // Wait for block production

	getBlockHashCtx, getBlockHashCancel := context.WithTimeout(ctx, time.Second)
	blockHash, err := rpc.GetBlockHash(getBlockHashCtx, node.GetRPCPort(), "")
	getBlockHashCancel()
	require.NoError(t, err)

	const (
		randomHash        = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
		ErrKeyNotFound    = "Key not found"
		InvalidHashFormat = "invalid hash format"
		// `:grandpa_authorities` key
		GrandpaAuthorityKey            = "0x3a6772616e6470615f617574686f726974696573"
		GrandpaAuthorityValue          = "0x012488dc3417d5058ec4b4503e0c12ea1a0a89be200fe98922423d4334014fa6b0ee0100000000000000d17c2d7823ebf260fd138f2d7e27d114c0145d968b5ff5006125f2414fadae690100000000000000439660b36c6c03afafca027b910b4fecf99801834c62a5e6006f27d978de234f01000000000000005e639b43e0052c47447dac87d6fd2b6ec50bdd4d0f614e4299c665249bbd09d901000000000000001dfe3e22cc0d45c70779c1095f7489a8ef3cf52d62fbd8c2fa38c9f1723502b50100000000000000568cb4a574c6d178feb39c27dfc8b3f789e5f5423e19c71633c748b9acf086b5010000000000000008ee9f4a5246647ebb938ece750d3d3be5e5f31978460258a1ab850c5d2b698201000000000000005c2c289b817ff4f843447a3346c0f63876acca1b0b93ff65736b4d4f26b8323101000000000000001da77f955bcd0745d2bc7a7e6544a661f4536deabf57fe79737b3e9157e39e420100000000000000" //nolint:lll
		StorageSizeGrandpaAuthorityKey = "362"
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
			endpoint := rpc.NewEndpoint(node.GetRPCPort())
			respBody, err := rpc.Post(postRPCCtx, endpoint, test.method, test.params)
			cancel()
			require.NoError(t, err)

			require.Contains(t, string(respBody), test.expected)
		})
	}
}

func TestRPCStructParamUnmarshal(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	config := config.CreateDefault(t)
	node := node.New(t, node.SetBabeLead(true),
		node.SetGenesis(genesisPath), node.SetConfig(config))
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
		endpoint := rpc.NewEndpoint(node.GetRPCPort())
		respBody, err := rpc.Post(postRPCCtx, endpoint, test.method, test.params)
		postRPCCancel()
		require.NoError(t, err)
		require.NotContains(t, string(respBody), "json: cannot unmarshal")
		fmt.Println(string(respBody))
	})
}
