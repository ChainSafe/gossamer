// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	"github.com/stretchr/testify/require"
)

var (
	rpcSuite = "rpc"
)

type testCase struct {
	description string
	method      string
	params      string
	expected    interface{}
	skip        bool
}

func getResponse(ctx context.Context, t *testing.T, test *testCase) interface{} {
	if test.skip {
		t.Skip("RPC endpoint not yet implemented")
		return nil
	}

	const currentPort = "8540"
	endpoint := rpc.NewEndpoint(currentPort)
	respBody, err := rpc.Post(ctx, endpoint, test.method, test.params)
	require.NoError(t, err)

	target := reflect.New(reflect.TypeOf(test.expected)).Interface()
	err = rpc.Decode(respBody, target)
	require.NoError(t, err)

	require.NotNil(t, target)

	return target
}
