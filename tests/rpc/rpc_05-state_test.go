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
			skip:        true,
		},
		{ //TODO
			description: "test state_getPairs",
			method:      "state_getPairs",
			skip:        true,
		},
		{ //TODO
			description: "test state_getKeysPaged",
			method:      "state_getKeysPaged",
			skip:        true,
		},
		{ //TODO
			description: "test state_getStorage",
			method:      "state_getStorage",
			skip:        true,
		},
		{ //TODO
			description: "test state_getStorageHash",
			method:      "state_getStorageHash",
			skip:        true,
		},
		{ //TODO
			description: "test state_getStorageSize",
			method:      "state_getStorageSize",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildKeys",
			method:      "state_getChildKeys",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildStorage",
			method:      "state_getChildStorage",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildStorageHash",
			method:      "state_getChildStorageHash",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildStorageSize",
			method:      "state_getChildStorageSize",
			skip:        true,
		},
		{ //TODO
			description: "test state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			skip:        true,
		},
		{ //TODO
			description: "test state_queryStorage",
			method:      "state_queryStorage",
			skip:        true,
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

	const randomHash = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
	const ErrKeyNotFound = "Key not found"
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
			description: "Test empty value state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			params:      `[""]`,
		},
		{
			description: "Test empty value hash state_getPairs",
			method:      "state_getPairs",
			params:      `["0x", ""]`,
		},
		{
			description: "Test empty value hash state_getMetadata",
			method:      "state_getMetadata",
			params:      `[""]`,
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
			description: "Test required params missing hash state_getPairs",
			method:      "state_getPairs",
			params:      `[]`,
			expected:    "required field missing in params",
			isErr:       true,
		},
	}

	// Cases for valid block hash in RPC params
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			respBody, err := utils.PostRPC(test.method, utils.NewEndpoint(nodes[0].RPCPort), test.params)
			require.Nil(t, err)

			if test.isErr {
				require.Contains(t, string(respBody), test.expected)
			} else {
				require.NotContains(t, string(respBody), test.expected)
			}
			time.Sleep(100 * time.Millisecond)
		})
	}
}
