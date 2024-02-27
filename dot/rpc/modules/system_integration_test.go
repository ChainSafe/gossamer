// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package modules

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	testHealth = common.Health{
		Peers:           0,
		IsSyncing:       true,
		ShouldHavePeers: true,
	}
	testPeers []common.PeerInfo
)

func newNetworkService(t *testing.T) *network.Service {
	ctrl := gomock.NewController(t)

	blockStateMock := NewMockBlockState(ctrl)
	blockStateMock.EXPECT().
		BestBlockHeader().
		Return(types.NewEmptyHeader(), nil).AnyTimes()
	blockStateMock.EXPECT().
		GetHighestFinalisedHeader().
		Return(types.NewEmptyHeader(), nil).AnyTimes()

	syncerMock := NewMockSyncer(ctrl)

	transactionHandlerMock := NewMockTransactionHandler(ctrl)
	transactionHandlerMock.EXPECT().TransactionsCount().Return(0).AnyTimes()

	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	cfg := &network.Config{
		BasePath:           t.TempDir(),
		SlotDuration:       time.Second,
		BlockState:         blockStateMock,
		Port:               0,
		Syncer:             syncerMock,
		TransactionHandler: transactionHandlerMock,
		Telemetry:          telemetryMock,
	}

	srv, err := network.NewService(cfg)
	require.NoError(t, err)

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
	ctrl := gomock.NewController(t)

	networkMock := mocks.NewMockNetworkAPI(ctrl)
	networkMock.EXPECT().Health().Return(testHealth)

	sys := NewSystemModule(networkMock, nil, nil, nil, nil, nil, nil)

	res := &SystemHealthResponse{}
	err := sys.Health(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, SystemHealthResponse(testHealth), *res)
}

// Test RPC's System.NetworkState() response
func TestSystemModule_NetworkState(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil, nil, nil, nil, nil, nil)

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
	sys := NewSystemModule(net, nil, nil, nil, nil, nil, nil)

	res := &SystemPeersResponse{}
	err := sys.Peers(nil, nil, res)
	require.NoError(t, err)

	if len(*res) != len(testPeers) {
		t.Errorf("System.Peers: expected: %+v got: %+v\n", testPeers, *res)
	}
}

func TestSystemModule_NodeRoles(t *testing.T) {
	net := newNetworkService(t)
	sys := NewSystemModule(net, nil, nil, nil, nil, nil, nil)
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

func TestSystemModule_Chain(t *testing.T) {
	ctrl := gomock.NewController(t)

	api := mocks.NewMockSystemAPI(ctrl)
	api.EXPECT().ChainName().Return(testGenesisData.Name)
	sys := NewSystemModule(nil, api, nil, nil, nil, nil, nil)

	res := new(string)
	err := sys.Chain(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testGenesisData.Name, *res)
}

func TestSystemModule_ChainType(t *testing.T) {
	ctrl := gomock.NewController(t)

	api := mocks.NewMockSystemAPI(ctrl)
	api.EXPECT().ChainType().Return(testGenesisData.ChainType)

	sys := NewSystemModule(nil, api, nil, nil, nil, nil, nil)

	res := new(string)
	sys.ChainType(nil, nil, res)
	require.Equal(t, testGenesisData.ChainType, *res)
}
func TestSystemModule_Name(t *testing.T) {
	ctrl := gomock.NewController(t)

	api := mocks.NewMockSystemAPI(ctrl)
	api.EXPECT().SystemName().Return(testSystemInfo.SystemName)
	sys := NewSystemModule(nil, api, nil, nil, nil, nil, nil)

	res := new(string)
	err := sys.Name(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemName, *res)
}

func TestSystemModule_Version(t *testing.T) {
	ctrl := gomock.NewController(t)

	api := mocks.NewMockSystemAPI(ctrl)
	api.EXPECT().SystemVersion().Return(testSystemInfo.SystemVersion)

	sys := NewSystemModule(nil, api, nil, nil, nil, nil, nil)

	res := new(string)
	err := sys.Version(nil, nil, res)
	require.NoError(t, err)
	require.Equal(t, testSystemInfo.SystemVersion, *res)
}

func TestSystemModule_Properties(t *testing.T) {
	ctrl := gomock.NewController(t)

	api := mocks.NewMockSystemAPI(ctrl)
	api.EXPECT().Properties().Return(nil)

	sys := NewSystemModule(nil, api, nil, nil, nil, nil, nil)

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
	signedExt := common.MustHexToBytes("0xad018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a" +
		"56da27d0146d0050619728683af4e9659bf202aeb2b8b13b48a875adb663f449f1a71453903546f3252193964185eb91c482" +
		"cf95caf327db407d57ebda95046b5ef890187001000000108abcd")
	vtx := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(signedExt),
		Validity:  new(transaction.Validity),
	}
	expectedPending := U64Response(uint64(4))
	sys.txStateAPI.(*state.TransactionState).AddToPool(vtx)

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
	signedExt := common.MustHexToBytes("0xad018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a56" +
		"84e7a56da27d0146d0050619728683af4e9659bf202aeb2b8b13b48a875adb663f449f1a71453903546f325219396418" +
		"5eb91c482cf95caf327db407d57ebda95046b5ef890187001000000108abcd")
	vtx := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(signedExt),
		Validity:  new(transaction.Validity),
	}
	expectedPending := U64Response(uint64(4))
	sys.txStateAPI.(*state.TransactionState).AddToPool(vtx)

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

	aliceAcctStoKey, err := common.HexToBytes("0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c" +
		"0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")
	require.NoError(t, err)

	aliceAcctInfo := types.AccountInfo{
		Nonce: 3,
		//RefCount: 0,
		Data: types.AccountData{
			Free:       scale.MustNewUint128(big.NewInt(0)),
			Reserved:   scale.MustNewUint128(big.NewInt(0)),
			MiscFrozen: scale.MustNewUint128(big.NewInt(0)),
			FreeFrozen: scale.MustNewUint128(big.NewInt(0)),
		},
	}

	aliceAcctEncoded, err := scale.Marshal(aliceAcctInfo)
	require.NoError(t, err)
	ts.Put(aliceAcctStoKey, aliceAcctEncoded)

	err = chain.Storage.StoreTrie(ts, nil)
	require.NoError(t, err)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	err = chain.Block.AddBlock(&types.Block{
		Header: types.Header{
			Number:     3,
			ParentHash: chain.Block.BestBlockHash(),
			StateRoot:  ts.MustRoot(trie.V0),
			Digest:     digest,
		},
		Body: types.Body{},
	})
	require.NoError(t, err)

	core := newCoreService(t, chain)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)

	telemetryMock.
		EXPECT().
		SendMessage(gomock.Any()).
		AnyTimes()

	txQueue := state.NewTransactionState(telemetryMock)
	return NewSystemModule(net, nil, core, chain.Storage, txQueue, nil, nil)
}

func newCoreService(t *testing.T, srvc *state.Service) *core.Service {
	// setup service
	tt := trie.NewEmptyTrie()
	rt := wazero_runtime.NewTestInstanceWithTrie(t, runtime.WESTEND_RUNTIME_v0929, tt)
	ks := keystore.NewGlobalKeystore()
	t.Cleanup(func() {
		rt.Stop()
	})

	if srvc != nil {
		srvc.Block.StoreRuntime(srvc.Block.BestBlockHash(), rt)
	}

	// insert alice key for testing
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Acco.Insert(kr.Alice())

	if srvc == nil {
		srvc = newTestStateService(t)
	}

	ctrl := gomock.NewController(t)

	mocknet := NewMockNetwork(ctrl)
	mocknet.EXPECT().GossipMessage(
		gomock.AssignableToTypeOf(new(network.TransactionMessage))).
		AnyTimes()

	cfg := &core.Config{
		Runtime:              rt,
		Keystore:             ks,
		TransactionState:     srvc.Transaction,
		BlockState:           srvc.Block,
		StorageState:         srvc.Storage,
		Network:              mocknet,
		CodeSubstitutedState: srvc.Base,
	}

	s, err := core.NewService(cfg)
	require.NoError(t, err)

	return s
}

func TestSyncState(t *testing.T) {
	ctrl := gomock.NewController(t)

	fakeCommonHash := common.NewHash([]byte("fake"))
	fakeHeader := &types.Header{
		Number: 49,
	}

	blockapiMock := mocks.NewMockBlockAPI(ctrl)
	blockapiMock.EXPECT().BestBlockHash().Return(fakeCommonHash).Times(2)
	blockapiMock.EXPECT().GetHeader(fakeCommonHash).Return(fakeHeader, nil)

	netapiMock := mocks.NewMockNetworkAPI(ctrl)
	netapiMock.EXPECT().StartingBlock().Return(int64(10))

	syncapiCtrl := gomock.NewController(t)
	syncapiMock := NewMockSyncAPI(syncapiCtrl)
	syncapiMock.EXPECT().HighestBlock().Return(uint(90))

	sysmodule := new(SystemModule)
	sysmodule.blockAPI = blockapiMock
	sysmodule.networkAPI = netapiMock
	sysmodule.syncAPI = syncapiMock

	var res SyncStateResponse
	err := sysmodule.SyncState(nil, nil, &res)
	require.NoError(t, err)

	expectedSyncState := SyncStateResponse{
		CurrentBlock:  uint32(49),
		HighestBlock:  uint32(90),
		StartingBlock: uint32(10),
	}

	require.Equal(t, expectedSyncState, res)

	blockapiMock.EXPECT().GetHeader(fakeCommonHash).Return(nil, errors.New("Problems while getting header"))
	err = sysmodule.SyncState(nil, nil, nil)
	require.Error(t, err)
}

func TestLocalListenAddresses(t *testing.T) {
	ctrl := gomock.NewController(t)

	ma, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/7001/p2p/12D3KooWCYyh5xoAc5oRyiGU4d9ktcqFQ23JjitNFR6bEcbw7YdN")
	require.NoError(t, err)

	mockedNetState := common.NetworkState{
		PeerID:     "fake-peer-id",
		Multiaddrs: []multiaddr.Multiaddr{ma},
	}

	mockNetAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetAPI.EXPECT().NetworkState().Return(mockedNetState)

	res := make([]string, 0)

	sysmodule := new(SystemModule)
	sysmodule.networkAPI = mockNetAPI

	err = sysmodule.LocalListenAddresses(nil, nil, &res)
	require.NoError(t, err)

	require.Len(t, res, 1)
	require.Equal(t, res[0], ma.String())

	mockNetAPI.EXPECT().NetworkState().Return(common.NetworkState{Multiaddrs: []multiaddr.Multiaddr{}})
	err = sysmodule.LocalListenAddresses(nil, nil, &res)
	require.Error(t, err, "multiaddress list is empty")
}

func TestLocalPeerId(t *testing.T) {
	ctrl := gomock.NewController(t)

	peerID := "12D3KooWBrwpqLE9Z23NEs59m2UHUs9sGYWenxjeCk489Xq7SG2h"
	encoded := base58.Encode([]byte(peerID))

	state := common.NetworkState{
		PeerID: peerID,
	}

	mocknetAPI := mocks.NewMockNetworkAPI(ctrl)
	mocknetAPI.EXPECT().NetworkState().Return(state)

	sysmodules := new(SystemModule)
	sysmodules.networkAPI = mocknetAPI

	var res string
	err := sysmodules.LocalPeerId(nil, nil, &res)
	require.NoError(t, err)

	require.Equal(t, res, encoded)

	state.PeerID = ""
	mocknetAPI.EXPECT().NetworkState().Return(state)
	err = sysmodules.LocalPeerId(nil, nil, &res)
	require.Error(t, err)
}

func TestAddReservedPeer(t *testing.T) {
	t.Run("Test_Add_and_Remove_reserved_peers_with_success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		networkMock := mocks.NewMockNetworkAPI(ctrl)
		networkMock.EXPECT().AddReservedPeers(gomock.Any()).Return(nil)
		networkMock.EXPECT().RemoveReservedPeers(gomock.Any()).Return(nil)

		multiAddrPeer := "/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"
		sysModule := &SystemModule{
			networkAPI: networkMock,
		}

		var b *[]byte
		err := sysModule.AddReservedPeer(nil, &StringRequest{String: multiAddrPeer}, b)
		require.NoError(t, err)
		require.Nil(t, b)

		peerID := "QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"
		err = sysModule.RemoveReservedPeer(nil, &StringRequest{String: peerID}, b)
		require.NoError(t, err)
		require.Nil(t, b)
	})

	t.Run("Test_Add_and_Remove_reserved_peers_without_success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		networkMock := mocks.NewMockNetworkAPI(ctrl)
		networkMock.EXPECT().AddReservedPeers(gomock.Any()).Return(errors.New("some problems"))
		networkMock.EXPECT().RemoveReservedPeers(gomock.Any()).Return(errors.New("other problems"))

		sysModule := &SystemModule{
			networkAPI: networkMock,
		}

		var b *[]byte
		err := sysModule.AddReservedPeer(nil, &StringRequest{String: ""}, b)
		require.Error(t, err, "cannot add an empty reserved peer")
		require.Nil(t, b)

		multiAddrPeer := "/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"
		err = sysModule.AddReservedPeer(nil, &StringRequest{String: multiAddrPeer}, b)
		require.Error(t, err, "some problems")
		require.Nil(t, b)

		peerID := "QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"
		err = sysModule.RemoveReservedPeer(nil, &StringRequest{String: peerID}, b)
		require.Error(t, err, "other problems")
		require.Nil(t, b)
	})

	t.Run("Test_trying_to_add_or_remove_peers_with_empty_or_white_space_request", func(t *testing.T) {
		sysModule := &SystemModule{}
		require.Error(t, sysModule.AddReservedPeer(nil, &StringRequest{String: ""}, nil))
		require.Error(t, sysModule.RemoveReservedPeer(nil, &StringRequest{String: "    "}, nil))
	})
}
