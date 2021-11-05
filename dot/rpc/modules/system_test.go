// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/mock"

	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	testdata "github.com/ChainSafe/gossamer/dot/rpc/modules/test_data"
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

	var res []interface{}
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
	}{
		{
			name: "Full",
			fields: fields{mockNetworkAPI1, nil, nil, nil, nil, nil},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "LightClient",
			fields: fields{mockNetworkAPI2, nil, nil, nil, nil, nil},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Authority",
			fields: fields{mockNetworkAPI3, nil, nil, nil, nil, nil},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "UnknownRole",
			fields: fields{mockNetworkAPI4, nil, nil, nil, nil, nil},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := sm.NodeRoles(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("NodeRoles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSystemModule_AccountNextIndex(t *testing.T) {
	signedExt := common.MustHexToBytes("0xad018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0146d0050619728683af4e9659bf202aeb2b8b13b48a875adb663f449f1a71453903546f3252193964185eb91c482cf95caf327db407d57ebda95046b5ef890187001000000108abcd")
	v := make([]*transaction.ValidTransaction, 1)
	v[0] = &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(signedExt),
		Validity:  new(transaction.Validity),
	}

	mockTxStateAPI := new(apimocks.TransactionStateAPI)
	mockTxStateAPI.On("Pending").Return(v, nil)

	mockCoreAPI := new(apimocks.CoreAPI)
	mockCoreAPI.On("GetMetadata", mock.AnythingOfType("*common.Hash")).Return(common.MustHexToBytes(testdata.TestData), nil)

	mockCoreAPIErr := new(apimocks.CoreAPI)
	mockCoreAPIErr.On("GetMetadata", mock.AnythingOfType("*common.Hash")).Return(nil, errors.New("getMetadata error"))

	// Magic number mismatch
	mockCoreAPIMagicNumMismatch := new(apimocks.CoreAPI)
	mockCoreAPIMagicNumMismatch.On("GetMetadata", mock.AnythingOfType("*common.Hash")).Return(common.MustHexToBytes("0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"), nil)

	mockStorageAPI := new(apimocks.StorageAPI)
	mockStorageAPI.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(common.MustHexToBytes("0x03000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"), nil)

	mockStorageAPIErr := new(apimocks.StorageAPI)
	mockStorageAPIErr.On("GetStorage", mock.AnythingOfType("*common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil, errors.New("getStorage error"))

	var res U64Response
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
	}{
		{
			name: "Nil Request",
			fields: fields{nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				r: nil,
				req: nil,
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Found",
			fields: fields{nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				r: nil,
				req: &StringRequest{String: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Not found",
			fields: fields{nil, nil, mockCoreAPI, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				r: nil,
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "GetMetadata Err",
			fields: fields{nil, nil, mockCoreAPIErr, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				r: nil,
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Magic Number Mismatch",
			fields: fields{nil, nil, mockCoreAPIMagicNumMismatch, mockStorageAPI, mockTxStateAPI, nil},
			args: args{
				r: nil,
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetStorage Err",
			fields: fields{nil, nil, mockCoreAPI, mockStorageAPIErr, mockTxStateAPI, nil},
			args: args{
				r: nil,
				req: &StringRequest{String: "5FrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := sm.AccountNextIndex(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("AccountNextIndex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSystemModule_SyncState(t *testing.T) {
	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockBlockAPI := new(apimocks.BlockAPI)
	mockBlockAPI.On("BestBlockHash").Return(hash)
	mockBlockAPI.On("GetHeader", mock.AnythingOfType("common.Hash")).Return(types.NewEmptyHeader(), nil)

	mockBlockAPIErr := new(apimocks.BlockAPI)
	mockBlockAPIErr.On("BestBlockHash").Return(hash)
	mockBlockAPIErr.On("GetHeader", mock.AnythingOfType("common.Hash")).Return(nil, errors.New("GetHeader Err"))

	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockNetworkAPI.On("HighestBlock").Return(int64(21))
	mockNetworkAPI.On("StartingBlock").Return(int64(23))

	var res SyncStateResponse
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
	}{
		{
			name: "OK",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, mockBlockAPI},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Err",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, mockBlockAPIErr},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := sm.SyncState(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("SyncState() error = %v, wantErr %v", err, tt.wantErr)
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

	var res []string
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
	}{
		{
			name: "OK",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, nil},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Empty multiaddress list",
			fields: fields{mockNetworkAPIEmpty, nil, nil, nil, nil, nil},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := sm.LocalListenAddresses(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("LocalListenAddresses() error = %v, wantErr %v", err, tt.wantErr)
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

	var res string
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
	}{
		{
			name: "OK",
			fields: fields{mockNetworkAPI, nil, nil, nil, nil, nil},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Empty peerId",
			fields: fields{mockNetworkAPIEmpty, nil, nil, nil, nil, nil},
			args: args{
				r: nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SystemModule{
				networkAPI: tt.fields.networkAPI,
				systemAPI:  tt.fields.systemAPI,
				coreAPI:    tt.fields.coreAPI,
				storageAPI: tt.fields.storageAPI,
				txStateAPI: tt.fields.txStateAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := sm.LocalPeerId(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("LocalPeerId() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}