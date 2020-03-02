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
	"github.com/ChainSafe/gossamer/lib/scale"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

var testHealth = &common.Health{}
var testNetworkState = &common.NetworkState{}
var testPeers = &[]common.PeerInfo{{ PeerID: "ID1", BestHash: common.Hash{}, BestNumber: 1, ProtocolVersion:2, Roles: 0x03},
	{ PeerID: "ID40", BestHash: common.Hash{}, BestNumber: 50, ProtocolVersion:60, Roles: 0x70},
}
//var testPeers = &[]common.PeerInfo{}


// test state.Network
func TestNetworkState(t *testing.T) {
	state := newTestService(t)

	header := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
	}

	err := state.Initialize(header, trie.NewEmptyTrie(nil))
	if err != nil {
		t.Fatal(err)
	}

	err = state.Start()
	if err != nil {
		t.Fatal(err)
	}

	err = state.Network.SetHealth(testHealth)
	require.Nil(t, err)

	health, err := state.Network.GetHealth()
	require.Nil(t, err)

	require.Equal(t, health, testHealth)

	err = state.Network.SetNetworkState(testNetworkState)
	require.Nil(t, err)

	networkState, err := state.Network.GetNetworkState()
	require.Nil(t, err)

	require.Equal(t, networkState, testNetworkState)

	err = state.Network.SetPeers(testPeers)
	require.Nil(t, err)

	peers, err := state.Network.GetPeers()
	require.Nil(t, err)

	require.Equal(t, peers, testPeers)
}

func TestEncodePeers(t *testing.T) {
	t.Logf("before encoded %v", testPeers)
	// this fails with: panic: reflect: call of reflect.Value.NumField on slice Value
	// at: encode.go 254
	//res, err := scale.Encode(testPeers)
	//t.Logf("encoded %v, err: %v", res, err)

	// this works, but should we be passing a pointer to this?
	// and we had to update encode.go encodeArray
	res, err := scale.Encode(*testPeers)
	t.Logf("encoded %v, err: %v", res, err)
}

func TestDecodePeer(t *testing.T) {
	data := []byte{8, 12, 73, 68, 49, 3, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 16, 73, 68, 52, 48, 112, 60, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 0, 0, 0, 0, 0, 0, 0}
	res := new ([]common.PeerInfo)

	// this fails with: panic: reflect.MakeSlice of non-slice type
	// at decode.go 328
	//err := scale.DecodePtr(data, res)
	//t.Logf("decoded %v, err: %v", res, err)

	// This will call common.PeerInfo Decode method, see common.types.go for comments
	// on that.
	// It's not changing res
	err := scale.DecodePtr(data, *res)
	t.Logf("decoded %v, err: %v", *res, err)
}