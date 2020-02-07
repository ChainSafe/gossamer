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

package state

import (
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/stretchr/testify/require"
)

var testHealth = &common.Health{}
var testNetworkState = &common.NetworkState{}
var testPeers = &[]common.PeerInfo{}

// test state.Network
func TestNetworkState(t *testing.T) {
	state := newTestService(t)

	// test state.Network.Health()
	health, err := state.Network.GetHealth()
	require.Nil(t, err)

	if health != testHealth {
		t.Errorf("System.Health - expected %+v got: %+v\n", testHealth, health)
	}

	// test state.Network.NetworkState()
	networkState, err := state.Network.GetNetworkState()
	require.Nil(t, err)

	if networkState != testNetworkState {
		t.Errorf("System.NetworkState - expected %+v got: %+v\n", testNetworkState, networkState)
	}

	// test state.Network.Peers()
	peers, err := state.Network.GetPeers()
	require.Nil(t, err)

	require.Equal(t, peers, testPeers)
}
