// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"reflect"
	"testing"
	"time"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/stretchr/testify/require"
)

func TestEngineRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	testCases := []*testCase{
		{ //TODO
			description: "test engine_createBlock",
			method:      "engine_createBlock",
			skip:        true,
		},
		{ //TODO
			description: "test engine_finalizeBlock",
			method:      "engine_finalizeBlock",
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

	time.Sleep(time.Second) // give server a second to start

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
