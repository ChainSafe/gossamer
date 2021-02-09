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

package modules

import (
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/stretchr/testify/require"
)

var (
	testHealth = common.Health{
		Peers:           0,
		IsSyncing:       true,
		ShouldHavePeers: true,
	}
	testPeers = []common.PeerInfo{}
)

type mockSyncer struct{}

func (s *mockSyncer) CreateBlockResponse(msg *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	return nil, nil
}

func (s *mockSyncer) HandleBlockResponse(msg *network.BlockResponseMessage) *network.BlockRequestMessage {
	return nil
}

func (s *mockSyncer) HandleBlockAnnounce(msg *network.BlockAnnounceMessage) *network.BlockRequestMessage {
	return nil
}

func (s *mockSyncer) HandleBlockAnnounceHandshake(num *big.Int) *network.BlockRequestMessage {
	return nil
}

type mockBlockState struct{}

func (s *mockBlockState) BestBlockHeader() (*types.Header, error) {
	return genesisHeader, nil
}

func (s *mockBlockState) GenesisHash() common.Hash {
	return genesisHeader.Hash()
}

func (s *mockSyncer) IsSynced() bool {
	return false
}

type mockTransactionHandler struct{}

func (h *mockTransactionHandler) HandleTransactionMessage(_ *network.TransactionMessage) error {
	return nil
}

func newNetworkService(t *testing.T) *network.Service {
	testDir := path.Join(os.TempDir(), "test_data")

	cfg := &network.Config{
		BlockState:         &mockBlockState{},
		BasePath:           testDir,
		Syncer:             &mockSyncer{},
		TransactionHandler: &mockTransactionHandler{},
	}

	srv, err := network.NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return srv
}

// Test RPC's System.Health() response
func TestSystemModule_Health(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil, nil, nil)

	res := &SystemHealthResponse{}
	err := sys.Health(nil, nil, res)
	require.NoError(t, err)

	if *res != SystemHealthResponse(testHealth) {
		t.Errorf("System.Health.: expected: %+v got: %+v\n", testHealth, *res)
	}
}

// Test RPC's System.NetworkState() response
func TestSystemModule_NetworkState(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil, nil, nil)

	res := &SystemNetworkStateResponse{}
	err := sys.NetworkState(nil, nil, res)
	require.NoError(t, err)

	testNetworkState := net.NetworkState()

	if res.NetworkState.PeerID != testNetworkState.PeerID {
		t.Errorf("System.NetworkState: expected: %+v got: %+v\n", testNetworkState, res.NetworkState)
	}
}

// Test RPC's System.Peers() response
func TestSystemModule_Peers(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil, nil, nil)

	res := &SystemPeersResponse{}
	err := sys.Peers(nil, nil, res)
	require.NoError(t, err)

	if len(*res) != len(testPeers) {
		t.Errorf("System.Peers: expected: %+v got: %+v\n", testPeers, *res)
	}
}

func TestSystemModule_NodeRoles(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil, nil, nil)
	expected := []interface{}{"Full"}

	var res []interface{}
	err := sys.NodeRoles(nil, nil, &res)
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

var testSystemInfo = &types.SystemInfo{
	SystemName:       "gossamer",
	SystemVersion:    "0",
	NodeName:         "gssmr",
	SystemProperties: make(map[string]interface{}),
	ChainType:        "Local",
}

type mockSystemAPI struct {
	info *types.SystemInfo
}

func newMockSystemAPI() *mockSystemAPI {
	return &mockSystemAPI{
		info: testSystemInfo,
	}
}

func (api *mockSystemAPI) SystemName() string {
	return api.info.SystemName
}

func (api *mockSystemAPI) SystemVersion() string {
	return api.info.SystemVersion
}

func (api *mockSystemAPI) NodeName() string {
	return api.info.NodeName
}

func (api *mockSystemAPI) Properties() map[string]interface{} {
	return api.info.SystemProperties
}

func (api *mockSystemAPI) ChainType() string {
	return api.info.ChainType
}

func TestSystemModule_Chain(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI(), nil, nil)

	res := new(string)
	err := sys.Chain(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.NodeName, *res)
}

func TestSystemModule_ChainType(t *testing.T) {
	api := newMockSystemAPI()

	sys := NewSystemModule(nil, api, nil, nil)

	res := new(string)
	sys.ChainType(nil, nil, res)
	require.Equal(t, api.info.ChainType, *res)
}

func TestSystemModule_Name(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI(), nil, nil)

	res := new(string)
	err := sys.Name(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemName, *res)
}

func TestSystemModule_Version(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI(), nil, nil)

	res := new(string)
	err := sys.Version(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemVersion, *res)
}

func TestSystemModule_Properties(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI(), nil, nil)

	res := new(interface{})
	err := sys.Properties(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemProperties, *res)
}

func TestSystemModule_AccountNextIndex(t *testing.T) {
	sys := setupSystemModule(t)
	expected := U64Response(uint64(10))

	res := new(U64Response)
	req := StringRequest{
		String: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
	}
	err := sys.AccountNextIndex(nil, &req, res)
	require.NoError(t, err)

	require.Equal(t, expected, *res)

}

func setupSystemModule(t *testing.T) *SystemModule {
	// setup service
	net := newNetworkService(t)
	chain := newTestStateService(t)
	// init storage with test data
	ts, err := chain.Storage.TrieState(nil)
	require.NoError(t, err)

	aliceAcctStoKey, err := common.HexToBytes("0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")
	require.NoError(t, err)
	aliceAcctInfo := types.AccountInfo{
		Nonce:    10,
		RefCount: 0,
		Data: struct {
			Free       common.Uint128
			Reserved   common.Uint128
			MiscFrozen common.Uint128
			FreeFrozen common.Uint128
		}{},
	}
	aliceAcctEncoded, err := scale.Encode(aliceAcctInfo)
	require.NoError(t, err)
	err = ts.Set(aliceAcctStoKey, aliceAcctEncoded)
	require.NoError(t, err)

	err = chain.Storage.StoreTrie(ts)
	require.NoError(t, err)

	core := newCoreService(t, chain)
	return NewSystemModule(net, nil, core, chain.Storage)
}
