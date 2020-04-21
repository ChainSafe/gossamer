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
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	rpc "github.com/ChainSafe/gossamer/tests/rpc"
)

func TestStateRPC(t *testing.T) {
	testsCases := []struct {
		description string
		method      string
		expected    interface{}
		skip        bool
	}{
		{ //TODO
			description: "test state_call",
			method:      "state_call",
			skip:        true,
		},
		{ //TODO
			description: "test state_getPairs",
			method:      "state_getPairs",
			skip:        true,
		},
		{ //TODO
			description: "test state_getKeysPaged",
			method:      "state_getKeysPaged",
			skip:        true,
		},
		{ //TODO
			description: "test state_getStorage",
			method:      "state_getStorage",
			skip:        true,
		},
		{ //TODO
			description: "test state_getStorageHash",
			method:      "state_getStorageHash",
			skip:        true,
		},
		{ //TODO
			description: "test state_getStorageSize",
			method:      "state_getStorageSize",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildKeys",
			method:      "state_getChildKeys",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildStorage",
			method:      "state_getChildStorage",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildStorageHash",
			method:      "state_getChildStorageHash",
			skip:        true,
		},
		{ //TODO
			description: "test state_getChildStorageSize",
			method:      "state_getChildStorageSize",
			skip:        true,
		},
		{ //TODO
			description: "test state_getRuntimeVersion",
			method:      "state_getRuntimeVersion",
			skip:        true,
		},
		{ //TODO
			description: "test state_queryStorage",
			method:      "state_queryStorage",
			skip:        true,
		},
	}

	t.Log("going to Bootstrap Gossamer node")

	localPidList, err := rpc.Bootstrap(t, make([]*exec.Cmd, 1))
	require.Nil(t, err)

	time.Sleep(time.Second) // give server a second to start

	for _, test := range testsCases {
		t.Run(test.description, func(t *testing.T) {
			if test.skip {
				t.Skip("RPC endpoint not yet implemented")
				return
			}

			respBody, err := rpc.PostRPC(t, test.method, "http://"+rpc.GOSSAMER_NODE_HOST+":"+currentPort, "{}")
			require.Nil(t, err)

			target := rpc.DecodeRPC(t, respBody, test.method)

			require.NotNil(t, target)

		})
	}

	t.Log("going to TearDown Gossamer node")

	errList := rpc.TearDown(t, localPidList)
	require.Len(t, errList, 0)
}
