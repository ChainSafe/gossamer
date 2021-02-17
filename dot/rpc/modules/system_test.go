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
	"github.com/ChainSafe/gossamer/lib/genesis"
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

func (s *mockSyncer) ProcessBlockData(_ []*types.BlockData) error {
	return nil
}

func (s *mockSyncer) HandleBlockAnnounce(msg *network.BlockAnnounceMessage) error {
	return nil
}

func (s *mockSyncer) HandleBlockAnnounceHandshake(num *big.Int) []*network.BlockRequestMessage {
	return nil
}

type mockBlockState struct{}

func (s *mockBlockState) BestBlockHeader() (*types.Header, error) {
	return genesisHeader, nil
}

func (s *mockBlockState) BestBlockNumber() (*big.Int, error) {
	return big.NewInt(0), nil
}

func (s *mockBlockState) GenesisHash() common.Hash {
	return genesisHeader.Hash()
}

func (s *mockBlockState) HasBlockBody(_ common.Hash) (bool, error) {
	return false, nil
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

	err = srv.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = srv.Stop()
	})

	return srv
}

// Test RPC's System.Health() response
func TestSystemModule_Health(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil)

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
	sys := NewSystemModule(net, nil)

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
	sys := NewSystemModule(net, nil)

	res := &SystemPeersResponse{}
	err := sys.Peers(nil, nil, res)
	require.NoError(t, err)

	if len(*res) != len(testPeers) {
		t.Errorf("System.Peers: expected: %+v got: %+v\n", testPeers, *res)
	}
}

func TestSystemModule_NodeRoles(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil)
	expected := []interface{}{"Full"}

	var res []interface{}
	err := sys.NodeRoles(nil, nil, &res)
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

var testSystemInfo = &types.SystemInfo{
	SystemName:    "gossamer",
	SystemVersion: "0",
}

var testGenesisData = &genesis.Data{
	Name:      "Gossamer",
	ID:        "gssmr",
	ChainType: "Local",
}

type mockSystemAPI struct {
	info    *types.SystemInfo
	genData *genesis.Data
}

func newMockSystemAPI() *mockSystemAPI {
	return &mockSystemAPI{
		info:    testSystemInfo,
		genData: testGenesisData,
	}
}

func (api *mockSystemAPI) SystemName() string {
	return api.info.SystemName
}

func (api *mockSystemAPI) SystemVersion() string {
	return api.info.SystemVersion
}

func (api *mockSystemAPI) NodeName() string {
	return api.genData.Name
}

func (api *mockSystemAPI) Properties() map[string]interface{} {
	return nil
}

func (api *mockSystemAPI) ChainType() string {
	return api.genData.ChainType
}

func TestSystemModule_Chain(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI())

	res := new(string)
	err := sys.Chain(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testGenesisData.Name, *res)
}

func TestSystemModule_ChainType(t *testing.T) {
	api := newMockSystemAPI()

	sys := NewSystemModule(nil, api)

	res := new(string)
	sys.ChainType(nil, nil, res)
	require.Equal(t, api.genData.ChainType, *res)
}

func TestSystemModule_Name(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI())

	res := new(string)
	err := sys.Name(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemName, *res)
}

func TestSystemModule_Version(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI())

	res := new(string)
	err := sys.Version(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemVersion, *res)
}

func TestSystemModule_Properties(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI())
	expected := map[string]interface{}(nil)
	res := new(interface{})
	err := sys.Properties(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, expected, *res)
}
