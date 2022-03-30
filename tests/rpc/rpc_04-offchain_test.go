// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"testing"
	"time"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
)

func TestOffchainRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	testCases := []*testCase{
		{ //TODO
			description: "test offchain_localStorageSet",
			method:      "offchain_localStorageSet",
			skip:        true,
		},
		{ //TODO
			description: "test offchain_localStorageGet",
			method:      "offchain_localStorageGet",
			skip:        true,
		},
		{ //TODO
			description: "test offchain_localStorageGet",
			method:      "offchain_localStorageGet",
			skip:        true,
		},
	}

	genesisPath := libutils.GetGssmrGenesisRawPathTest(t)
	config := config.CreateDefault(t)
	node := node.New(t, node.SetBabeLead(true),
		node.SetGenesis(genesisPath), node.SetConfig(config))
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	time.Sleep(time.Second) // give server a second to start

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
			defer getResponseCancel()
			_ = getResponse(getResponseCtx, t, test)
		})
	}
}
