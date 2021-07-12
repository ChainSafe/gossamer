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
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	log "github.com/ChainSafe/log15"
	"github.com/btcsuite/btcutil/base58"
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

func newNetworkService(t *testing.T) *network.Service {
	testDir := path.Join(os.TempDir(), "test_data")

	cfg := &network.Config{
		BlockState:         network.NewMockBlockState(nil),
		BasePath:           testDir,
		Syncer:             network.NewMockSyncer(),
		TransactionHandler: network.NewMockTransactionHandler(),
	}

	srv, err := network.NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = srv.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = srv.Stop()
		time.Sleep(time.Second)
		err = os.RemoveAll(cfg.BasePath)
		if err != nil {
			fmt.Printf("failed to remove path %s : %s\n", cfg.BasePath, err)
		}
	})

	return srv
}

// Test RPC's System.Health() response
func TestSystemModule_Health(t *testing.T) {
	net := newNetworkService(t)
	net.Stop()
	sys := NewSystemModule(net, nil, nil, nil, nil)

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
	sys := NewSystemModule(net, nil, nil, nil, nil)

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
	net.Stop()
	sys := NewSystemModule(net, nil, nil, nil, nil)

	res := &SystemPeersResponse{}
	err := sys.Peers(nil, nil, res)
	require.NoError(t, err)

	if len(*res) != len(testPeers) {
		t.Errorf("System.Peers: expected: %+v got: %+v\n", testPeers, *res)
	}
}

func TestSystemModule_NodeRoles(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil, nil, nil, nil)
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

func (api *mockSystemAPI) ChainName() string {
	return api.genData.Name
}
func (api *mockSystemAPI) Properties() map[string]interface{} {
	return nil
}

func (api *mockSystemAPI) ChainType() string {
	return api.genData.ChainType
}

func TestSystemModule_Chain(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI(), nil, nil, nil)

	res := new(string)
	err := sys.Chain(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testGenesisData.Name, *res)
}

func TestSystemModule_ChainType(t *testing.T) {
	api := newMockSystemAPI()

	sys := NewSystemModule(nil, api, nil, nil, nil)

	res := new(string)
	sys.ChainType(nil, nil, res)
	require.Equal(t, api.genData.ChainType, *res)
}

func TestSystemModule_Name(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI(), nil, nil, nil)

	res := new(string)
	err := sys.Name(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemName, *res)
}

func TestSystemModule_Version(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI(), nil, nil, nil)

	res := new(string)
	err := sys.Version(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemVersion, *res)
}

func TestSystemModule_Properties(t *testing.T) {
	sys := NewSystemModule(nil, newMockSystemAPI(), nil, nil, nil)

	expected := map[string]interface{}(nil)

	res := new(interface{})
	err := sys.Properties(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, expected, *res)
}

func TestSystemModule_AccountNextIndex_StoragePending(t *testing.T) {
	sys := setupSystemModule(t)
	expectedStored := U64Response(uint64(3))

	res := new(U64Response)
	req := StringRequest{
		String: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
	}
	err := sys.AccountNextIndex(nil, &req, res)
	require.NoError(t, err)
	require.Equal(t, expectedStored, *res)

	// extrinsic for transfer signed by alice, nonce 4 (created with polkadot.js/api test_transaction)
	signedExt := common.MustHexToBytes("0xad018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0146d0050619728683af4e9659bf202aeb2b8b13b48a875adb663f449f1a71453903546f3252193964185eb91c482cf95caf327db407d57ebda95046b5ef890187001000000108abcd")
	vtx := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(signedExt),
		Validity:  new(transaction.Validity),
	}
	expectedPending := U64Response(uint64(4))
	sys.txStateAPI.AddToPool(vtx)

	err = sys.AccountNextIndex(nil, &req, res)
	require.NoError(t, err)
	require.Equal(t, expectedPending, *res)
}

func TestSystemModule_AccountNextIndex_Storage(t *testing.T) {
	sys := setupSystemModule(t)
	expectedStored := U64Response(uint64(3))

	res := new(U64Response)
	req := StringRequest{
		String: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
	}
	err := sys.AccountNextIndex(nil, &req, res)
	require.NoError(t, err)

	require.Equal(t, expectedStored, *res)
}

func TestSystemModule_AccountNextIndex_Pending(t *testing.T) {
	sys := setupSystemModule(t)
	res := new(U64Response)
	req := StringRequest{
		String: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
	}

	// extrinsic for transfer signed by alice, nonce 4 (created with polkadot.js/api test_transaction)
	signedExt := common.MustHexToBytes("0xad018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0146d0050619728683af4e9659bf202aeb2b8b13b48a875adb663f449f1a71453903546f3252193964185eb91c482cf95caf327db407d57ebda95046b5ef890187001000000108abcd")
	vtx := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(signedExt),
		Validity:  new(transaction.Validity),
	}
	expectedPending := U64Response(uint64(4))
	sys.txStateAPI.AddToPool(vtx)

	err := sys.AccountNextIndex(nil, &req, res)
	require.NoError(t, err)
	require.Equal(t, expectedPending, *res)
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
		Nonce: 3,
		//RefCount: 0,
		Data: struct {
			Free       *scale.Uint128
			Reserved   *scale.Uint128
			MiscFrozen *scale.Uint128
			FreeFrozen *scale.Uint128
		}{
			Free:       scale.MustNewUint128(big.NewInt(0)),
			Reserved:   scale.MustNewUint128(big.NewInt(0)),
			MiscFrozen: scale.MustNewUint128(big.NewInt(0)),
			FreeFrozen: scale.MustNewUint128(big.NewInt(0)),
		},
	}

	aliceAcctEncoded, err := scale.Marshal(aliceAcctInfo)
	require.NoError(t, err)
	ts.Set(aliceAcctStoKey, aliceAcctEncoded)

	err = chain.Storage.StoreTrie(ts, nil)
	require.NoError(t, err)
	err = chain.Block.AddBlock(&types.Block{
		Header: &types.Header{
			Number:     big.NewInt(1),
			ParentHash: chain.Block.BestBlockHash(),
			StateRoot:  ts.MustRoot(),
		},
		Body: &types.Body{},
	})
	require.NoError(t, err)

	core := newCoreService(t, chain)
	// TODO (ed) add transactions to txQueue and add test for those
	txQueue := state.NewTransactionState()
	return NewSystemModule(net, nil, core, chain.Storage, txQueue)
}

type mockNetwork struct{}

func (n *mockNetwork) SendMessage(_ network.NotificationsMessage) {}

func newCoreService(t *testing.T, srvc *state.Service) *core.Service {
	// setup service
	tt := trie.NewEmptyTrie()
	rt := wasmer.NewTestInstanceWithTrie(t, runtime.NODE_RUNTIME, tt, log.LvlInfo)
	ks := keystore.NewGlobalKeystore()
	t.Cleanup(func() {
		rt.Stop()
	})

	// insert alice key for testing
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Acco.Insert(kr.Alice())

	if srvc == nil {
		srvc = newTestStateService(t)
	}

	cfg := &core.Config{
		Runtime:              rt,
		Keystore:             ks,
		TransactionState:     srvc.Transaction,
		BlockState:           srvc.Block,
		StorageState:         srvc.Storage,
		EpochState:           srvc.Epoch,
		Network:              &mockNetwork{},
		CodeSubstitutedState: srvc.Base,
	}

	return core.NewTestService(t, cfg)
}

func TestLocalPeerId(t *testing.T) {
	peerID := "12D3KooWBrwpqLE9Z23NEs59m2UHUs9sGYWenxjeCk489Xq7SG2h"
	encoded := base58.Encode([]byte(peerID))

	state := common.NetworkState{
		PeerID: peerID,
	}

	mocknetAPI := new(mocks.MockNetworkAPI)
	mocknetAPI.On("NetworkState").Return(state)

	sysmodules := new(SystemModule)
	sysmodules.networkAPI = mocknetAPI

	var res string
	err := sysmodules.LocalPeerId(nil, nil, &res)
	require.NoError(t, err)

	require.Equal(t, res, encoded)
}
