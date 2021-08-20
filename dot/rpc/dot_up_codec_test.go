// Copyright 2019 ChainSafe Systems (ON) Corp.
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
