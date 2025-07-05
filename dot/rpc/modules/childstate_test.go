// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTrieState(t *testing.T) trie.Trie {
	t.Helper()

	_, genesisTrie, _ := newWestendLocalGenesisWithTrieAndHeader(t)
	tr := rtstorage.NewInMemoryTrieState(genesisTrie)

	err := tr.SetChildStorage([]byte(":child_storage_key"), []byte(":child_first"), []byte(":child_first_value"))
	require.NoError(t, err)

	err = tr.SetChildStorage([]byte(":child_storage_key"), []byte(":child_second"), []byte(":child_second_value"))
	require.NoError(t, err)

	err = tr.SetChildStorage([]byte(":child_storage_key"), []byte(":another_child"), []byte("value"))
	require.NoError(t, err)

	return genesisTrie
}

func TestChildStateModule_GetKeys(t *testing.T) {
	ctrl := gomock.NewController(t)

	tr := createTestTrieState(t)

	expKeys := tr.GetKeysWithPrefix([]byte{})
	expHexKeys := make([]string, len(expKeys))
	for idx, k := range expKeys {
		expHexKeys[idx] = common.BytesToHex(k)
	}

	mockStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI := apimocks.NewMockStorageAPI(ctrl)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI.EXPECT().
		GetStorageChild((*common.Hash)(nil), []byte(":child_storage_key")).
		Return(tr, nil).
		MaxTimes(2)
	mockStorageAPI.EXPECT().
		GetStorageChild(&hash, []byte(":child_storage_key")).
		Return(tr, nil).
		MaxTimes(2)

	mockErrorStorageAPI.EXPECT().
		GetStorageChild((*common.Hash)(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error")).
		MaxTimes(2)
	mockErrorStorageAPI.EXPECT().
		GetStorageChild(&common.Hash{}, []byte(nil)).
		Return(nil, errors.New("GetStorageChild error")).
		MaxTimes(2)

	childStateModule := NewChildStateModule(mockStorageAPI, nil)
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
				mockErrorStorageAPI,
				nil,
			},
			args: args{
				req: &GetKeysRequest{
					Hash: &common.Hash{},
				},
			},
			expErr: errors.New("GetStorageChild error"),
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

	mockStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockBlockAPI := apimocks.NewMockBlockAPI(ctrl)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI.EXPECT().
		GetStorageFromChild(&hash, []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte(""), nil).
		MaxTimes(2)
	mockStorageAPI.EXPECT().
		GetStorageFromChild((*common.Hash)(nil), []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte(""), nil).
		MaxTimes(2)

	mockErrorStorageAPI.EXPECT().
		GetStorageFromChild(&hash, []byte(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error")).
		MaxTimes(2)
	mockErrorStorageAPI.EXPECT().
		GetStorageFromChild((*common.Hash)(nil), []byte(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error")).
		MaxTimes(2)

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
				mockErrorStorageAPI,
				mockBlockAPI,
			},
			args: args{
				req: &GetChildStorageRequest{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStorageChild error"),
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

	mockStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockBlockAPI := apimocks.NewMockBlockAPI(ctrl)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI.EXPECT().
		GetStorageFromChild((*common.Hash)(nil), []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte(""), nil).
		MaxTimes(2)
	mockStorageAPI.EXPECT().
		GetStorageFromChild(&hash, []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte(""), nil).
		MaxTimes(2)

	mockErrorStorageAPI.EXPECT().
		GetStorageFromChild(&hash, []byte(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error")).
		MaxTimes(2)

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
				mockErrorStorageAPI,
				mockBlockAPI,
			},
			args: args{
				req: &GetStorageHash{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStorageChild error"),
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

	mockStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockErrorStorageAPI := apimocks.NewMockStorageAPI(ctrl)
	mockBlockAPI := apimocks.NewMockBlockAPI(ctrl)

	hash := common.MustHexToHash("0x3aa96b0149b6ca3688878bdbd19464448624136398e3ce45b9e755d3ab61355a")

	mockStorageAPI.EXPECT().
		GetStorageFromChild(&hash, []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte("test"), nil).
		MaxTimes(2)
	mockStorageAPI.EXPECT().
		GetStorageFromChild((*common.Hash)(nil), []byte(":child_storage_key"), []byte(":child_first")).
		Return([]byte("test"), nil).
		MaxTimes(2)

	mockErrorStorageAPI.EXPECT().
		GetStorageFromChild(&hash, []byte(nil), []byte(nil)).
		Return(nil, errors.New("GetStorageChild error")).
		MaxTimes(2)

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
				mockErrorStorageAPI,
				mockBlockAPI,
			},
			args: args{
				req: &ChildStateStorageRequest{
					Hash: &hash,
				},
			},
			expErr: errors.New("GetStorageChild error"),
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
