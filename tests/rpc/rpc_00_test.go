// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"fmt"

	"github.com/ChainSafe/gossamer/tests/utils/rpc"
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

func getResponse(ctx context.Context, method, params string, target interface{}) (err error) {
	const currentPort = "8540"
	endpoint := rpc.NewEndpoint(currentPort)
	respBody, err := rpc.Post(ctx, endpoint, method, params)
	if err != nil {
		return fmt.Errorf("cannot RPC post: %w", err)
	}

	err = rpc.Decode(respBody, &target)
	if err != nil {
		return fmt.Errorf("cannot decode RPC response: %w", err)
	}

	return nil
}
