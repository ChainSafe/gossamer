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

package spec

import (
	"github.com/stretchr/testify/require"
	"os/exec"
	"testing"
	"time"

	rpc "github.com/ChainSafe/gossamer/tests/rpc"
)

func TestPaymentRPC(t *testing.T) {
	testsCases := []struct {
		description string
		method      string
		expected    interface{}
		skip        bool
	}{
		{ //TODO
			description: "test payment_queryInfo",
			method:      "payment_queryInfo",
			skip:        true,
		},
	}

	t.Log("going to Bootstrap Gossamer node")

	localPidList, err := rpc.Bootstrap(t, make([]*exec.Cmd, 1))

	//use only first server for tests
	currentPort := "8540"
	require.Nil(t, err)

	time.Sleep(time.Second) // give server a second to start

	for _, test := range testsCases {
		t.Run(test.description, func(t *testing.T) {
			if test.skip {
				t.Skip("RPC endpoint not yet implemented")
				return
			}

			respBody := rpc.PostRPC(t, test.method, "http://"+rpc.GOSSAMER_NODE_HOST+":"+currentPort)

			target := rpc.DecodeRPC(t, respBody, test.method)

			require.NotNil(t, target)

		})
	}

	t.Log("going to TearDown Gossamer node")

	errList := rpc.TearDown(t, localPidList)
	require.Len(t, errList, 0)
}
