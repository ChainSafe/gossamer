// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
)

func TestStateRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip RPC suite tests")
		return
	}

	testCases := []*testCase{
		{ //TODO
			description: "test state_call",
			method:      "state_call",
			params:      `["", "","0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"]`,
			expected:    modules.StateCallResponse{},
		},
		{ //TODO disable skip when implemented
			description: "test state_getKeysPaged",
			method:      "state_getKeysPaged",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildKeys",
			method:      "state_getChildKeys",
			params:      `["","","0x579deccea7183c2afedbdaea59ad23e970458186afc4d57d5577842d4a219925"]`,
			expected:    modules.StateKeysResponse{},
		},
		{ //TODO
			description: "test state_getChildStorage",
			method:      "state_getChildStorage",
			params:      `["","","0x579deccea7183c2afedbdaea59ad23e970458186afc4d57d5577842d4a219925"]`,
			expected:    modules.StateStorageDataResponse{},
		},
		{ //TODO
			description: "test state_getChildStorageHash",
			method:      "state_getChildStorageHash",
			params:      `["","","0x579deccea7183c2afedbdaea59ad23e970458186afc4d57d5577842d4a219925"]`,
			expected:    modules.StateChildStorageResponse{},
		},
		{ //TODO
			description: "test state_getChildStorageSize",
			method:      "state_getChildStorageSize",
			params:      `["","","0x579deccea7183c2afedbdaea59ad23e970458186afc4d57d5577842d4a219925"]`,
			expected:    modules.StateChildStorageSizeResponse{},
		},
		{ //TODO
			description: "test state_queryStorage",
			method:      "state_queryStorage",
			params:      `[["0xf2794c22e353e9a839f12faab03a911bf68967d635641a7087e53f2bff1ecad3c6756fee45ec79ead60347fffb770bcdf0ec74da701ab3d6495986fe1ecc3027"], "0xa32c60dee8647b07435ae7583eb35cee606209a595718562dd4a486a07b6de15", null]`,
			expected:    modules.StorageChangeSetResponse{},
		},
	}

	t.Log("starting gossamer...")
	nodes, err := utils.InitializeAndStartNodes(t, 1, utils.GenesisDefault, utils.ConfigDefault)
	require.Nil(t, err)

	time.Sleep(time.Second) // give server a second to start

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			_ = getResponse(t, test)
		})
	}

	t.Log("going to tear down gossamer...")
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}

func TestStateRPCAPI(t *testing.T) {
	if utils.MODE != rpcSuite {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip RPC suite tests")
		return
	}

	t.Log("starting gossamer...")

	utils.CreateConfigBabeMaxThreshold()
	defer os.Remove(utils.ConfigBABEMaxThreshold)

	nodes, err := utils.InitializeAndStartNodes(t, 1, utils.GenesisDefault, utils.ConfigBABEMaxThreshold)
	require.Nil(t, err)

	defer func() {
		t.Log("going to tear down gossamer...")
		errList := utils.TearDown(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(5 * time.Second) // Wait for block production

	blockHash, err := utils.GetBlockHash(t, nodes[0], "")

	if err != nil {
		blockHash = common.Hash{}
	}

	const (
		randomHash                     = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
		ErrKeyNotFound                 = "Key not found"
		InvalidHashFormat              = "invalid hash format"
		GrandpaAuthorityKey            = "0x3a6772616e6470615f617574686f726974696573" // `:grandpa_authorities` key
		GrandpaAuthorityValue          = "0x012434602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a691010000000000000094a297125bf31bc15e2a2f1d7d44d2c2a99ce3ed81fdc3a7acf4a4cc30480fb7010000000000000041a8d68c449e3afc7e4676827a4b11a0c9ec238542327f2e46a8b70a32501bca01000000000000007d1bfc260fee0dcdd73457c15a3895747d1c2fdc4c097060e34f54c99ea1c6c10100000000000000b1f9449c9dea2baa872a96bf655a6e266888ec4ea55051508d8bb725e936cf0c01000000000000002e4cc1538f2fd132e0396282ad5c1d7a54eba14d9ad0eee3768c3ae656577a6001000000000000009dff473ab4f7b55caa005060053a7b315cedb928cf9f99792753ad3a3b6ae8a401000000000000004e30525ea941dc9ccd21302b195a6312024ad627cbe4814c89473ce03c3a20e301000000000000009a2d335e656481978c39fb571ed37c3e04ac2c4c44d450aefb2e71205e2e1d230100000000000000"
		StorageHashGrandpaAuthorityKey = "0x8c39fb571ed37c3e04ac2c4c44d450aefb2e71205e2e1d230100000000000000"
		StorageSizeGrandpaAuthorityKey = "362"
	)

	testCases := []*testCase{
		{
			description: "Test valid block hash state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			params:      fmt.Sprintf(`["%s"]`, blockHash.String()),
		},
		{
			description: "Test valid block hash state_getPairs",
			method:      "state_getPairs",
			params:      fmt.Sprintf(`["0x", "%s"]`, blockHash.String()),
		},
		{
			description: "Test valid block hash state_getMetadata",
			method:      "state_getMetadata",
			params:      fmt.Sprintf(`["%s"]`, blockHash.String()),
		},
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
			expected:    StorageHashGrandpaAuthorityKey,
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
			isErr:       true,
		},
		{
			description: "Test empty value hash state_getPairs",
			method:      "state_getPairs",
			params:      `["0x", ""]`,
			expected:    InvalidHashFormat,
			isErr:       true,
		},
		{
			description: "Test empty value hash state_getMetadata",
			method:      "state_getMetadata",
			params:      `[""]`,
			expected:    InvalidHashFormat,
			isErr:       true,
		},
		{
			description: "Test empty value hash state_getStorage",
			method:      "state_getStorage",
			params:      fmt.Sprintf(`["%s", ""]`, GrandpaAuthorityKey),
			expected:    InvalidHashFormat,
			isErr:       true,
		},
		{
			description: "Test empty value hash state_getStorageHash",
			method:      "state_getStorageHash",
			params:      fmt.Sprintf(`["%s",""]`, GrandpaAuthorityKey),
			expected:    InvalidHashFormat,
			isErr:       true,
		},
		{
			description: "Test empty value hash state_getStorageSize",
			method:      "state_getStorageSize",
			params:      fmt.Sprintf(`["%s", ""]`, GrandpaAuthorityKey),
			expected:    InvalidHashFormat,
			isErr:       true,
		},
		{
			description: "Test optional params state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			params:      `[]`,
		},
		{
			description: "Test optional params hash state_getPairs",
			method:      "state_getPairs",
			params:      `["0x"]`,
		},
		{
			description: "Test optional params hash state_getMetadata",
			method:      "state_getMetadata",
			params:      `[]`,
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
			expected:    StorageHashGrandpaAuthorityKey,
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
			isErr:       true,
		},
		{
			description: "Test invalid block hash state_getPairs",
			method:      "state_getPairs",
			params:      fmt.Sprintf(`["0x", "%s"]`, randomHash),
			expected:    ErrKeyNotFound,
			isErr:       true,
		},
		{
			description: "Test invalid block hash state_getMetadata",
			method:      "state_getMetadata",
			params:      fmt.Sprintf(`["%s"]`, randomHash),
			expected:    ErrKeyNotFound,
			isErr:       true,
		},
		{
			description: "Test invalid block hash  state_getStorage",
			method:      "state_getStorage",
			params:      fmt.Sprintf(`["%s", "%s"]`, GrandpaAuthorityKey, randomHash),
			expected:    ErrKeyNotFound,
			isErr:       true,
		},
		{
			description: "Test invalid block hash state_getStorageHash",
			method:      "state_getStorageHash",
			params:      fmt.Sprintf(`["%s","%s"]`, GrandpaAuthorityKey, randomHash),
			expected:    ErrKeyNotFound,
			isErr:       true,
		},
		{
			description: "Test invalid block hash state_getStorageSize",
			method:      "state_getStorageSize",
			params:      fmt.Sprintf(`["%s","%s"]`, GrandpaAuthorityKey, randomHash),
			expected:    ErrKeyNotFound,
			isErr:       true,
		},
		// TODO: Add validation for this RPC and enable the test.
		//{
		//	description: "Test required params missing hash state_getPairs",
		//	method:      "state_getPairs",
		//	params:      `[]`,
		//	expected:    "required field missing in params",
		//	isErr:       true,
		//},
	}

	// Cases for valid block hash in RPC params
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			respBody, err := utils.PostRPC(test.method, utils.NewEndpoint(nodes[0].RPCPort), test.params)
			require.Nil(t, err)

			if test.isErr || test.expected != nil {
				require.Contains(t, string(respBody), test.expected)
			} else {
				require.NotContains(t, string(respBody), test.expected)
			}
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestRPCStructParamUnmarshal(t *testing.T) {
	if utils.MODE != rpcSuite {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip RPC suite tests")
		return
	}

	t.Log("starting gossamer...")

	utils.CreateConfigBabeMaxThreshold()
	defer os.Remove(utils.ConfigBABEMaxThreshold)

	nodes, err := utils.InitializeAndStartNodes(t, 1, utils.GenesisDefault, utils.ConfigBABEMaxThreshold)
	require.Nil(t, err)

	defer func() {
		t.Log("going to tear down gossamer...")
		errList := utils.TearDown(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(2 * time.Second) // Wait for block production

	test := testCase{
		description: "Test valid read request in local json2",
		method:      "state_queryStorage",
		params:      `[["0xf2794c22e353e9a839f12faab03a911bf68967d635641a7087e53f2bff1ecad3c6756fee45ec79ead60347fffb770bcdf0ec74da701ab3d6495986fe1ecc3027"],"0xa32c60dee8647b07435ae7583eb35cee606209a595718562dd4a486a07b6de15", null]`,
	}
	t.Run(test.description, func(t *testing.T) {
		respBody, err := utils.PostRPC(test.method, utils.NewEndpoint(nodes[0].RPCPort), test.params)
		require.Nil(t, err)
		require.NotContains(t, string(respBody), "json: cannot unmarshal")
		fmt.Println(string(respBody))
	})
}
