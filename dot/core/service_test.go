// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"context"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestDummyError = errors.New("test dummy error")
var testWasmPaths []string

func TestGenerateWasm(t *testing.T) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	require.NoError(t, err)
	testWasmPaths = wasmFilePaths
}

func Test_Service_StorageRoot(t *testing.T) {
	emptyTrie := trie.NewEmptyTrie()
	ts, err := rtstorage.NewTrieState(emptyTrie)
	require.NoError(t, err)
	
	tests := []struct {
		name         string
		service      *Service
		exp          common.Hash
		retTrieState *rtstorage.TrieState
		retErr       error
		expErr       error
		expErrMsg    string
	}{
		{
			name:      "nil storage state",
			service:   &Service{},
			expErr:    ErrNilStorageState,
			expErrMsg: ErrNilStorageState.Error(),
		},
		{
			name:      "storage trie state error",
			service:   &Service{},
			retErr:    errTestDummyError,
			expErr:    errTestDummyError,
			expErrMsg: errTestDummyError.Error(),
		},
		{
			name:    "storage trie state ok",
			service: &Service{},
			exp: common.Hash{0x3, 0x17, 0xa, 0x2e, 0x75, 0x97, 0xb7, 0xb7, 0xe3, 0xd8, 0x4c, 0x5, 0x39, 0x1d, 0x13, 0x9a,
				0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0, 0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14},
			retTrieState: ts,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			if tt.name != "nil storage state" {
				ctrl := gomock.NewController(t)
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().TrieState(nil).Return(tt.retTrieState, tt.retErr)
				s.storageState = mockStorageState
			}

			res, err := s.StorageRoot()
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func Test_Service_handleCodeSubstitution(t *testing.T) {
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
	runtimeMock.On("NetworkService").Return(new(runtime.TestRuntimeNetwork))
	runtimeMock.On("Validator").Return(true)

	ctrl := gomock.NewController(t)
	mockBlockStateGetRtErr := NewMockBlockState(ctrl)
	mockBlockStateGetRtErr.EXPECT().GetRuntime(gomock.Any()).Return(nil, errTestDummyError)

	mockBlockStateGetRtOk1 := NewMockBlockState(ctrl)
	mockBlockStateGetRtOk1.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock, nil)

	mockBlockStateGetRtOk2 := NewMockBlockState(ctrl)
	mockBlockStateGetRtOk2.EXPECT().GetRuntime(gomock.Any()).Return(runtimeMock, nil)
	mockBlockStateGetRtOk2.EXPECT().StoreRuntime(blockHash, gomock.Any())

	mockCodeSubState1 := NewMockCodeSubstitutedState(ctrl)
	mockCodeSubState1.EXPECT().StoreCodeSubstitutedBlockHash(blockHash).Return(errTestDummyError)

	mockCodeSubState2 := NewMockCodeSubstitutedState(ctrl)
	mockCodeSubState2.EXPECT().StoreCodeSubstitutedBlockHash(blockHash).Return(nil)

	type args struct {
		hash  common.Hash
		state *rtstorage.TrieState
	}
	tests := []struct {
		name      string
		service   *Service
		args      args
		expErr    error
		expErrMsg string
	}{
		{
			name:    "nil value",
			service: &Service{codeSubstitute: map[common.Hash]string{}},
			args: args{
				hash: common.Hash{},
			},
		},
		{
			name: "getRuntime error",
			service: &Service{
				codeSubstitute: testCodeSubstitute,
				blockState:     mockBlockStateGetRtErr,
			},
			args: args{
				hash: blockHash,
			},
			expErr:    errTestDummyError,
			expErrMsg: errTestDummyError.Error(),
		},
		{
			name: "code substitute error",
			service: &Service{
				codeSubstitute:       testCodeSubstitute,
				blockState:           mockBlockStateGetRtOk1,
				codeSubstitutedState: mockCodeSubState1,
			},
			args: args{
				hash: blockHash,
			},
			expErr:    errTestDummyError,
			expErrMsg: errTestDummyError.Error(),
		},
		{
			name: "happyPath",
			service: &Service{
				codeSubstitute:       testCodeSubstitute,
				blockState:           mockBlockStateGetRtOk2,
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

func Test_Service_handleBlock(t *testing.T) {
	emptyTrie := trie.NewEmptyTrie()
	trieState, err := rtstorage.NewTrieState(emptyTrie)
	require.NoError(t, err)

	testHeader := types.NewEmptyHeader()
	block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
	block.Header.Number = big.NewInt(21)

	ctrl := gomock.NewController(t)

	//Store trie error
	mockStorageStateErr := NewMockStorageState(ctrl)
	mockStorageStateErr.EXPECT().StoreTrie(trieState, &block.Header).Return(errTestDummyError)

	// add block error
	mockStorageStateOk1 := NewMockStorageState(ctrl)
	mockStorageStateOk1.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateErrNotFine := NewMockBlockState(ctrl)
	mockBlockStateErrNotFine.EXPECT().AddBlock(&block).Return(errTestDummyError)

	//add block prent not found error
	mockStorageStateOk2 := NewMockStorageState(ctrl)
	mockStorageStateOk2.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateErrNotFine2 := NewMockBlockState(ctrl)
	mockBlockStateErrNotFine2.EXPECT().AddBlock(&block).Return(blocktree.ErrParentNotFound)

	//add block cont err
	mockStorageStateOk3 := NewMockStorageState(ctrl)
	mockStorageStateOk3.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateErrFine := NewMockBlockState(ctrl)
	mockBlockStateErrFine.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
	mockBlockStateErrFine.EXPECT().GetRuntime(&block.Header.ParentHash).Return(nil, errTestDummyError)
	mockDigestHandler := NewMockDigestHandler(ctrl)
	mockDigestHandler.EXPECT().HandleDigests(&block.Header)

	//handle runtime changes error
	runtimeMock := new(mocksruntime.Instance)
	mockStorageStateOk4 := NewMockStorageState(ctrl)
	mockStorageStateOk4.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockStateRuntimeChangeErr := NewMockBlockState(ctrl)
	mockBlockStateRuntimeChangeErr.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
	mockBlockStateRuntimeChangeErr.EXPECT().GetRuntime(&block.Header.ParentHash).Return(runtimeMock, nil)
	mockBlockStateRuntimeChangeErr.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).
		Return(errTestDummyError)
	mockDigestHandler1 := NewMockDigestHandler(ctrl)
	mockDigestHandler1.EXPECT().HandleDigests(&block.Header)

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
		name      string
		service   *Service
		args      args
		expErr    error
		expErrMsg string
	}{
		{
			name:      "nil input",
			service:   &Service{},
			expErr:    ErrNilBlockHandlerParameter,
			expErrMsg: ErrNilBlockHandlerParameter.Error(),
		},
		{
			name:    "storeTrie error",
			service: &Service{storageState: mockStorageStateErr},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr:    errTestDummyError,
			expErrMsg: errTestDummyError.Error(),
		},
		{
			name: "addBlock quit error",
			service: &Service{
				storageState: mockStorageStateOk1,
				blockState:   mockBlockStateErrNotFine,
			},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr:    errTestDummyError,
			expErrMsg: errTestDummyError.Error(),
		},
		{
			name: "addBlock parent not found error",
			service: &Service{
				storageState: mockStorageStateOk2,
				blockState:   mockBlockStateErrNotFine2,
			},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr:    blocktree.ErrParentNotFound,
			expErrMsg: blocktree.ErrParentNotFound.Error(),
		},
		{
			name: "addBlock error continue",
			service: &Service{
				storageState:  mockStorageStateOk3,
				blockState:    mockBlockStateErrFine,
				digestHandler: mockDigestHandler,
			},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr:    errTestDummyError,
			expErrMsg: errTestDummyError.Error(),
		},
		{
			name: "handle runtime changes error",
			service: &Service{
				storageState:  mockStorageStateOk4,
				blockState:    mockBlockStateRuntimeChangeErr,
				digestHandler: mockDigestHandler1,
			},
			args: args{
				block: &block,
				state: trieState,
			},
			expErr:    errTestDummyError,
			expErrMsg: errTestDummyError.Error(),
		},
		{
			name: "code substitution ok",
			service: &Service{
				storageState:  mockStorageStateOk5,
				blockState:    mockBlockStateOk,
				digestHandler: mockDigestHandler2,
				ctx:           context.TODO(),
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

func Test_Service_HandleBlockProduced(t *testing.T) {
	emptyTrie := trie.NewEmptyTrie()
	trieState, err := rtstorage.NewTrieState(emptyTrie)
	require.NoError(t, err)

	digest := types.NewDigest()
	err = digest.Add(
		types.PreRuntimeDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		})
	require.NoError(t, err)

	testHeader := types.NewEmptyHeader()
	block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
	block.Header.Number = big.NewInt(21)
	block.Header.Digest = digest

	msg := &network.BlockAnnounceMessage{
		ParentHash:     block.Header.ParentHash,
		Number:         block.Header.Number,
		StateRoot:      block.Header.StateRoot,
		ExtrinsicsRoot: block.Header.ExtrinsicsRoot,
		Digest:         digest,
		BestBlock:      true,
	}

	ctrl := gomock.NewController(t)
	runtimeMock := new(mocksruntime.Instance)
	mockStorageStateOk := NewMockStorageState(ctrl)
	mockStorageStateOk.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
	mockBlockState.EXPECT().GetRuntime(&block.Header.ParentHash).Return(runtimeMock, nil)
	mockBlockState.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).Return(nil)
	mockDigestHandler := NewMockDigestHandler(ctrl)
	mockDigestHandler.EXPECT().HandleDigests(&block.Header)
	mockNetwork := NewMockNetwork(ctrl)
	mockNetwork.EXPECT().GossipMessage(msg)

	type args struct {
		block *types.Block
		state *rtstorage.TrieState
	}
	tests := []struct {
		name      string
		service   *Service
		args      args
		expErr    error
		expErrMsg string
	}{
		{
			name:      "nil input",
			service:   &Service{},
			expErr:    ErrNilBlockHandlerParameter,
			expErrMsg: ErrNilBlockHandlerParameter.Error(),
		},
		{
			name: "happy path",
			service: &Service{
				storageState:  mockStorageStateOk,
				blockState:    mockBlockState,
				digestHandler: mockDigestHandler,
				net:           mockNetwork,
				ctx:           context.TODO(),
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
			err := s.HandleBlockProduced(tt.args.block, tt.args.state)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}

func Test_Service_maintainTransactionPool(t *testing.T) {
	validity := &transaction.Validity{
		Priority:  0x3e8,
		Requires:  [][]byte{{0xb5, 0x47, 0xb1, 0x90, 0x37, 0x10, 0x7e, 0x1f, 0x79, 0x4c, 0xa8, 0x69, 0x0, 0xa1, 0xb5, 0x98}},
		Provides:  [][]byte{{0xe4, 0x80, 0x7d, 0x1b, 0x67, 0x49, 0x37, 0xbf, 0xc7, 0x89, 0xbb, 0xdd, 0x88, 0x6a, 0xdd, 0xd6}},
		Longevity: 0x40,
		Propagate: true,
	}

	extrinsic := types.Extrinsic{21}
	vt := transaction.NewValidTransaction(extrinsic, validity)

	testHeader := types.NewEmptyHeader()
	block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
	block.Header.Number = big.NewInt(21)

	ctrl := gomock.NewController(t)
	tx := transaction.NewValidTransaction(types.Extrinsic{21}, &transaction.Validity{Propagate: true})

	// valid txn err case
	runtimeMockErr := new(mocksruntime.Instance)
	runtimeMockErr.On("ValidateTransaction", types.Extrinsic{21}).Return(nil, errTestDummyError)
	mockTxnStateErr := NewMockTransactionState(ctrl)
	mockTxnStateErr.EXPECT().RemoveExtrinsic(types.Extrinsic{21}).MaxTimes(2)
	mockTxnStateErr.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{vt})
	mockBlockStateOk := NewMockBlockState(ctrl)
	mockBlockStateOk.EXPECT().GetRuntime(nil).Return(runtimeMockErr, nil)

	// valid txn case
	runtimeMockOk := new(mocksruntime.Instance)
	runtimeMockOk.On("ValidateTransaction", types.Extrinsic{21}).
		Return(&transaction.Validity{Propagate: true}, nil)
	mockTxnStateOk := NewMockTransactionState(ctrl)
	mockTxnStateOk.EXPECT().RemoveExtrinsic(types.Extrinsic{21})
	mockTxnStateOk.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{vt})
	mockTxnStateOk.EXPECT().Push(tx).Return(common.Hash{}, nil)
	mockTxnStateOk.EXPECT().RemoveExtrinsicFromPool(types.Extrinsic{21})
	mockBlockStateOk2 := NewMockBlockState(ctrl)
	mockBlockStateOk2.EXPECT().GetRuntime(nil).Return(runtimeMockOk, nil)

	type args struct {
		block *types.Block
	}
	tests := []struct {
		name    string
		service *Service
		args    args
	}{
		{
			name: "Validate Transaction err",
			service: &Service{
				transactionState: mockTxnStateErr,
				blockState:       mockBlockStateOk,
			},
			args: args{
				block: &block,
			},
		},
		{
			name: "Validate Transaction ok",
			service: &Service{
				transactionState: mockTxnStateOk,
				blockState:       mockBlockStateOk2,
			},
			args: args{
				block: &block,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			s.maintainTransactionPool(tt.args.block)
		})
	}
}

func Test_Service_handleBlocksAsync(t *testing.T) {
	validity := &transaction.Validity{
		Priority:  0x3e8,
		Requires:  [][]byte{{0xb5, 0x47, 0xb1, 0x90, 0x37, 0x10, 0x7e, 0x1f, 0x79, 0x4c, 0xa8, 0x69, 0x0, 0xa1, 0xb5, 0x98}},
		Provides:  [][]byte{{0xe4, 0x80, 0x7d, 0x1b, 0x67, 0x49, 0x37, 0xbf, 0xc7, 0x89, 0xbb, 0xdd, 0x88, 0x6a, 0xdd, 0xd6}},
		Longevity: 0x40,
		Propagate: true,
	}

	extrinsic := types.Extrinsic{21}
	vt := transaction.NewValidTransaction(extrinsic, validity)

	testHeader := types.NewEmptyHeader()
	block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
	block.Header.Number = big.NewInt(21)

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{}).MaxTimes(4)

	// chain reorg err case
	runtimeMockOk := new(mocksruntime.Instance)
	runtimeMockOk.On("ValidateTransaction", types.Extrinsic{21}).Return(nil, errTestDummyError)
	mockBlockStateReorgErr := NewMockBlockState(ctrl)
	mockBlockStateReorgErr.EXPECT().BestBlockHash().Return(common.Hash{}).MaxTimes(2)
	mockBlockStateReorgErr.EXPECT().HighestCommonAncestor(common.Hash{}, block.Header.Hash()).
		Return(common.Hash{}, errTestDummyError)
	mockBlockStateReorgErr.EXPECT().GetRuntime(nil).Return(runtimeMockOk, nil)
	mockTxnStateErr := NewMockTransactionState(ctrl)
	mockTxnStateErr.EXPECT().RemoveExtrinsic(types.Extrinsic{21}).MaxTimes(2)
	mockTxnStateErr.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{vt})

	// Testing channels
	blockAddChan := make(chan *types.Block)
	blockAddChan2 := make(chan *types.Block)
	close(blockAddChan2)
	blockAddChan3 := make(chan *types.Block)
	go func() {
		time.Sleep(1 * time.Second)
		blockAddChan3 <- nil
		close(blockAddChan3)
	}()
	blockAddChan4 := make(chan *types.Block)
	go func() {
		time.Sleep(1 * time.Second)
		blockAddChan4 <- &block
		close(blockAddChan4)
	}()

	// Closed context case
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Millisecond)
	cancel()

	tests := []struct {
		name    string
		service *Service
	}{
		{
			name: "closed context",
			service: &Service{
				blockState: mockBlockState,
				blockAddCh: blockAddChan,
				ctx:        ctx,
			},
		},
		{
			name: "channel not ok",
			service: &Service{
				blockState: mockBlockState,
				blockAddCh: blockAddChan2,
				ctx:        context.TODO(),
			},
		},
		{
			name: "nil block",
			service: &Service{
				blockState: mockBlockState,
				blockAddCh: blockAddChan3,
				ctx:        context.TODO(),
			},
		},
		{
			name: "handleChainReorg error",
			service: &Service{
				blockState:       mockBlockStateReorgErr,
				transactionState: mockTxnStateErr,
				blockAddCh:       blockAddChan4,
				ctx:              context.TODO(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			s.handleBlocksAsync()
		})
	}
}

func TestCleanup(t *testing.T) {
	err := runtime.RemoveFiles(testWasmPaths)
	require.NoError(t, err)
}
