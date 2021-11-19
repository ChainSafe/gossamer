// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	testdata "github.com/ChainSafe/gossamer/dot/rpc/modules/test_data"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestSystemModule_BasicGetters(t *testing.T) {
	var emptyMap map[string]interface{}
	mockSystemAPI := new(apimocks.SystemAPI)
	mockSystemAPI.On("ChainName").Return("polkadot", nil)
	mockSystemAPI.On("SystemName").Return("kusama", nil)
	mockSystemAPI.On("ChainType").Return("testChainType", nil)
	mockSystemAPI.On("Properties").Return(emptyMap)
	mockSystemAPI.On("SystemVersion").Return("1.2.1", nil)

	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockNetworkAPI.On("Health").Return(common.Health{}, nil)
	mockNetworkAPI.On("NetworkState").Return(common.NetworkState{}, nil)
	mockNetworkAPI.On("Peers").Return([]common.PeerInfo{}, nil)

	sm := NewSystemModule(mockNetworkAPI, mockSystemAPI, new(apimocks.CoreAPI), new(apimocks.StorageAPI), new(apimocks.TransactionStateAPI), new(apimocks.BlockAPI))

	req := &EmptyRequest{}
	var res string
	var resMap interface{}
	var sysHealthRes SystemHealthResponse
	var networkStateRes SystemNetworkStateResponse
	var sysPeerRes SystemPeersResponse

	err := sm.Chain(nil, req, &res)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "polkadot", res)

	err = sm.Name(nil, req, &res)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "kusama", res)

	err = sm.ChainType(nil, req, &res)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "testChainType", res)

	err = sm.Properties(nil, req, &resMap)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, emptyMap, resMap)

	err = sm.Version(nil, req, &res)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "1.2.1", res)

	err = sm.Health(nil, req, &sysHealthRes)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, SystemHealthResponse(common.Health{}), sysHealthRes)

	err = sm.NetworkState(nil, req, &networkStateRes)
	require.NoError(t, err)
	require.NotNil(t, res)

	err = sm.Peers(nil, req, &sysPeerRes)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, SystemPeersResponse{}, sysPeerRes)
}

func TestSystemModule_TestNodeRoles(t *testing.T) {
	mockNetworkAPI1 := new(apimocks.NetworkAPI)
	mockNetworkAPI1.On("NodeRoles").Return(byte(1), nil)

	mockNetworkAPI2 := new(apimocks.NetworkAPI)
	mockNetworkAPI2.On("NodeRoles").Return(byte(2), nil)

	mockNetworkAPI3 := new(apimocks.NetworkAPI)
	mockNetworkAPI3.On("NodeRoles").Return(byte(4), nil)

	mockNetworkAPI4 := new(apimocks.NetworkAPI)
	mockNetworkAPI4.On("NodeRoles").Return(byte(21), nil)

	type fields struct {
		networkAPI NetworkAPI
		systemAPI  SystemAPI
		coreAPI    CoreAPI
		storageAPI StorageAPI
		txStateAPI TransactionStateAPI
		blockAPI   BlockAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res *[]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     []interface{}
	}{
		{
			name:   "Full",
			fields: fields{mockNetworkAPI1, nil, nil, nil, nil, nil},
			args: args{
				req: &EmptyRequest{},
			},
			exp: []interface{}{"Full"},
		},
		{
			name:   "LightClient",
			fields: fields{mockNetworkAPI2, nil, nil, nil, nil, nil},
			args: args{
				req: &EmptyRequest{},
			},
			exp: []interface{}{"LightClient"},
		},
		{
			name:   "Authority",
			fields: fields{mockNetworkAPI3, nil, nil, nil, nil, nil},
			args: args{
				req: &EmptyRequest{},
			},
			exp: []interface{}{"Authority"},
		},
		{
			name:   "UnknownRole",
			fields: fields{mockNetworkAPI4, nil, nil, nil, nil, nil},
			args: args{
				req: &EmptyRequest{},
			},
			exp: []interface{}{"UnknownRole", []interface{}{uint8(21)}},
		},
	}
	for _, tt := range tests {
		var res []interface{}
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = sm.NodeRoles(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("NodeRoles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestSystemModule_AccountNextIndex(t *testing.T) {
	storageKeyHex := common.MustHexToBytes("0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da93116aec311d8421cece41129ffaac05aa7f9580382edb384b1b43cbcf3d1b1e7f1a1d232cf4139bd48eaafb9656da27d")
	signedExt := common.MustHexToBytes("0xad018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0146d0050619728683af4e9659bf202aeb2b8b13b48a875adb663f449f1a71453903546f3252193964185eb91c482cf95caf327db407d57ebda95046b5ef890187001000000108abcd")
	v := make([]*transaction.ValidTransaction, 1)
	v[0] = &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(signedExt),
		Validity:  new(transaction.Validity),
	}

	mockTxStateAPI := new(apimocks.TransactionStateAPI)
	mockTxStateAPI.On("Pending").Return(v, nil)

	mockCoreAPI := new(apimocks.CoreAPI)
	mockCoreAPI.On("GetMetadata", (*common.Hash)(nil)).Return(common.MustHexToBytes(testdata.GetTestData()), nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("GetMetadata", (*common.Hash)(nil)).Return(nil, errors.New("getMetadata error"))

	// Magic number mismatch
	mockCoreAPIMagicNumMismatch := new(apimocks.CoreAPI)
	mockCoreAPIMagicNumMismatch.On("GetMetadata", (*common.Hash)(nil)).Return(storageKeyHex, nil)

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStorage", (*common.Hash)(nil), storageKeyHex).Return(common.MustHexToBytes("0x03000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetStorage", (*common.Hash)(nil), storageKeyHex).Return(nil, errors.New("getStorage error"))

	type fields struct {
		networkAPI NetworkAPI
		systemAPI  SystemAPI
		coreAPI    CoreAPI
		storageAPI StorageAPI
		txStateAPI TransactionStateAPI
		blockAPI   BlockAPI
	}
	type args struct {
		r   *http.Request
		req *StringRequest
		res *U64Response
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     U64Response
	}{
		{
			name:    "Nil Request",
			fields:  fields{nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil},
			args:    args{},
			wantErr: true,
			err:     errors.New("account address must be valid"),
		},
		{
			name:   "Found",
			fields: fields{nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				req: &StringRequest{String: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			exp: U64Response(4),
		},
		{
			name:   "Not found",
			fields: fields{nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			exp: U64Response(3),
		},
		{
			name:   "GetMetadata Err",
			fields: fields{nil, nil, mockCoreAPIErr, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			wantErr: true,
			err:     errors.New("getMetadata error"),
		},
		{
			name:   "Magic Number Mismatch",
			fields: fields{nil, nil, mockCoreAPIMagicNumMismatch, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			wantErr: true,
			err:     errors.New("magic number mismatch: expected 0x6174656d, found 0xe03056ea"),
		},
		{
			name:   "GetStorage Err",
			fields: fields{nil, nil, mockCoreAPI, mockStorageAPIErr, mockTxStateAPI, nil},
			args: args{
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
			},
			wantErr: true,
			err:     errors.New("getStorage error"),
		},
	}
	for _, tt := range tests {
		var res U64Response
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = sm.AccountNextIndex(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("AccountNextIndex() error = %v, wantErr %v", err, tt.wantErr)
				//return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestSystemModule_SyncState(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("BestBlockHash").Return(hash)
	mockBlockAPI.On("GetHeader", hash).Return(types.NewEmptyHeader(), nil)

	mockBlockAPIErr := new(apimocks.BlockAPI)
	mockBlockAPIErr.On("BestBlockHash").Return(hash)
	mockBlockAPIErr.On("GetHeader", hash).Return(nil, errors.New("GetHeader Err"))

	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockNetworkAPI.On("HighestBlock").Return(int64(21))
	mockNetworkAPI.On("StartingBlock").Return(int64(23))

	type fields struct {
		networkAPI NetworkAPI
		systemAPI  SystemAPI
		coreAPI    CoreAPI
		storageAPI StorageAPI
		txStateAPI TransactionStateAPI
		blockAPI   BlockAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res *SyncStateResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     SyncStateResponse
	}{
		{
			name:   "OK",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, mockBlockAPI},
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
			name:   "Err",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, mockBlockAPIErr},
			args: args{
				req: &EmptyRequest{},
			},
			wantErr: true,
			err:     errors.New("GetHeader Err"),
		},
	}
	for _, tt := range tests {
		var res SyncStateResponse
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = sm.SyncState(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("SyncState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestSystemModule_LocalListenAddresses(t *testing.T) {
	mockNetworkAPIEmpty := new(apimocks.NetworkAPI)
	mockNetworkAPIEmpty.On("NetworkState").Return(common.NetworkState{})

	addr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	require.NoError(t, err)
	multiAddy := make([]multiaddr.Multiaddr, 1)
	multiAddy[0] = addr
	ns := common.NetworkState{
		PeerID:     "jimbo",
		Multiaddrs: multiAddy,
	}

	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockNetworkAPI.On("NetworkState").Return(ns, nil)

	type fields struct {
		networkAPI NetworkAPI
		systemAPI  SystemAPI
		coreAPI    CoreAPI
		storageAPI StorageAPI
		txStateAPI TransactionStateAPI
		blockAPI   BlockAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res *[]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     []string
	}{
		{
			name:   "OK",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, nil},
			args: args{
				req: &EmptyRequest{},
			},
			exp: []string{"/ip4/1.2.3.4/tcp/80"},
		},
		{
			name:   "Empty multiaddress list",
			fields: fields{mockNetworkAPIEmpty, nil, nil, nil, nil, nil},
			args: args{
				req: &EmptyRequest{},
			},
			wantErr: true,
			err:     errors.New("multiaddress list is empty"),
		},
	}
	for _, tt := range tests {
		var res []string
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = sm.LocalListenAddresses(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("LocalListenAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestSystemModule_LocalPeerId(t *testing.T) {
	mockNetworkAPIEmpty := new(apimocks.NetworkAPI)
	mockNetworkAPIEmpty.On("NetworkState").Return(common.NetworkState{})

	addr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	require.NoError(t, err)
	multiAddy := make([]multiaddr.Multiaddr, 1)
	multiAddy[0] = addr
	ns := common.NetworkState{
		PeerID:     "jimbo",
		Multiaddrs: multiAddy,
	}

	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockNetworkAPI.On("NetworkState").Return(ns, nil)

	type fields struct {
		networkAPI NetworkAPI
		systemAPI  SystemAPI
		coreAPI    CoreAPI
		storageAPI StorageAPI
		txStateAPI TransactionStateAPI
		blockAPI   BlockAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     string
	}{
		{
			name:   "OK",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, nil},
			args: args{
				req: &EmptyRequest{},
			},
			exp: "D1KeRhQ",
		},
		{
			name:   "Empty peerId",
			fields: fields{mockNetworkAPIEmpty, nil, nil, nil, nil, nil},
			args: args{
				req: &EmptyRequest{},
			},
			wantErr: true,
			err:     errors.New("peer id cannot be empty"),
		},
	}
	for _, tt := range tests {
		var res string
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = sm.LocalPeerId(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("LocalPeerId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestSystemModule_AddReservedPeer(t *testing.T) {
	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockNetworkAPI.On("AddReservedPeers", "jimbo").Return(nil)

	mockNetworkAPIErr := new(apimocks.NetworkAPI)
	mockNetworkAPIErr.On("AddReservedPeers", "jimbo").Return(errors.New("addReservedPeer error"))

	type fields struct {
		networkAPI NetworkAPI
		systemAPI  SystemAPI
		coreAPI    CoreAPI
		storageAPI StorageAPI
		txStateAPI TransactionStateAPI
		blockAPI   BlockAPI
	}
	type args struct {
		r   *http.Request
		req *StringRequest
		res *[]byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     []byte
	}{
		{
			name:   "OK",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, nil},
			args: args{
				req: &StringRequest{"jimbo"},
			},
			exp: []byte(nil),
		},
		{
			name:   "AddReservedPeer Error",
			fields: fields{mockNetworkAPIErr, nil, nil, nil, nil, nil},
			args: args{
				req: &StringRequest{"jimbo"},
			},
			wantErr: true,
			err:     errors.New("addReservedPeer error"),
		},
		{
			name:   "Empty StringRequest Error",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, nil},
			args: args{
				req: &StringRequest{""},
			},
			wantErr: true,
			err:     errors.New("cannot add an empty reserved peer"),
		},
	}
	for _, tt := range tests {
		var res []byte
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = sm.AddReservedPeer(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("AddReservedPeer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestSystemModule_RemoveReservedPeer(t *testing.T) {
	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockNetworkAPI.On("RemoveReservedPeers", "jimbo").Return(nil)

	mockNetworkAPIErr := new(apimocks.NetworkAPI)
	mockNetworkAPIErr.On("RemoveReservedPeers", "jimbo").Return(errors.New("removeReservedPeer error"))

	type fields struct {
		networkAPI NetworkAPI
		systemAPI  SystemAPI
		coreAPI    CoreAPI
		storageAPI StorageAPI
		txStateAPI TransactionStateAPI
		blockAPI   BlockAPI
	}
	type args struct {
		r   *http.Request
		req *StringRequest
		res *[]byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     []byte
	}{
		{
			name:   "OK",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, nil},
			args: args{
				req: &StringRequest{"jimbo"},
			},
			exp: []byte(nil),
		},
		{
			name:   "RemoveReservedPeer Error",
			fields: fields{mockNetworkAPIErr, nil, nil, nil, nil, nil},
			args: args{
				req: &StringRequest{"jimbo"},
			},
			wantErr: true,
			err:     errors.New("removeReservedPeer error"),
		},
		{
			name:   "Empty StringRequest Error",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, nil},
			args: args{
				req: &StringRequest{""},
			},
			wantErr: true,
			err:     errors.New("cannot remove an empty reserved peer"),
		},
	}
	for _, tt := range tests {
		var res []byte
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = sm.RemoveReservedPeer(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("RemoveReservedPeer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}
