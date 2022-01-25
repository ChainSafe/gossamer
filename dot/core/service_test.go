// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"errors"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"os"

	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

var testDummyError = errors.New("test dummy error")
var testWasmPaths []string

func TestGenerateWasm(t *testing.T) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	require.NoError(t, err)
	testWasmPaths = wasmFilePaths
}

func TestService_StorageRoot(t *testing.T) {
	emptyTrie := trie.NewEmptyTrie()
	ts, err := rtstorage.NewTrieState(emptyTrie)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockStorageState := NewMockStorageState(ctrl)
	mockStorageState.EXPECT().TrieState(nil).Return(nil, testDummyError)
	mockStorageStateErr := NewMockStorageState(ctrl)
	mockStorageStateErr.EXPECT().TrieState(nil).Return(ts, nil)
	tests := []struct {
		name    string
		service  *Service
		exp    common.Hash
		expErr error
		expErrMsg string
	}{
		{
			name: "nil storage state",
			service: &Service{},
			expErr: ErrNilStorageState,
			expErrMsg: ErrNilStorageState.Error(),
		},
		{
			name: "storage trie state error",
			service: &Service{storageState: mockStorageState},
			expErr: testDummyError,
			expErrMsg: testDummyError.Error(),
		},
		{
			name: "storage trie state ok",
			service: &Service{storageState: mockStorageStateErr},
			exp: common.Hash{0x3, 0x17, 0xa, 0x2e, 0x75, 0x97, 0xb7, 0xb7, 0xe3, 0xd8, 0x4c, 0x5, 0x39, 0x1d, 0x13, 0x9a,
				0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0, 0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			res, err := s.StorageRoot()
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestService_handleCodeSubstitution(t *testing.T) {
	testRuntime, err := os.ReadFile(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)

	// hash for known test code substitution
	blockHash := common.MustHexToHash("0x86aa36a140dfc449c30dbce16ce0fea33d5c3786766baa764e33f336841b9e29")
	testCodeSubstitute := map[common.Hash]string{
		blockHash: common.BytesToHex(testRuntime),
	}

	//testCodeSubstitute := map[common.Hash]string{}
	//testCodeSubstitute[common.MustHexToHash("0x01")] = "0x1111"

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetRuntime(gomock.Any()).Return(nil, testDummyError)
	type args struct {
		hash  common.Hash
		state *rtstorage.TrieState
	}
	tests := []struct {
		name    string
		service  *Service
		args    args
		expErr  error
		expErrMsg string
	}{
		{
			name: "nil value",
			service: &Service{codeSubstitute: map[common.Hash]string{}},
			args: args{
				hash: common.Hash{},
			},
		},
		{
			name: "getRuntime error",
			service: &Service{
				codeSubstitute: testCodeSubstitute,
				blockState: mockBlockState,
			},
			args: args{
				hash: blockHash,
			},
			expErr: testDummyError,
			expErrMsg: testDummyError.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			err := s.handleCodeSubstitution(tt.args.hash, tt.args.state)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	err := runtime.RemoveFiles(testWasmPaths)
	require.NoError(t, err)
}
