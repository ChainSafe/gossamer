// Copyright 2021 ChainSafe Systems (ON)

// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/stretchr/testify/require"
)

type testRPCCall struct {
	nodeIdx int
	method  string
	params  string
	delay   time.Duration
}

type checkDBCall struct {
	call1idx int
	call2idx int
	field    string
}

var tests = []testRPCCall{
	{nodeIdx: 0, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 1, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 2, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 0, method: "chain_getHeader", params: "[]", delay: time.Second * 10},
	{nodeIdx: 1, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 2, method: "chain_getHeader", params: "[]", delay: 0},
}

var checks = []checkDBCall{
	{call1idx: 0, call2idx: 1, field: "parentHash"},
	{call1idx: 0, call2idx: 2, field: "parentHash"},
	{call1idx: 3, call2idx: 4, field: "parentHash"},
	{call1idx: 3, call2idx: 5, field: "parentHash"},
}

// this starts nodes and runs RPC calls (which loads db)
func TestCalls(t *testing.T) {
	if utils.MODE != "sync" {
		t.Skip("MODE != 'sync', skipping stress test")
	}

	err := utils.BuildGossamer()
	require.NoError(t, err)

	ctx := context.Background()

	const qtyNodes = 3
	tomlConfig := config.Default()
	framework, err := utils.InitFramework(ctx, t, qtyNodes, tomlConfig)

	require.NoError(t, err)

	nodesCtx, nodesCancel := context.WithCancel(ctx)

	runtimeErrors, startErr := framework.StartNodes(nodesCtx, t)

	t.Cleanup(func() {
		nodesCancel()
		for _, runtimeError := range runtimeErrors {
			<-runtimeError
		}
	})

	require.NoError(t, startErr)

	for _, call := range tests {
		time.Sleep(call.delay)

		const callRPCTimeout = time.Second
		callRPCCtx, cancel := context.WithTimeout(ctx, callRPCTimeout)

		_, err := framework.CallRPC(callRPCCtx, call.nodeIdx, call.method, call.params)

		cancel()

		require.NoError(t, err)
	}

	framework.PrintDB()

	// test check
	for _, check := range checks {
		res := framework.CheckEqual(check.call1idx, check.call2idx, check.field)
		require.True(t, res)
	}
}
