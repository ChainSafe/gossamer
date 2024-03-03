// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	inmemory_storage "github.com/ChainSafe/gossamer/lib/runtime/storage/inmemory"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTrieState(t *testing.T) (*trie.InMemoryTrie, common.Hash) {
	t.Helper()

	_, genesisTrie, _ := newWestendLocalGenesisWithTrieAndHeader(t)
	tr := inmemory_storage.NewTrieState(genesisTrie)

	tr.Put([]byte(":first_key"), []byte(":value1"))
	tr.Put([]byte(":second_key"), []byte(":second_value"))

	childTr := trie.NewEmptyInmemoryTrie()
	childTr.Put([]byte(":child_first"), []byte(":child_first_value"))
	childTr.Put([]byte(":child_second"), []byte(":child_second_value"))
	childTr.Put([]byte(":another_child"), []byte("value"))

	err := tr.SetChild([]byte(":child_storage_key"), childTr)
	require.NoError(t, err)

	stateRoot, err := tr.Root()
	require.NoError(t, err)

	return tr.Trie(), stateRoot
}

func TestChildStateModule_GetKeys(t *testing.T) {
	ctrl := gomock.NewController(t)

	tr, sr := createTestTrieState(t)

	expKeys := tr.GetKeysWithPrefix([]byte{})
	expHexKeys := make([]string, len(expKeys))
	for idx, k := range expKeys {
		expHexKeys[idx] = common.BytesToHex(k)
	}

	mockStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI1 := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI2 := apimocks.NewMockStorageAPI(ctrl)
	mockBlockAPI := apimocks.NewMockBlockAPI(ctrl)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI.EXPECT().BestBlockHash().Return(hash).Times(2)

	mockStorageAPI.EXPECT().GetStateRootFromBlock(&hash).Return(&sr, nil).Times(2)
	mockStorageAPI.EXPECT().GetStorageChild(&sr, []byte(":child_storage_key")).
		Return(tr, nil).Times(2)

	mockErrorStorageAPI1.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(nil, nil)
	mockErrorStorageAPI1.EXPECT().GetStorageChild((*common.Hash)(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.EXPECT().GetStateRootFromBlock(&hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	type fields struct {
		storageAPI StorageAPI
		blockAPI   BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *GetKeysRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    []string
	}{
		{
			name: "Get_Keys_Nil_Hash",
			fields: fields{
				childStateModule.storageAPI,
				childStateModule.blockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Key: []byte(":child_storage_key"),
				},
			},
			exp: expHexKeys,
		},
		{
			name: "Get_Keys_with_Hash",
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
			exp: expHexKeys,
		},
		{
			name: "GetStorageChild_error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Hash: &common.Hash{},
				},
			},
			expErr: errors.New("GetStorageChild error"),
		},
		{
			name: "GetStateRootFromBlock_error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetKeysRequest{
					Key: []byte(":child_storage_key"),
				},
			},
			expErr: errors.New("GetStateRootFromBlock error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			var res []string
			err := cs.GetKeys(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestChildStateModule_GetStorageSize(t *testing.T) {
	ctrl := gomock.NewController(t)

	_, sr := createTestTrieState(t)

	mockStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI1 := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI2 := apimocks.NewMockStorageAPI(ctrl)
	mockBlockAPI := apimocks.NewMockBlockAPI(ctrl)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI.EXPECT().BestBlockHash().Return(hash)

	mockStorageAPI.EXPECT().GetStateRootFromBlock(&hash).Return(&sr, nil).Times(2)
	mockStorageAPI.EXPECT().GetStorageFromChild(&sr, []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte(""), nil).Times(2)

	mockErrorStorageAPI1.EXPECT().GetStateRootFromBlock(&hash).Return(nil, nil)
	mockErrorStorageAPI1.EXPECT().GetStorageFromChild((*common.Hash)(nil), []byte(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.EXPECT().GetStateRootFromBlock(&hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	type fields struct {
		storageAPI StorageAPI
		blockAPI   BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *GetChildStorageRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    uint64
	}{
		{
			name: "Get_Keys_Nil_Hash",
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
			name: "Get_Keys_with_Hash",
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
			name: "GetStorageChild_error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStorageChild error"),
		},
		{
			name: "GetStateRootFromBlock_error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStateRootFromBlock error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			res := uint64(0)
			err := cs.GetStorageSize(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestChildStateModule_GetStorageHash(t *testing.T) {
	ctrl := gomock.NewController(t)

	_, sr := createTestTrieState(t)

	mockStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI1 := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI2 := apimocks.NewMockStorageAPI(ctrl)
	mockBlockAPI := apimocks.NewMockBlockAPI(ctrl)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI.EXPECT().BestBlockHash().Return(hash)

	mockStorageAPI.EXPECT().GetStateRootFromBlock(&hash).Return(&sr, nil).Times(2)
	mockStorageAPI.EXPECT().GetStorageFromChild(&sr, []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte(""), nil).Times(2)

	mockErrorStorageAPI1.EXPECT().GetStateRootFromBlock(&hash).Return(nil, nil)
	mockErrorStorageAPI1.EXPECT().GetStorageFromChild((*common.Hash)(nil), []byte(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.EXPECT().GetStateRootFromBlock(&hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	type fields struct {
		storageAPI StorageAPI
		blockAPI   BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *GetStorageHash
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    string
	}{
		{
			name: "Get_Keys_Nil_Hash",
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
			name: "Get_Keys_with_Hash",
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
			name: "GetStorageChild_error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &GetStorageHash{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStorageChild error"),
		},
		{
			name: "GetStateRootFromBlock_error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &GetStorageHash{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStateRootFromBlock error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			res := ""
			err := cs.GetStorageHash(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestChildStateModule_GetStorage(t *testing.T) {
	ctrl := gomock.NewController(t)

	_, sr := createTestTrieState(t)

	mockStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI1 := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI2 := apimocks.NewMockStorageAPI(ctrl)
	mockBlockAPI := apimocks.NewMockBlockAPI(ctrl)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")
	mockBlockAPI.EXPECT().BestBlockHash().Return(hash)

	mockStorageAPI.EXPECT().GetStateRootFromBlock(&hash).Return(&sr, nil).Times(2)
	mockStorageAPI.EXPECT().GetStorageFromChild(&sr, []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte("test"), nil).Times(2)

	mockErrorStorageAPI1.EXPECT().GetStateRootFromBlock(&hash).Return(nil, nil)
	mockErrorStorageAPI1.EXPECT().GetStorageFromChild((*common.Hash)(nil), []byte(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error"))

	mockErrorStorageAPI2.EXPECT().GetStateRootFromBlock(&hash).Return(nil, errors.New("GetStateRootFromBlock error"))

	childStateModule := NewChildStateModule(mockStorageAPI, mockBlockAPI)
	type fields struct {
		storageAPI StorageAPI
		blockAPI   BlockAPI
	}
	type args struct {
		in0 *http.Request
		req *ChildStateStorageRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    StateStorageResponse
	}{
		{
			name: "Get_Keys_Nil_Hash",
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
			name: "Get_Keys_with_Hash",
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
			name: "GetStorageChild_error",
			fields: fields{
				mockErrorStorageAPI1,
				mockBlockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStorageChild error"),
		},
		{
			name: "GetStateRootFromBlock_error",
			fields: fields{
				mockErrorStorageAPI2,
				mockBlockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStateRootFromBlock error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChildStateModule{
				storageAPI: tt.fields.storageAPI,
				blockAPI:   tt.fields.blockAPI,
			}
			res := StateStorageResponse("")
			err := cs.GetStorage(tt.args.in0, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
