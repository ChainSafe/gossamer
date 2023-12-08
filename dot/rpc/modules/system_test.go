// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	testdata "github.com/ChainSafe/gossamer/dot/rpc/modules/test_data"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemModule_ChainTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockSystemAPI := mocks.NewMockSystemAPI(ctrl)
	mockSystemAPI.EXPECT().ChainName().Return("polkadot")
	sm := &SystemModule{
		systemAPI: mockSystemAPI,
	}

	req := &EmptyRequest{}
	var res string
	err := sm.Chain(nil, req, &res)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "polkadot", res)
}

func TestSystemModule_NameTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockSystemAPI := mocks.NewMockSystemAPI(ctrl)
	mockSystemAPI.EXPECT().SystemName().Return("kusama")
	sm := &SystemModule{
		systemAPI: mockSystemAPI,
	}

	req := &EmptyRequest{}
	var res string
	err := sm.Name(nil, req, &res)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "kusama", res)
}

func TestSystemModule_ChainTypeTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockSystemAPI := mocks.NewMockSystemAPI(ctrl)
	mockSystemAPI.EXPECT().ChainType().Return("testChainType")
	sm := &SystemModule{
		systemAPI: mockSystemAPI,
	}

	req := &EmptyRequest{}
	var res string
	err := sm.ChainType(nil, req, &res)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "testChainType", res)
}

func TestSystemModule_PropertiesTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	var emptyMap map[string]interface{}
	mockSystemAPI := mocks.NewMockSystemAPI(ctrl)
	mockSystemAPI.EXPECT().Properties().Return(emptyMap)
	sm := &SystemModule{
		systemAPI: mockSystemAPI,
	}

	req := &EmptyRequest{}
	var resMap interface{}
	err := sm.Properties(nil, req, &resMap)
	require.NoError(t, err)
	require.Equal(t, emptyMap, resMap)
}

func TestSystemModule_SystemVersionTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockSystemAPI := mocks.NewMockSystemAPI(ctrl)
	mockSystemAPI.EXPECT().SystemVersion().Return("1.2.1")
	sm := &SystemModule{
		systemAPI: mockSystemAPI,
	}

	req := &EmptyRequest{}
	var res string
	err := sm.Version(nil, req, &res)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "1.2.1", res)
}

func TestSystemModule_HealthTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI.EXPECT().Health().Return(common.Health{})
	sm := &SystemModule{
		networkAPI: mockNetworkAPI,
	}

	req := &EmptyRequest{}
	var sysHealthRes SystemHealthResponse
	err := sm.Health(nil, req, &sysHealthRes)
	require.NoError(t, err)
	require.Equal(t, SystemHealthResponse(common.Health{}), sysHealthRes)
}

func TestSystemModule_NetworkStateTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI.EXPECT().NetworkState().Return(common.NetworkState{})
	sm := &SystemModule{
		networkAPI: mockNetworkAPI,
	}

	req := &EmptyRequest{}
	var networkStateRes SystemNetworkStateResponse
	err := sm.NetworkState(nil, req, &networkStateRes)
	require.NoError(t, err)
	require.Equal(t, SystemNetworkStateResponse{}, networkStateRes)
}

func TestSystemModule_PeersTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI.EXPECT().Peers().Return([]common.PeerInfo{})
	sm := &SystemModule{
		networkAPI: mockNetworkAPI,
	}

	req := &EmptyRequest{}
	var sysPeerRes SystemPeersResponse
	err := sm.Peers(nil, req, &sysPeerRes)
	require.NoError(t, err)
	require.Equal(t, SystemPeersResponse{}, sysPeerRes)
}

func TestSystemModule_NodeRolesTest(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockNetworkAPI1 := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI1.EXPECT().NodeRoles().Return(common.FullNodeRole)

	mockNetworkAPI2 := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI2.EXPECT().NodeRoles().Return(common.LightClientRole)

	mockNetworkAPI3 := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI3.EXPECT().NodeRoles().Return(common.AuthorityRole)

	mockNetworkAPI4 := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI4.EXPECT().NodeRoles().Return(common.NetworkRole(21))

	type args struct {
		r   *http.Request
		req *EmptyRequest
	}
	tests := []struct {
		name      string
		sysModule *SystemModule
		args      args
		expErr    error
		exp       []interface{}
	}{
		{
			name:      "Full",
			sysModule: NewSystemModule(mockNetworkAPI1, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &EmptyRequest{},
			},
			exp: []interface{}{"Full"},
		},
		{
			name:      "LightClient",
			sysModule: NewSystemModule(mockNetworkAPI2, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &EmptyRequest{},
			},
			exp: []interface{}{"LightClient"},
		},
		{
			name:      "Authority",
			sysModule: NewSystemModule(mockNetworkAPI3, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &EmptyRequest{},
			},
			exp: []interface{}{"Authority"},
		},
		{
			name:      "UnknownRole",
			sysModule: NewSystemModule(mockNetworkAPI4, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &EmptyRequest{},
			},
			exp: []interface{}{"UnknownRole", []interface{}{common.NetworkRole(21)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.sysModule
			res := []interface{}{}
			err := sm.NodeRoles(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestSystemModule_AccountNextIndex(t *testing.T) {
	ctrl := gomock.NewController(t)

	storageKeyHex := common.MustHexToBytes("0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886" +
		"371da93116aec311d8421cece41129ffaac05aa7f9580382edb384b1b43cbcf3d1b1e7f1a1d232cf4139bd48eaafb9656da27d")
	signedExt := common.MustHexToBytes("0xad018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e" +
		"7a56da27d0146d0050619728683af4e9659bf202aeb2b8b13b48a875adb663f449f1a71453903546f3252193964185eb91" +
		"c482cf95caf327db407d57ebda95046b5ef890187001000000108abcd")
	v := make([]*transaction.ValidTransaction, 1)
	v[0] = &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(signedExt),
		Validity:  new(transaction.Validity),
	}

	mockTxStateAPI := mocks.NewMockTransactionStateAPI(ctrl)
	mockTxStateAPI.EXPECT().Pending().Return(v).Times(5)

	mockCoreAPI := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPI.EXPECT().GetMetadata((*common.Hash)(nil)).
		Return(common.MustHexToBytes(testdata.NewTestMetadata()), nil).Times(2)

	mockCoreAPIErr := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIErr.EXPECT().GetMetadata((*common.Hash)(nil)).
		Return(nil, errors.New("getMetadata error"))

	// Magic number mismatch
	mockCoreAPIMagicNumMismatch := mocks.NewMockCoreAPI(ctrl)
	mockCoreAPIMagicNumMismatch.EXPECT().GetMetadata((*common.Hash)(nil)).Return(storageKeyHex, nil)

	mockStorageAPI := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPI.EXPECT().GetStorage((*common.Hash)(nil), storageKeyHex).
		Return(common.MustHexToBytes("0x0300000000000000000000000000000000000000000000000000000000000000000000"+
			"000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), nil)

	mockStorageAPIErr := mocks.NewMockStorageAPI(ctrl)
	mockStorageAPIErr.EXPECT().GetStorage((*common.Hash)(nil), storageKeyHex).Return(nil, errors.New("getStorage error"))

	type args struct {
		r   *http.Request
		req *StringRequest
	}
	tests := []struct {
		name      string
		sysModule *SystemModule
		args      args
		expErr    error
		exp       U64Response
	}{
		{
			name:      "Nil Request",
			sysModule: NewSystemModule(nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil, nil),
			args:      args{},
			expErr:    errors.New("account address must be valid"),
		},
		{
			name:      "found_in_pending_transactions",
			sysModule: NewSystemModule(nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil, nil),
			args: args{
				req: &StringRequest{String: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			exp: U64Response(4),
		},
		{
			name:      "not_found_in_pending_transactions",
			sysModule: NewSystemModule(nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil, nil),
			args: args{
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			exp: U64Response(3),
		},
		{
			name:      "GetMetadata Err",
			sysModule: NewSystemModule(nil, nil, mockCoreAPIErr, mockStorageAPI, mockTxStateAPI, nil, nil),
			args: args{
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			expErr: errors.New("getMetadata error"),
		},
		{
			name:      "Magic Number Mismatch",
			sysModule: NewSystemModule(nil, nil, mockCoreAPIMagicNumMismatch, mockStorageAPI, mockTxStateAPI, nil, nil),
			args: args{
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			expErr: errors.New("magic number mismatch: expected 0x6174656d, found 0xe03056ea"),
		},
		{
			name:      "GetStorage Err",
			sysModule: NewSystemModule(nil, nil, mockCoreAPI, mockStorageAPIErr, mockTxStateAPI, nil, nil),
			args: args{
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			expErr: errors.New("getStorage error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := U64Response(0)
			sm := tt.sysModule
			err := sm.AccountNextIndex(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestSystemModule_SyncState(t *testing.T) {
	ctrl := gomock.NewController(t)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI := mocks.NewMockBlockAPI(ctrl)
	mockBlockAPI.EXPECT().BestBlockHash().Return(hash)
	mockBlockAPI.EXPECT().GetHeader(hash).Return(types.NewEmptyHeader(), nil)

	mockBlockAPIErr := mocks.NewMockBlockAPI(ctrl)
	mockBlockAPIErr.EXPECT().BestBlockHash().Return(hash)
	mockBlockAPIErr.EXPECT().GetHeader(hash).Return(nil, errors.New("GetHeader Err"))

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI.EXPECT().StartingBlock().Return(int64(23))

	ctrlSyncAPI := gomock.NewController(t)
	mockSyncAPI := NewMockSyncAPI(ctrlSyncAPI)
	mockSyncAPI.EXPECT().HighestBlock().Return(uint(21))

	type args struct {
		r   *http.Request
		req *EmptyRequest
	}
	tests := []struct {
		name      string
		sysModule *SystemModule
		args      args
		expErr    error
		exp       SyncStateResponse
	}{
		{
			name:      "OK",
			sysModule: NewSystemModule(mockNetworkAPI, nil, nil, nil, nil, mockBlockAPI, mockSyncAPI),
			args: args{
				req: &EmptyRequest{},
			},
			exp: SyncStateResponse{
				CurrentBlock:  0x0,
				HighestBlock:  0x15,
				StartingBlock: 0x17,
			},
		},
		{
			name:      "Err",
			sysModule: NewSystemModule(mockNetworkAPI, nil, nil, nil, nil, mockBlockAPIErr, nil),
			args: args{
				req: &EmptyRequest{},
			},
			expErr: errors.New("GetHeader Err"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.sysModule
			res := SyncStateResponse{}
			err := sm.SyncState(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestSystemModule_LocalListenAddresses(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockNetworkAPIEmpty := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPIEmpty.EXPECT().NetworkState().Return(common.NetworkState{})

	addr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	require.NoError(t, err)
	multiAddy := make([]multiaddr.Multiaddr, 1)
	multiAddy[0] = addr
	ns := common.NetworkState{
		PeerID:     "jimbo",
		Multiaddrs: multiAddy,
	}

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI.EXPECT().NetworkState().Return(ns)

	type args struct {
		r   *http.Request
		req *EmptyRequest
	}
	tests := []struct {
		name      string
		sysModule *SystemModule
		args      args
		expErr    error
		exp       []string
	}{
		{
			name:      "OK",
			sysModule: NewSystemModule(mockNetworkAPI, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &EmptyRequest{},
			},
			exp: []string{"/ip4/1.2.3.4/tcp/80"},
		},
		{
			name:      "Empty multiaddress list",
			sysModule: NewSystemModule(mockNetworkAPIEmpty, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &EmptyRequest{},
			},
			expErr: errors.New("multiaddress list is empty"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.sysModule
			var res []string
			err := sm.LocalListenAddresses(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestSystemModule_LocalPeerId(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockNetworkAPIEmpty := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPIEmpty.EXPECT().NetworkState().Return(common.NetworkState{})

	addr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	require.NoError(t, err)
	multiAddy := make([]multiaddr.Multiaddr, 1)
	multiAddy[0] = addr
	ns := common.NetworkState{
		PeerID:     "jimbo",
		Multiaddrs: multiAddy,
	}

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI.EXPECT().NetworkState().Return(ns)

	type args struct {
		r   *http.Request
		req *EmptyRequest
	}
	tests := []struct {
		name      string
		sysModule *SystemModule
		args      args
		expErr    error
		exp       string
	}{
		{
			name:      "OK",
			sysModule: NewSystemModule(mockNetworkAPI, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &EmptyRequest{},
			},
			exp: "D1KeRhQ",
		},
		{
			name:      "Empty peerId",
			sysModule: NewSystemModule(mockNetworkAPIEmpty, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &EmptyRequest{},
			},
			expErr: errors.New("peer id cannot be empty"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.sysModule
			res := ""
			err := sm.LocalPeerId(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestSystemModule_AddReservedPeer(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI.EXPECT().AddReservedPeers("jimbo").Return(nil)

	mockNetworkAPIErr := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPIErr.EXPECT().AddReservedPeers("jimbo").Return(errors.New("addReservedPeer error"))

	type args struct {
		r   *http.Request
		req *StringRequest
	}
	tests := []struct {
		name      string
		sysModule *SystemModule
		args      args
		expErr    error
		exp       []byte
	}{
		{
			name:      "OK",
			sysModule: NewSystemModule(mockNetworkAPI, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &StringRequest{"jimbo"},
			},
			exp: []byte(nil),
		},
		{
			name:      "AddReservedPeer Error",
			sysModule: NewSystemModule(mockNetworkAPIErr, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &StringRequest{"jimbo"},
			},
			expErr: errors.New("addReservedPeer error"),
		},
		{
			name:      "Empty StringRequest Error",
			sysModule: NewSystemModule(mockNetworkAPI, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &StringRequest{""},
			},
			expErr: errors.New("cannot add an empty reserved peer"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.sysModule
			res := []byte(nil)
			err := sm.AddReservedPeer(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestSystemModule_RemoveReservedPeer(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPI.EXPECT().RemoveReservedPeers("jimbo").Return(nil)

	mockNetworkAPIErr := mocks.NewMockNetworkAPI(ctrl)
	mockNetworkAPIErr.EXPECT().RemoveReservedPeers("jimbo").Return(errors.New("removeReservedPeer error"))

	type args struct {
		r   *http.Request
		req *StringRequest
	}
	tests := []struct {
		name      string
		sysModule *SystemModule
		args      args
		expErr    error
		exp       []byte
	}{
		{
			name:      "OK",
			sysModule: NewSystemModule(mockNetworkAPI, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &StringRequest{"jimbo"},
			},
			exp: []byte(nil),
		},
		{
			name:      "RemoveReservedPeer Error",
			sysModule: NewSystemModule(mockNetworkAPIErr, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &StringRequest{"jimbo"},
			},
			expErr: errors.New("removeReservedPeer error"),
		},
		{
			name:      "Empty StringRequest Error",
			sysModule: NewSystemModule(mockNetworkAPI, nil, nil, nil, nil, nil, nil),
			args: args{
				req: &StringRequest{""},
			},
			expErr: errors.New("cannot remove an empty reserved peer"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.sysModule
			res := []byte(nil)
			err := sm.RemoveReservedPeer(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
