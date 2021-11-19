// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

func createTestTrieState(t *testing.T) (*trie.Trie, common.Hash) {
	t.Helper()

	_, genTrie, _ := genesis.NewTestGenesisWithTrieAndHeader(t)
	tr, err := rtstorage.NewTrieState(genTrie)
	require.NoError(t, err)

	tr.Set([]byte(":first_key"), []byte(":value1"))
	tr.Set([]byte(":second_key"), []byte(":second_value"))

	childTr := trie.NewEmptyTrie()
	childTr.Put([]byte(":child_first"), []byte(":child_first_value"))
	childTr.Put([]byte(":child_second"), []byte(":child_second_value"))
	childTr.Put([]byte(":another_child"), []byte("value"))

	err = tr.SetChild([]byte(":child_storage_key"), childTr)
	require.NoError(t, err)

	stateRoot, err := tr.Root()
	require.NoError(t, err)

	return tr.Trie(), stateRoot
}

func TestChildStateModule_GetKeys(t *testing.T) {
	expStr := []string{"0x11f3ba2e1cdd6d62f2ff9b5589e7ff81ba7fb8745735dc3be2a2c61a72c39e78", "0x1cb6f36e027abb2091cfb5110ab5087f5e0621c4869aa60c02be9adcc98a0d1d", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da904f6d6860c6ef3990f6d34dd83a345aabe1d9d59de1283380100550a7b024501cb62d6cc40e3db35fcc5cf341814986e", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da923a05cabf6d3bde7ca3ef0d11596b5611cbd2d43530a44705ad088af313e18f80b53ef16b36177cd4b77b846f2a5f07c", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da94f9aea1afa791265fae359272badc1cf8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da97a26f8a0cd62e2d6addec8cbfdd102af1206960f920a23f7f4c43cc9081ec2ed0721f31a9bef2c10fd7602e16e08a32c", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da981b4d873b42c98d6628cf8d2b2373afe4603307f855321776922daeea21ee31720388d097cdaac66f05a6f8462b31757", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9b0edae20838083f2cde1c4080db8cf8090b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9e5e802737cce3a54b0bc9e3d3e6be26e306721211d5404bd9da88e0204360a1a9ab8b87c66c1bc2fcdd37f3c2222cc20", "0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9edeaa42c2163f68084a988529a0e2ec5e659a7a1628cdd93febc04a4e0646ea20e9f5f0ce097d9a05290d4a9e054df4e", "0x26aa394eea5630e07c48ae0c9558cef7c21aab032aaa6e946ca50ad39ab66603", "0x3a6368696c645f73746f726167653a64656661756c743a3a6368696c645f73746f726167655f6b6579", "0x3a636f6465", "0x3a66697273745f6b6579", "0x3a6772616e6470615f617574686f726974696573", "0x3a7365636f6e645f6b6579", "0x426e15054d267946093858132eb537f1a47a9ff5cd5bf4d848a80a0b1a947dc3", "0x426e15054d267946093858132eb537f1ba7fb8745735dc3be2a2c61a72c39e78", "0x426e15054d267946093858132eb537f1d0b4a3f7631f0c0e761898fe198211de", "0x4342193e496fab7ec59d615ed0dc5530d2d505c0e6f76fd7ce0796ebe187401c", "0x492a52699edf49c972c21db794cfcf57ba7fb8745735dc3be2a2c61a72c39e78", "0x5f3e4907f716ac89b6347d15ececedca138e71612491192d68deab7e6f563fe1", "0x5f3e4907f716ac89b6347d15ececedca28dccb559b95c40168a1b2696581b5a7", "0x5f3e4907f716ac89b6347d15ececedca5579297f4dfb9609e7e4c2ebab9ce40a", "0x5f3e4907f716ac89b6347d15ececedcaac0a2cbf8e355f5ea6cb2de8727bfb0c", "0x5f3e4907f716ac89b6347d15ececedcab49a2738eeb30896aacb8b3fb46471bd", "0x5f3e4907f716ac89b6347d15ececedcac29a0310e1bb45d20cace77ccb62c97d", "0x5f3e4907f716ac89b6347d15ececedcaf7dad0317324aecae8744b87fc95f2f3", "0x8985776095addd4789fccbce8ca77b23ba7fb8745735dc3be2a2c61a72c39e78", "0xcec5070d609dd3497f72bde07fc96ba04c014e6bf8b8c2c011e7290b85696bb3e535263148daaf49be5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f", "0xcec5070d609dd3497f72bde07fc96ba0726380404683fc89e8233450c8aa195066b8d48da86b869b6261626580d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d", "0xcec5070d609dd3497f72bde07fc96ba0726380404683fc89e8233450c8aa1950c9b0c13125732d276175646980d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d", "0xcec5070d609dd3497f72bde07fc96ba0726380404683fc89e8233450c8aa1950ed43a85541921049696d6f6e80d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d", "0xcec5070d609dd3497f72bde07fc96ba0726380404683fc89e8233450c8aa1950f5537bdb2a1f626b6772616e8088dc3417d5058ec4b4503e0c12ea1a0a89be200fe98922423d4334014fa6b0ee", "0xe2e62dd81c48a88f73b6f6463555fd8eba7fb8745735dc3be2a2c61a72c39e78"}
	tr, sr := createTestTrieState(t)

	mockStorageAPI := new(apimocks.StorageAPI)
	mockErrorStorageAPI1 := new(apimocks.StorageAPI)
	mockErrorStorageAPI2 := new(apimocks.StorageAPI)
	mockBlockAPI := new(apimocks.BlockAPI)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI.On("GetBlockHash").Return(hash)
	mockBlockAPI.On("BestBlockHash").Return(hash)

	mockStorageAPI.On("GetStateRootFromBlock", &hash).Return(&sr, nil)
	mockStorageAPI.On("GetStorageChild", &sr, []byte(":child_storage_key")).Return(tr, nil)

	mockErrorStorageAPI1.On("GetStateRootFromBlock", &common.Hash{}).Return(nil, nil)
	mockErrorStorageAPI1.On("GetStorageChild", (*common.Hash)(nil), []byte(nil)).Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	type fields struct {
		storageAPI StorageAPI
		blockAPI   BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *GetKeysRequest
		res *[]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     *[]string
	}{
		{
			name: "Get Keys Nil Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Key: []byte(":child_storage_key"),
				},
			},
			exp: &expStr,
		},
		{
			name: "Get Keys with Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Key:  []byte(":child_storage_key"),
					Hash: &hash,
				},
			},
			exp: &expStr,
		},
		{
			name: "GetStorageChild error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Hash: &common.Hash{},
				},
			},
			wantErr: true,
			err:     errors.New("GetStorageChild error"),
		},
		{
			name: "GetStateRootFromBlock error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Key: []byte(":child_storage_key"),
				},
			},
			wantErr: true,
			err:     errors.New("GetStateRootFromBlock error"),
		},
	}
	for _, tt := range tests {
		res := make([]string, 0)
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = cs.GetKeys(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, tt.args.res)
			}
		})
	}
}

func TestChildStateModule_GetStorageSize(t *testing.T) {
	_, sr := createTestTrieState(t)

	mockStorageAPI := new(apimocks.StorageAPI)
	mockErrorStorageAPI1 := new(apimocks.StorageAPI)
	mockErrorStorageAPI2 := new(apimocks.StorageAPI)
	mockBlockAPI := new(apimocks.BlockAPI)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI.On("GetBlockHash").Return(hash)
	mockBlockAPI.On("BestBlockHash").Return(hash)

	mockStorageAPI.On("GetStateRootFromBlock", &hash).Return(&sr, nil)
	mockStorageAPI.On("GetStorageFromChild", &sr, []byte(":child_storage_key"), []byte(":child_first")).Return([]byte(""), nil)

	mockErrorStorageAPI1.On("GetStateRootFromBlock", &hash).Return(nil, nil)
	mockErrorStorageAPI1.On("GetStorageFromChild", (*common.Hash)(nil), []byte(nil), []byte(nil)).Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	type fields struct {
		storageAPI StorageAPI
		blockAPI   BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *GetChildStorageRequest
		res *uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     uint64
	}{
		{
			name: "Get Keys Nil Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					KeyChild: []byte(":child_storage_key"),
					EntryKey: []byte(":child_first"),
				},
			},
			exp: uint64(0),
		},
		{
			name: "Get Keys with Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					KeyChild: []byte(":child_storage_key"),
					EntryKey: []byte(":child_first"),
					Hash:     &hash,
				},
			},
			exp: uint64(0),
		},
		{
			name: "GetStorageChild error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					Hash: &hash,
				},
			},
			wantErr: true,
			err:     errors.New("GetStorageChild error"),
		},
		{
			name: "GetStateRootFromBlock error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					Hash: &hash,
				},
			},
			wantErr: true,
			err:     errors.New("GetStateRootFromBlock error"),
		},
	}
	for _, tt := range tests {
		var res uint64
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = cs.GetStorageSize(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorageSize() error = %v, wantErr %v", err, tt.wantErr)
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

func TestChildStateModule_GetStorageHash(t *testing.T) {
	_, sr := createTestTrieState(t)

	mockStorageAPI := new(apimocks.StorageAPI)
	mockErrorStorageAPI1 := new(apimocks.StorageAPI)
	mockErrorStorageAPI2 := new(apimocks.StorageAPI)
	mockBlockAPI := new(apimocks.BlockAPI)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI.On("GetBlockHash").Return(hash)
	mockBlockAPI.On("BestBlockHash").Return(hash)

	mockStorageAPI.On("GetStateRootFromBlock", &hash).Return(&sr, nil)
	mockStorageAPI.On("GetStorageFromChild", &sr, []byte(":child_storage_key"), []byte(":child_first")).Return([]byte(""), nil)

	mockErrorStorageAPI1.On("GetStateRootFromBlock", &hash).Return(nil, nil)
	mockErrorStorageAPI1.On("GetStorageFromChild", (*common.Hash)(nil), []byte(nil), []byte(nil)).Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	type fields struct {
		storageAPI StorageAPI
		blockAPI   BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *GetStorageHash
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
			name: "Get Keys Nil Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetStorageHash{
					KeyChild: []byte(":child_storage_key"),
					EntryKey: []byte(":child_first"),
				},
			},
			exp: "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name: "Get Keys with Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetStorageHash{
					KeyChild: []byte(":child_storage_key"),
					EntryKey: []byte(":child_first"),
					Hash:     &hash,
				},
			},
			exp: "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name: "GetStorageChild error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetStorageHash{
					Hash: &hash,
				},
			},
			wantErr: true,
			err:     errors.New("GetStorageChild error"),
		},
		{
			name: "GetStateRootFromBlock error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetStorageHash{
					Hash: &hash,
				},
			},
			wantErr: true,
			err:     errors.New("GetStateRootFromBlock error"),
		},
	}
	for _, tt := range tests {
		var res string
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = cs.GetStorageHash(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorageHash() error = %v, wantErr %v", err, tt.wantErr)
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

func TestChildStateModule_GetStorage(t *testing.T) {
	_, sr := createTestTrieState(t)

	mockStorageAPI := new(apimocks.StorageAPI)
	mockErrorStorageAPI1 := new(apimocks.StorageAPI)
	mockErrorStorageAPI2 := new(apimocks.StorageAPI)
	mockBlockAPI := new(apimocks.BlockAPI)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI.On("GetBlockHash").Return(hash)
	mockBlockAPI.On("BestBlockHash").Return(hash)

	mockStorageAPI.On("GetStateRootFromBlock", &hash).Return(&sr, nil)
	mockStorageAPI.On("GetStorageFromChild", &sr, []byte(":child_storage_key"), []byte(":child_first")).Return([]byte("test"), nil)

	mockErrorStorageAPI1.On("GetStateRootFromBlock", &hash).Return(nil, nil)
	mockErrorStorageAPI1.On("GetStorageFromChild", (*common.Hash)(nil), []byte(nil), []byte(nil)).Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	type fields struct {
		storageAPI StorageAPI
		blockAPI   BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *ChildStateStorageRequest
		res *StateStorageResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     StateStorageResponse
	}{
		{
			name: "Get Keys Nil Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					ChildStorageKey: []byte(":child_storage_key"),
					Key:             []byte(":child_first"),
				},
			},
			exp: StateStorageResponse("0x74657374"),
		},
		{
			name: "Get Keys with Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					ChildStorageKey: []byte(":child_storage_key"),
					Key:             []byte(":child_first"),
					Hash:            &hash,
				},
			},
			exp: StateStorageResponse("0x74657374"),
		},
		{
			name: "GetStorageChild error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					Hash: &hash,
				},
			},
			wantErr: true,
			err:     errors.New("GetStorageChild error"),
		},
		{
			name: "GetStateRootFromBlock error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					Hash: &hash,
				},
			},
			wantErr: true,
			err:     errors.New("GetStateRootFromBlock error"),
		},
	}
	for _, tt := range tests {
		res := StateStorageResponse("")
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var err error
			if err = cs.GetStorage(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorage() error = %v, wantErr %v", err, tt.wantErr)
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
