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
	"reflect"
	"strconv"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/tests/utils"

	"github.com/stretchr/testify/require"
)

func TestStableNetworkRPC(t *testing.T) {
	if utils.MODE != "stable" {
		t.Skip("Integration tests are disabled, going to skip.")
	}
	t.Log("Running NetworkAPI tests with HOSTNAME=" + utils.HOSTNAME + " and PORT=" + utils.PORT)

	networkSize, err := strconv.Atoi(utils.NETWORK_SIZE)
	if err != nil {
		networkSize = 0
	}

	testsCases := []*testCase{
		{
			description: "test system_health",
			method:      "system_health",
			expected: modules.SystemHealthResponse{
				Peers:           networkSize - 1,
				IsSyncing:       true,
				ShouldHavePeers: true,
			},
		},
		{
			description: "test system_network_state",
			method:      "system_networkState",
			expected: modules.SystemNetworkStateResponse{
				NetworkState: modules.NetworkStateString{
					PeerID: "",
				},
			},
		},
		{
			description: "test system_peers",
			method:      "system_peers",
			expected:    modules.SystemPeersResponse{},
		},
	}

	for _, test := range testsCases {
		t.Run(test.description, func(t *testing.T) {
			respBody, err := utils.PostRPC(test.method, "http://"+utils.HOSTNAME+":"+utils.PORT, "{}")
			require.Nil(t, err)

			target := reflect.New(reflect.TypeOf(test.expected)).Interface()
			err = utils.DecodeRPC(t, respBody, target)
			require.Nil(t, err)

			switch v := target.(type) {
			case *modules.SystemHealthResponse:
				t.Log("Will assert SystemHealthResponse", "target", target)

				require.Equal(t, test.expected.(modules.SystemHealthResponse).IsSyncing, v.IsSyncing)
				require.Equal(t, test.expected.(modules.SystemHealthResponse).ShouldHavePeers, v.ShouldHavePeers)
				require.GreaterOrEqual(t, v.Peers, test.expected.(modules.SystemHealthResponse).Peers)

			case *modules.SystemNetworkStateResponse:
				t.Log("Will assert SystemNetworkStateResponse", "target", target)

				require.NotNil(t, v.NetworkState)
				require.NotNil(t, v.NetworkState.PeerID)

			case *modules.SystemPeersResponse:
				t.Log("Will assert SystemPeersResponse", "target", target)

				require.NotNil(t, *v)
				require.GreaterOrEqual(t, len(*v), networkSize-2)

				for _, vv := range *v {
					require.NotNil(t, vv.PeerID)
					require.NotNil(t, vv.Roles)
					require.NotNil(t, vv.BestHash)
					require.NotNil(t, vv.BestNumber)
				}
			}
		})
	}
}
