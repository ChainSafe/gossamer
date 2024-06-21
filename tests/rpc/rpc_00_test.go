// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	//if utils.MODE != "rpc" {
	//fmt.Println("Going to skip RPC suite tests")
	//os.Exit(0)
	//}

	err := utils.BuildGossamer()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

type testCase struct {
	description string
	method      string
	params      string
	expected    interface{}
}

func fetchWithTimeout(ctx context.Context, t *testing.T,
	method, params string, target interface{}) {
	t.Helper()

	getResponseCtx, getResponseCancel := context.WithTimeout(ctx, time.Second)
	defer getResponseCancel()
	err := getResponse(getResponseCtx, method, params, target)
	require.NoError(t, err)
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
