// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/mock"
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

	mockErrorStorageAPI1.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	mockErrorStorageAPI1.On("GetStorageChild", mock.AnythingOfType("*common.Hash"), []byte(nil)).Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	res := make([]string, 0)
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
	}{
		{
			name: "Get Keys Nil Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Key:    []byte(":child_storage_key"),
				},
				res: &res,
			},
		},
		{
			name: "Get Keys with Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Key:    []byte(":child_storage_key"),
					Hash:   &hash,
				},
				res: &res,
			},
		},
		{
			name: "GetStorageChild error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Hash:   &common.Hash{},
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetStateRootFromBlock error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Key:    []byte(":child_storage_key"),
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := cs.GetKeys(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetKeys() error = %v, wantErr %v", err, tt.wantErr)
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

	mockErrorStorageAPI1.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	mockErrorStorageAPI1.On("GetStorageFromChild", mock.AnythingOfType("*common.Hash"), []byte(nil), []byte(nil)).Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	res := uint64(0)
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
				res: &res,
			},
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
				res: &res,
			},
		},
		{
			name: "GetStorageChild error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					Hash:     &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetStateRootFromBlock error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					Hash:     &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := cs.GetStorageSize(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorageSize() error = %v, wantErr %v", err, tt.wantErr)
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

	mockErrorStorageAPI1.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	mockErrorStorageAPI1.On("GetStorageFromChild", mock.AnythingOfType("*common.Hash"), []byte(nil), []byte(nil)).Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	res := ""
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
				res: &res,
			},
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
				res: &res,
			},
		},
		{
			name: "GetStorageChild error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetStorageHash{
					Hash:     &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetStateRootFromBlock error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetStorageHash{
					Hash:     &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := cs.GetStorageHash(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorageHash() error = %v, wantErr %v", err, tt.wantErr)
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

	mockErrorStorageAPI1.On("GetStateRootFromBlock", mock.AnythingOfType("*common.Hash")).Return(nil, nil)
	mockErrorStorageAPI1.On("GetStorageFromChild", mock.AnythingOfType("*common.Hash"), []byte(nil), []byte(nil)).Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.On("GetStateRootFromBlock", &hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	res := StateStorageResponse("")
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
				res: &res,
			},
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
				res: &res,
			},
		},
		{
			name: "GetStorageChild error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					Hash:            &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "GetStateRootFromBlock error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					Hash:            &hash,
				},
				res: &res,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			if err := cs.GetStorage(tt.args.in0, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("GetStorage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
