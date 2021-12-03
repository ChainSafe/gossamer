// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

var testCases = []struct {
	rpcDataBody string
	expected    string
}{
	{
		rpcDataBody: fmt.Sprintf(
			`{"jsonrpc":"2.0","method":"%s","params":[],"id":1}`,
			"chain_getFinalisedHead",
		),
		expected: "chain.GetFinalizedHead",
	},
	{
		rpcDataBody: fmt.Sprintf(
			`{"jsonrpc":"2.0","method":"%s","params":["..."],"id":1}`,
			"account_nextIndex",
		),
		expected: "system.AccountNextIndex",
	},
	{
		rpcDataBody: fmt.Sprintf(
			`{"jsonrpc":"2.0","method":"%s","params":[50, "0x64", 200],"id":1}`,
			"chain_getHead",
		),
		expected: "chain.GetBlockHash",
	},
}

func TestAliasesMethodReplace(t *testing.T) {
	c := NewDotUpCodec()

	for _, test := range testCases {
		buf := new(bytes.Buffer)
		buf.Write([]byte(test.rpcDataBody))

		testRequest, err := http.NewRequest(http.MethodPost, "http://fake_url", buf)
		require.NoError(t, err)

		codecRequest := c.NewRequest(testRequest)
		got, err := codecRequest.Method()
		require.NoError(t, err)
		require.Equal(t, test.expected, got)
	}
}
