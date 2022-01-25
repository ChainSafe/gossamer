// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"context"
	"errors"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/big"
	"os"

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

	runtimeMock := new(mocksruntime.Instance)
	runtimeMock.On("Keystore").Return(&keystore.GlobalKeystore{})
	runtimeMock.On("NodeStorage").Return(runtime.NodeStorage{})
	// Nil, might need to return something real but have to learn how
	runtimeMock.On("NetworkService").Return(new(runtime.TestRuntimeNetwork))
	runtimeMock.On("Validator").Return(true)

	ctrl := gomock.NewController(t)
	mockBlockStateGetRtErr := NewMockBlockState(ctrl)
	mockBlockStateGetRtErr.EXPECT().GetRuntime(gomock.Any()).Return(nil, testDummyError)

	mockBlockStateGetRtOk1 := NewMockBlockState(ctrl)
	mockBlockStateGetRtOk1.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock, nil)

	mockBlockStateGetRtOk2 := NewMockBlockState(ctrl)
	mockBlockStateGetRtOk2.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock, nil)
	mockBlockStateGetRtOk2.EXPECT().StoreRuntime(blockHash, gomock.Any())

	mockCodeSubState1 := NewMockCodeSubstitutedState(ctrl)
	mockCodeSubState1.EXPECT().StoreCodeSubstitutedBlockHash(blockHash).Return(testDummyError)

	mockCodeSubState2 := NewMockCodeSubstitutedState(ctrl)
	mockCodeSubState2.EXPECT().StoreCodeSubstitutedBlockHash(blockHash).Return(nil)

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
				blockState: mockBlockStateGetRtErr,
			},
			args: args{
				hash: blockHash,
			},
			expErr: testDummyError,
			expErrMsg: testDummyError.Error(),
		},
		{
			name: "code substitute error",
			service: &Service{
				codeSubstitute: testCodeSubstitute,
				blockState: mockBlockStateGetRtOk1,
				codeSubstitutedState: mockCodeSubState1,
			},
			args: args{
				hash: blockHash,
			},
			expErr: testDummyError,
			expErrMsg: testDummyError.Error(),
		},
		{
			name: "happyPath",
			service: &Service{
				codeSubstitute: testCodeSubstitute,
				blockState: mockBlockStateGetRtOk2,
				codeSubstitutedState: mockCodeSubState2,
			},
			args: args{
				hash: blockHash,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			err = s.handleCodeSubstitution(tt.args.hash, tt.args.state)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}

func TestService_handleBlock(t *testing.T) {
	emptyTrie := trie.NewEmptyTrie()
	trieState, err := rtstorage.NewTrieState(emptyTrie)
	require.NoError(t, err)

	testHeader := types.NewEmptyHeader()
	block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
	block.Header.Number = big.NewInt(21)

	ctrl := gomock.NewController(t)

	//Store trie error
	mockStorageStateErr := NewMockStorageState(ctrl)
	mockStorageStateErr.EXPECT().StoreTrie(trieState, &block.Header).Return(testDummyError)

	// add block error
	mockStorageStateOk1 := NewMockStorageState(ctrl)
	mockStorageStateOk1.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateErrNotFine := NewMockBlockState(ctrl)
	mockBlockStateErrNotFine.EXPECT().AddBlock(&block).Return(testDummyError)

	//add block prent not found error
	mockStorageStateOk2 := NewMockStorageState(ctrl)
	mockStorageStateOk2.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateErrNotFine2 := NewMockBlockState(ctrl)
	mockBlockStateErrNotFine2.EXPECT().AddBlock(&block).Return(blocktree.ErrParentNotFound)

	//add block cont err
	//runtimeMock := new(mocksruntime.Instance)
	mockStorageStateOk3 := NewMockStorageState(ctrl)
	mockStorageStateOk3.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateErrFine := NewMockBlockState(ctrl)
	mockBlockStateErrFine.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
	mockBlockStateErrFine.EXPECT().GetRuntime(&block.Header.ParentHash).Return(nil, testDummyError)
	mockDigestHandler := NewMockDigestHandler(ctrl)
	mockDigestHandler.EXPECT().HandleDigests(&block.Header)

	//handle runtime changes error
	runtimeMock := new(mocksruntime.Instance)
	mockStorageStateOk4 := NewMockStorageState(ctrl)
	mockStorageStateOk4.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateRuntimeChangeErr := NewMockBlockState(ctrl)
	mockBlockStateRuntimeChangeErr.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
	mockBlockStateRuntimeChangeErr.EXPECT().GetRuntime(&block.Header.ParentHash).Return(runtimeMock, nil)
	mockBlockStateRuntimeChangeErr.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).Return(testDummyError)
	mockDigestHandler1 := NewMockDigestHandler(ctrl)
	mockDigestHandler1.EXPECT().HandleDigests(&block.Header)

	//code sub error
	runtimeMock2 := new(mocksruntime.Instance)
	mockStorageStateOk5 := NewMockStorageState(ctrl)
	mockStorageStateOk5.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateOk := NewMockBlockState(ctrl)
	mockBlockStateOk.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
	mockBlockStateOk.EXPECT().GetRuntime(&block.Header.ParentHash).Return(runtimeMock2, nil)
	mockBlockStateOk.EXPECT().HandleRuntimeChanges(trieState, runtimeMock2, block.Header.Hash()).Return(nil)
	mockDigestHandler2 := NewMockDigestHandler(ctrl)
	mockDigestHandler2.EXPECT().HandleDigests(&block.Header)

	type args struct {
		block *types.Block
		state *rtstorage.TrieState
	}
	tests := []struct {
		name    string
		service  *Service
		args    args
		expErr error
		expErrMsg string
	}{
		{
			name: "nil input",
			service: &Service{},
			expErr: errNilBlockHandlerParameter,
			expErrMsg: errNilBlockHandlerParameter.Error(),
		},
		{
			name: "storeTrie error",
			service: &Service{storageState: mockStorageStateErr},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr: testDummyError,
			expErrMsg: testDummyError.Error(),
		},
		{
			name: "addBlock quit error",
			service: &Service{
				storageState: mockStorageStateOk1,
				blockState: mockBlockStateErrNotFine,
			},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr: testDummyError,
			expErrMsg: testDummyError.Error(),
		},
		{
			name: "addBlock parent not found error",
			service: &Service{
				storageState: mockStorageStateOk2,
				blockState: mockBlockStateErrNotFine2,
			},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr: blocktree.ErrParentNotFound,
			expErrMsg: blocktree.ErrParentNotFound.Error(),
		},
		{
			name: "addBlock error continue",
			service: &Service{
				storageState: mockStorageStateOk3,
				blockState: mockBlockStateErrFine,
				digestHandler: mockDigestHandler,
			},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr: testDummyError,
			expErrMsg: testDummyError.Error(),
		},
		{
			name: "handle runtime changes error",
			service: &Service{
				storageState: mockStorageStateOk4,
				blockState: mockBlockStateRuntimeChangeErr,
				digestHandler: mockDigestHandler1,
			},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr: testDummyError,
			expErrMsg: testDummyError.Error(),
		},
		{
			name: "code substitution ok",
			service: &Service{
				storageState: mockStorageStateOk5,
				blockState: mockBlockStateOk,
				digestHandler: mockDigestHandler2,
				ctx: context.Background(),
			},
			args: args{
				block: &block,
				state: trieState,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			err := s.handleBlock(tt.args.block, tt.args.state)
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
