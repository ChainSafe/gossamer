// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/stretchr/testify/require"
)

var (
	currentPort = strconv.Itoa(utils.BaseRPCPort)
	rpcSuite    = "rpc"
)

func TestMain(m *testing.M) {
	_, _ = fmt.Fprintln(os.Stdout, "Going to start RPC suite test")

	utils.CreateDefaultConfig()
	defer os.Remove(utils.ConfigDefault)

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

type testCase struct {
	description string
	method      string
	params      string
	expected    interface{}
	skip        bool
}

func getResponse(t *testing.T, test *testCase) interface{} {
	if test.skip {
		t.Skip("RPC endpoint not yet implemented")
		return nil
	}

	respBody, err := utils.PostRPC(test.method, utils.NewEndpoint(currentPort), test.params)
	require.Nil(t, err)

	target := reflect.New(reflect.TypeOf(test.expected)).Interface()
	err = utils.DecodeRPC(t, respBody, target)
	require.Nil(t, err, "Could not DecodeRPC", string(respBody))

	require.NotNil(t, target)

	return target
}
