// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"bytes"
	"context"
	"errors"
	testdata "github.com/ChainSafe/gossamer/dot/rpc/modules/test_data"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	cscale "github.com/centrifuge/go-substrate-rpc-client/v3/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	mocksruntime "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestDummyError = errors.New("test dummy error")
var testWasmPaths []string

func generateExtrinsic(t *testing.T) (ext types.Extrinsic, externExt types.Extrinsic, body *types.Body) {
	rawMeta := common.MustHexToBytes(testdata.NewTestMetadata())
	var decoded []byte
	err := scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = ctypes.DecodeFromBytes(decoded, meta)
	require.NoError(t, err)

	testAPIItem := runtime.APIItem{
		Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Ver:  99,
	}
	rv := runtime.NewVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		25,
		0,
		[]runtime.APIItem{testAPIItem},
		5,
	)
	require.NoError(t, err)

	bob, err := ctypes.NewMultiAddressFromHexAccountID(
		"0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22")
	require.NoError(t, err)

	call, err := ctypes.NewCall(meta, "Balances.transfer", bob, ctypes.NewUCompactFromUInt(12345))
	require.NoError(t, err)

	// Create the extrinsic
	extrinsic := ctypes.NewExtrinsic(call)
	genHash, err := ctypes.NewHashFromHexString("0x35a28a7dbaf0ba07d1485b0f3da7757e3880509edc8c31d0850cb6dd6219361d")
	require.NoError(t, err)
	o := ctypes.SignatureOptions{
		BlockHash:          genHash,
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        genHash,
		Nonce:              ctypes.NewUCompactFromUInt(uint64(0)),
		SpecVersion:        ctypes.U32(rv.SpecVersion()),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion()),
	}

	// Sign the transaction using Alice's default account
	err = extrinsic.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	// Encode the signed extrinsic
	extEnc := bytes.Buffer{}
	encoder := cscale.NewEncoder(&extEnc)
	err = extrinsic.Encode(*encoder)
	require.NoError(t, err)

	encExt := []types.Extrinsic{extEnc.Bytes()}
	testExternalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, encExt[0]...))
	testUnencryptedBody := types.NewBody(encExt)
	return encExt[0], testExternalExt, testUnencryptedBody
}

func TestGenerateWasm(t *testing.T) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	require.NoError(t, err)
	testWasmPaths = wasmFilePaths
}

func Test_Service_StorageRoot(t *testing.T) {
	t.Parallel()
	emptyTrie := trie.NewEmptyTrie()
	ts, err := rtstorage.NewTrieState(emptyTrie)
	require.NoError(t, err)
	tests := []struct {
		name          string
		service       *Service
		exp           common.Hash
		retTrieState  *rtstorage.TrieState
		trieStateCall bool
		retErr        error
		expErr        error
		expErrMsg     string
	}{
		{
			name:      "nil storage state",
			service:   &Service{},
			expErr:    ErrNilStorageState,
			expErrMsg: ErrNilStorageState.Error(),
		},
		{
			name:          "storage trie state error",
			service:       &Service{},
			retErr:        errTestDummyError,
			expErr:        errTestDummyError,
			expErrMsg:     errTestDummyError.Error(),
			trieStateCall: true,
		},
		{
			name:    "storage trie state ok",
			service: &Service{},
			exp: common.Hash{0x3, 0x17, 0xa, 0x2e, 0x75, 0x97, 0xb7, 0xb7, 0xe3, 0xd8, 0x4c, 0x5, 0x39, 0x1d, 0x13, 0x9a,
				0x62, 0xb1, 0x57, 0xe7, 0x87, 0x86, 0xd8, 0xc0, 0x82, 0xf2, 0x9d, 0xcf, 0x4c, 0x11, 0x13, 0x14},
			retTrieState:  ts,
			trieStateCall: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := tt.service
			if tt.trieStateCall {
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
	t.Parallel()
	newTestInstance := func(code []byte, cfg *wasmer.Config) (*wasmer.Instance, error) {
		return &wasmer.Instance{}, nil
	}

	execTest := func(t *testing.T, s *Service, blockHash common.Hash, expErr error) {
		err := s._handleCodeSubstitution(blockHash, nil, newTestInstance)
		assert.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, errTestDummyError.Error())
		}
	}
	testRuntime := []byte{21}
	t.Run("nil value", func(t *testing.T) {
		t.Parallel()
		s := &Service{codeSubstitute: map[common.Hash]string{}}
		err := s._handleCodeSubstitution(common.Hash{}, nil, newTestInstance)
		assert.NoError(t, err)
	})

	t.Run("getRuntime error", func(t *testing.T) {
		t.Parallel()
		// hash for known test code substitution
		blockHash := common.MustHexToHash("0x86aa36a140dfc449c30dbce16ce0fea33d5c3786766baa764e33f336841b9e29")
		testCodeSubstitute := map[common.Hash]string{
			blockHash: common.BytesToHex(testRuntime),
		}

		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().GetRuntime(&blockHash).Return(nil, errTestDummyError)
		s := &Service{
			codeSubstitute: testCodeSubstitute,
			blockState:     mockBlockState,
		}
		execTest(t, s, blockHash, errTestDummyError)
	})

	t.Run("code substitute error", func(t *testing.T) {
		t.Parallel()
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
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().GetRuntime(&blockHash).Return(runtimeMock, nil)
		mockCodeSubState := NewMockCodeSubstitutedState(ctrl)
		mockCodeSubState.EXPECT().StoreCodeSubstitutedBlockHash(blockHash).Return(errTestDummyError)
		s := &Service{
			codeSubstitute:       testCodeSubstitute,
			blockState:           mockBlockState,
			codeSubstitutedState: mockCodeSubState,
		}
		execTest(t, s, blockHash, errTestDummyError)
	})

	t.Run("happyPath", func(t *testing.T) {
		t.Parallel()
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
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().GetRuntime(&blockHash).Return(runtimeMock, nil)
		mockBlockState.EXPECT().StoreRuntime(blockHash, gomock.Any())
		mockCodeSubState := NewMockCodeSubstitutedState(ctrl)
		mockCodeSubState.EXPECT().StoreCodeSubstitutedBlockHash(blockHash).Return(nil)
		s := &Service{
			codeSubstitute:       testCodeSubstitute,
			blockState:           mockBlockState,
			codeSubstitutedState: mockCodeSubState,
		}
		err := s._handleCodeSubstitution(blockHash, nil, newTestInstance)
		assert.NoError(t, err)
	})
}

func Test_Service_handleBlock(t *testing.T) {
	t.Parallel()
	execTest := func(t *testing.T, s *Service, block *types.Block, trieState *rtstorage.TrieState, expErr error) {
		err := s.handleBlock(block, trieState)
		assert.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, expErr.Error())
		}
	}
	t.Run("nil input", func(t *testing.T) {
		t.Parallel()
		s := &Service{}
		execTest(t, s, nil, nil, ErrNilBlockHandlerParameter)
	})

	t.Run("storeTrie error", func(t *testing.T) {
		t.Parallel()
		emptyTrie := trie.NewEmptyTrie()
		trieState, err := rtstorage.NewTrieState(emptyTrie)
		require.NoError(t, err)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(errTestDummyError)

		s := &Service{storageState: mockStorageState}
		execTest(t, s, &block, trieState, errTestDummyError)
	})

	t.Run("addBlock quit error", func(t *testing.T) {
		t.Parallel()
		emptyTrie := trie.NewEmptyTrie()
		trieState, err := rtstorage.NewTrieState(emptyTrie)
		require.NoError(t, err)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(errTestDummyError)

		s := &Service{
			storageState: mockStorageState,
			blockState:   mockBlockState,
		}
		execTest(t, s, &block, trieState, errTestDummyError)
	})

	t.Run("addBlock parent not found error", func(t *testing.T) {
		t.Parallel()
		emptyTrie := trie.NewEmptyTrie()
		trieState, err := rtstorage.NewTrieState(emptyTrie)
		require.NoError(t, err)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrParentNotFound)

		s := &Service{
			storageState: mockStorageState,
			blockState:   mockBlockState,
		}
		execTest(t, s, &block, trieState, blocktree.ErrParentNotFound)
	})

	t.Run("addBlock error continue", func(t *testing.T) {
		t.Parallel()
		emptyTrie := trie.NewEmptyTrie()
		trieState, err := rtstorage.NewTrieState(emptyTrie)
		require.NoError(t, err)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
		mockBlockState.EXPECT().GetRuntime(&block.Header.ParentHash).Return(nil, errTestDummyError)
		mockDigestHandler := NewMockDigestHandler(ctrl)
		mockDigestHandler.EXPECT().HandleDigests(&block.Header)

		s := &Service{
			storageState:  mockStorageState,
			blockState:    mockBlockState,
			digestHandler: mockDigestHandler,
		}
		execTest(t, s, &block, trieState, errTestDummyError)
	})

	t.Run("handle runtime changes error", func(t *testing.T) {
		t.Parallel()
		emptyTrie := trie.NewEmptyTrie()
		trieState, err := rtstorage.NewTrieState(emptyTrie)
		require.NoError(t, err)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		ctrl := gomock.NewController(t)
		runtimeMock := new(mocksruntime.Instance)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
		mockBlockState.EXPECT().GetRuntime(&block.Header.ParentHash).Return(runtimeMock, nil)
		mockBlockState.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).
			Return(errTestDummyError)
		mockDigestHandler := NewMockDigestHandler(ctrl)
		mockDigestHandler.EXPECT().HandleDigests(&block.Header)

		s := &Service{
			storageState:  mockStorageState,
			blockState:    mockBlockState,
			digestHandler: mockDigestHandler,
		}
		execTest(t, s, &block, trieState, errTestDummyError)
	})

	t.Run("code substitution ok", func(t *testing.T) {
		t.Parallel()
		emptyTrie := trie.NewEmptyTrie()
		trieState, err := rtstorage.NewTrieState(emptyTrie)
		require.NoError(t, err)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		ctrl := gomock.NewController(t)
		runtimeMock := new(mocksruntime.Instance)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
		mockBlockState.EXPECT().GetRuntime(&block.Header.ParentHash).Return(runtimeMock, nil)
		mockBlockState.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).Return(nil)
		mockDigestHandler := NewMockDigestHandler(ctrl)
		mockDigestHandler.EXPECT().HandleDigests(&block.Header)

		s := &Service{
			storageState:  mockStorageState,
			blockState:    mockBlockState,
			digestHandler: mockDigestHandler,
			ctx:           context.TODO(),
		}
		execTest(t, s, &block, trieState, nil)
	})
}

func Test_Service_HandleBlockProduced(t *testing.T) {
	t.Parallel()
	execTest := func(t *testing.T, s *Service, block *types.Block, trieState *rtstorage.TrieState, expErr error) {
		err := s.HandleBlockProduced(block, trieState)
		assert.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, expErr.Error())
		}
	}
	t.Run("nil input", func(t *testing.T) {
		t.Parallel()
		s := &Service{}
		execTest(t, s, nil, nil, ErrNilBlockHandlerParameter)
	})

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
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
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
		mockBlockState.EXPECT().GetRuntime(&block.Header.ParentHash).Return(runtimeMock, nil)
		mockBlockState.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).Return(nil)
		mockDigestHandler := NewMockDigestHandler(ctrl)
		mockDigestHandler.EXPECT().HandleDigests(&block.Header)
		mockNetwork := NewMockNetwork(ctrl)
		mockNetwork.EXPECT().GossipMessage(msg)

		s := &Service{
			storageState:  mockStorageState,
			blockState:    mockBlockState,
			digestHandler: mockDigestHandler,
			net:           mockNetwork,
			ctx:           context.TODO(),
		}
		execTest(t, s, &block, trieState, nil)
	})
}

func Test_Service_maintainTransactionPool(t *testing.T) {
	t.Parallel()
	t.Run("Validate Transaction err", func(t *testing.T) {
		t.Parallel()
		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		validity := &transaction.Validity{
			Priority: 0x3e8,
			Requires: [][]byte{{0xb5, 0x47, 0xb1, 0x90, 0x37, 0x10, 0x7e, 0x1f, 0x79,
				0x4c, 0xa8, 0x69, 0x0, 0xa1, 0xb5, 0x98}},
			Provides: [][]byte{{0xe4, 0x80, 0x7d, 0x1b, 0x67, 0x49, 0x37, 0xbf, 0xc7,
				0x89, 0xbb, 0xdd, 0x88, 0x6a, 0xdd, 0xd6}},
			Longevity: 0x40,
			Propagate: true,
		}

		extrinsic := types.Extrinsic{21}
		vt := transaction.NewValidTransaction(extrinsic, validity)

		ctrl := gomock.NewController(t)
		runtimeMock := new(mocksruntime.Instance)
		runtimeMock.On("ValidateTransaction", types.Extrinsic{21}).Return(nil, errTestDummyError)
		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().RemoveExtrinsic(types.Extrinsic{21}).MaxTimes(2)
		mockTxnState.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{vt})
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().GetRuntime(nil).Return(runtimeMock, nil)
		s := &Service{
			transactionState: mockTxnState,
			blockState:       mockBlockState,
		}
		s.maintainTransactionPool(&block)
	})

	t.Run("Validate Transaction ok", func(t *testing.T) {
		t.Parallel()
		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		validity := &transaction.Validity{
			Priority: 0x3e8,
			Requires: [][]byte{{0xb5, 0x47, 0xb1, 0x90, 0x37, 0x10, 0x7e, 0x1f, 0x79, 0x4c,
				0xa8, 0x69, 0x0, 0xa1, 0xb5, 0x98}},
			Provides: [][]byte{{0xe4, 0x80, 0x7d, 0x1b, 0x67, 0x49, 0x37, 0xbf, 0xc7, 0x89,
				0xbb, 0xdd, 0x88, 0x6a, 0xdd, 0xd6}},
			Longevity: 0x40,
			Propagate: true,
		}

		extrinsic := types.Extrinsic{21}
		vt := transaction.NewValidTransaction(extrinsic, validity)
		tx := transaction.NewValidTransaction(types.Extrinsic{21}, &transaction.Validity{Propagate: true})

		ctrl := gomock.NewController(t)
		runtimeMock := new(mocksruntime.Instance)
		runtimeMock.On("ValidateTransaction", types.Extrinsic{21}).
			Return(&transaction.Validity{Propagate: true}, nil)
		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().RemoveExtrinsic(types.Extrinsic{21})
		mockTxnState.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{vt})
		mockTxnState.EXPECT().Push(tx).Return(common.Hash{}, nil)
		mockTxnState.EXPECT().RemoveExtrinsicFromPool(types.Extrinsic{21})
		mockBlockStateOk := NewMockBlockState(ctrl)
		mockBlockStateOk.EXPECT().GetRuntime(nil).Return(runtimeMock, nil)
		s := &Service{
			transactionState: mockTxnState,
			blockState:       mockBlockStateOk,
		}
		s.maintainTransactionPool(&block)
	})
}

func Test_Service_handleBlocksAsync(t *testing.T) {
	t.Parallel()
	t.Run("cancelled context", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		blockAddChan := make(chan *types.Block)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		s := &Service{
			blockState: mockBlockState,
			blockAddCh: blockAddChan,
			ctx:        ctx,
		}
		s.handleBlocksAsync()
	})

	t.Run("channel not ok", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		blockAddChan := make(chan *types.Block)
		close(blockAddChan)
		s := &Service{
			blockState: mockBlockState,
			blockAddCh: blockAddChan,
			ctx:        context.Background(),
		}
		s.handleBlocksAsync()
	})

	t.Run("nil block", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{}).Times(2)
		blockAddChan := make(chan *types.Block)
		go func() {
			blockAddChan <- nil
			close(blockAddChan)
		}()
		s := &Service{
			blockState: mockBlockState,
			blockAddCh: blockAddChan,
			ctx:        context.Background(),
		}
		s.handleBlocksAsync()
	})

	t.Run("handleChainReorg error", func(t *testing.T) {
		t.Parallel()
		validity := &transaction.Validity{
			Priority: 0x3e8,
			Requires: [][]byte{{0xb5, 0x47, 0xb1, 0x90, 0x37, 0x10, 0x7e, 0x1f, 0x79, 0x4c,
				0xa8, 0x69, 0x0, 0xa1, 0xb5, 0x98}},
			Provides: [][]byte{{0xe4, 0x80, 0x7d, 0x1b, 0x67, 0x49, 0x37, 0xbf, 0xc7, 0x89,
				0xbb, 0xdd, 0x88, 0x6a, 0xdd, 0xd6}},
			Longevity: 0x40,
			Propagate: true,
		}

		extrinsic := types.Extrinsic{21}
		vt := transaction.NewValidTransaction(extrinsic, validity)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = big.NewInt(21)

		ctrl := gomock.NewController(t)
		runtimeMock := new(mocksruntime.Instance)
		runtimeMock.On("ValidateTransaction", types.Extrinsic{21}).Return(nil, errTestDummyError)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{}).Times(2)
		mockBlockState.EXPECT().HighestCommonAncestor(common.Hash{}, block.Header.Hash()).
			Return(common.Hash{}, errTestDummyError)
		mockBlockState.EXPECT().GetRuntime(nil).Return(runtimeMock, nil)
		mockTxnStateErr := NewMockTransactionState(ctrl)
		mockTxnStateErr.EXPECT().RemoveExtrinsic(types.Extrinsic{21}).Times(2)
		mockTxnStateErr.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{vt})
		blockAddChan := make(chan *types.Block)
		go func() {
			blockAddChan <- &block
			close(blockAddChan)
		}()
		s := &Service{
			blockState:       mockBlockState,
			transactionState: mockTxnStateErr,
			blockAddCh:       blockAddChan,
			ctx:              context.Background(),
		}
		s.handleBlocksAsync()
	})
}

func TestService_handleChainReorg(t *testing.T) {
	testPrevHash := common.MustHexToHash("0x01")
	testCurrentHash := common.MustHexToHash("0x02")
	testAncestorHash := common.MustHexToHash("0x03")
	testSubChain := []common.Hash{testPrevHash, testCurrentHash, testAncestorHash}
	ext, externExt, body := generateExtrinsic(t)
	testValidity := &transaction.Validity{Propagate: true}
	vtx := transaction.NewValidTransaction(ext, testValidity)

	ctrl := gomock.NewController(t)
	mockBlockStateAncestorErr := NewMockBlockState(ctrl)
	mockBlockStateAncestorErr.EXPECT().HighestCommonAncestor(testPrevHash, testCurrentHash).
		Return(common.Hash{}, errDummyErr)

	mockBlockStateAncestorEqPriv := NewMockBlockState(ctrl)
	mockBlockStateAncestorEqPriv.EXPECT().HighestCommonAncestor(testPrevHash, testCurrentHash).
		Return(testPrevHash, nil)

	mockBlockStateSubChainErr := NewMockBlockState(ctrl)
	mockBlockStateSubChainErr.EXPECT().HighestCommonAncestor(testPrevHash, testCurrentHash).
		Return(testAncestorHash, nil)
	mockBlockStateSubChainErr.EXPECT().SubChain(testAncestorHash, testPrevHash).Return([]common.Hash{}, errDummyErr)

	mockBlockStateEmptySubChain := NewMockBlockState(ctrl)
	mockBlockStateEmptySubChain.EXPECT().HighestCommonAncestor(testPrevHash, testCurrentHash).
		Return(testAncestorHash, nil)
	mockBlockStateEmptySubChain.EXPECT().SubChain(testAncestorHash, testPrevHash).Return([]common.Hash{}, nil)

	mockBlockStateRuntimeErr := NewMockBlockState(ctrl)
	mockBlockStateRuntimeErr.EXPECT().HighestCommonAncestor(testPrevHash, testCurrentHash).
		Return(testAncestorHash, nil)
	mockBlockStateRuntimeErr.EXPECT().SubChain(testAncestorHash, testPrevHash).Return(testSubChain, nil)
	mockBlockStateRuntimeErr.EXPECT().GetRuntime(nil).Return(nil, errDummyErr)

	// Invalid transaction
	runtimeMockErr := new(mocksruntime.Instance)
	mockBlockStateBlockBodyErr := NewMockBlockState(ctrl)
	mockBlockStateBlockBodyErr.EXPECT().HighestCommonAncestor(testPrevHash, testCurrentHash).
		Return(testAncestorHash, nil)
	mockBlockStateBlockBodyErr.EXPECT().SubChain(testAncestorHash, testPrevHash).Return(testSubChain, nil)
	mockBlockStateBlockBodyErr.EXPECT().GetRuntime(nil).Return(runtimeMockErr, nil)
	mockBlockStateBlockBodyErr.EXPECT().GetBlockBody(testCurrentHash).Return(nil, errDummyErr)
	mockBlockStateBlockBodyErr.EXPECT().GetBlockBody(testAncestorHash).Return(body, nil)
	runtimeMockErr.On("ValidateTransaction", externExt).Return(nil, errTestDummyError)

	//valid case
	runtimeMockOk := new(mocksruntime.Instance)
	mockBlockStateBlockBodyOk := NewMockBlockState(ctrl)
	mockBlockStateBlockBodyOk.EXPECT().HighestCommonAncestor(testPrevHash, testCurrentHash).
		Return(testAncestorHash, nil)
	mockBlockStateBlockBodyOk.EXPECT().SubChain(testAncestorHash, testPrevHash).Return(testSubChain, nil)
	mockBlockStateBlockBodyOk.EXPECT().GetRuntime(nil).Return(runtimeMockOk, nil)
	mockBlockStateBlockBodyOk.EXPECT().GetBlockBody(testCurrentHash).Return(nil, errDummyErr)
	mockBlockStateBlockBodyOk.EXPECT().GetBlockBody(testAncestorHash).Return(body, nil)
	runtimeMockOk.On("ValidateTransaction", externExt).
		Return(testValidity, nil)
	mockTxnStateOk := NewMockTransactionState(ctrl)
	mockTxnStateOk.EXPECT().AddToPool(vtx).Return(common.Hash{})

	type args struct {
		prev common.Hash
		curr common.Hash
	}
	tests := []struct {
		name      string
		service   *Service
		args      args
		expErr    error
		expErrMsg string
	}{
		{
			name: "highest common ancestor err",
			service: &Service{
				blockState: mockBlockStateAncestorErr,
			},
			args: args{
				prev: testPrevHash,
				curr: testCurrentHash,
			},
			expErr:    errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "ancestor eq priv",
			service: &Service{
				blockState: mockBlockStateAncestorEqPriv,
			},
			args: args{
				prev: testPrevHash,
				curr: testCurrentHash,
			},
		},
		{
			name: "subchain err",
			service: &Service{
				blockState: mockBlockStateSubChainErr,
			},
			args: args{
				prev: testPrevHash,
				curr: testCurrentHash,
			},
			expErr:    errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "empty subchain",
			service: &Service{
				blockState: mockBlockStateEmptySubChain,
			},
			args: args{
				prev: testPrevHash,
				curr: testCurrentHash,
			},
		},
		{
			name: "get runtime err",
			service: &Service{
				blockState: mockBlockStateRuntimeErr,
			},
			args: args{
				prev: testPrevHash,
				curr: testCurrentHash,
			},
			expErr:    errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "invalid transaction",
			service: &Service{
				blockState: mockBlockStateBlockBodyErr,
			},
			args: args{
				prev: testPrevHash,
				curr: testCurrentHash,
			},
		},
		{
			name: "happy path",
			service: &Service{
				blockState:       mockBlockStateBlockBodyOk,
				transactionState: mockTxnStateOk,
			},
			args: args{
				prev: testPrevHash,
				curr: testCurrentHash,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			err := s.handleChainReorg(tt.args.prev, tt.args.curr)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}

func TestServiceInsertKey(t *testing.T) {
	keyStore := keystore.GlobalKeystore{
		Babe: keystore.NewBasicKeystore(keystore.BabeName, crypto.Sr25519Type),
	}

	keyring, _ := keystore.NewSr25519Keyring()
	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	type args struct {
		kp           crypto.Keypair
		keystoreType string
	}
	tests := []struct {
		name      string
		service   *Service
		args      args
		expErr    error
		expErrMsg string
	}{
		{
			name: "ok case",
			service: &Service{
				keys: &keyStore,
			},
			args: args{
				kp:           aliceKeypair,
				keystoreType: (string)(keystore.BabeName),
			},
		},
		{
			name: "err case",
			service: &Service{
				keys: &keyStore,
			},
			args: args{
				kp:           aliceKeypair,
				keystoreType: "jimbo",
			},
			expErr:    keystore.ErrInvalidKeystoreName,
			expErrMsg: keystore.ErrInvalidKeystoreName.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			err := s.InsertKey(tt.args.kp, tt.args.keystoreType)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}

func TestServiceHasKey(t *testing.T) {
	keyStore := keystore.GlobalKeystore{
		Babe: keystore.NewBasicKeystore(keystore.BabeName, crypto.Sr25519Type),
	}

	keyring, _ := keystore.NewSr25519Keyring()
	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	type args struct {
		pubKeyStr    string
		keystoreType string
	}
	tests := []struct {
		name      string
		service   *Service
		args      args
		exp       bool
		expErr    error
		expErrMsg string
	}{
		{
			name: "ok case",
			service: &Service{
				keys: &keyStore,
			},
			args: args{
				pubKeyStr:    aliceKeypair.Public().Hex(),
				keystoreType: string(keystore.BabeName),
			},
		},
		{
			name: "err case",
			service: &Service{
				keys: &keyStore,
			},
			args: args{
				pubKeyStr:    aliceKeypair.Public().Hex(),
				keystoreType: "jimbo",
			},
			expErr:    keystore.ErrInvalidKeystoreName,
			expErrMsg: keystore.ErrInvalidKeystoreName.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			res, err := s.HasKey(tt.args.pubKeyStr, tt.args.keystoreType)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestService_DecodeSessionKeys(t *testing.T) {
	testEncKeys := []byte{1, 2, 3, 4}
	ctrl := gomock.NewController(t)
	mockBlockStateErr := NewMockBlockState(ctrl)
	mockBlockStateErr.EXPECT().GetRuntime(nil).Return(nil, errDummyErr)

	runtimeMockOk := new(mocksruntime.Instance)
	runtimeMockOk.On("DecodeSessionKeys", testEncKeys).Return(testEncKeys, nil)
	mockBlockStateOk := NewMockBlockState(ctrl)
	mockBlockStateOk.EXPECT().GetRuntime(nil).Return(runtimeMockOk, nil)

	tests := []struct {
		name      string
		service   *Service
		enc       []byte
		exp       []byte
		expErr    error
		expErrMsg string
	}{
		{
			name: "ok case",
			service: &Service{
				blockState: mockBlockStateOk,
			},
			enc: testEncKeys,
			exp: testEncKeys,
		},
		{
			name: "err case",
			service: &Service{
				blockState: mockBlockStateErr,
			},
			enc:       testEncKeys,
			expErr:    errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			res, err := s.DecodeSessionKeys(tt.enc)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestServiceGetRuntimeVersion(t *testing.T) {
	testAPIItem := runtime.APIItem{
		Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Ver:  99,
	}
	rv := runtime.NewVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		25,
		0,
		[]runtime.APIItem{testAPIItem},
		5,
	)
	emptyTrie := trie.NewEmptyTrie()
	ts, err := rtstorage.NewTrieState(emptyTrie)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockStorageStateGetRootErr := NewMockStorageState(ctrl)
	mockStorageStateGetRootErr.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(nil, errDummyErr)

	mockStorageStateTrieErr := NewMockStorageState(ctrl)
	mockStorageStateTrieErr.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)
	mockStorageStateTrieErr.EXPECT().TrieState(&common.Hash{}).Return(nil, errDummyErr)

	mockStorageStateOk := NewMockStorageState(ctrl)
	mockStorageStateOk.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil).MaxTimes(2)
	mockStorageStateOk.EXPECT().TrieState(&common.Hash{}).Return(ts, nil).MaxTimes(2)

	mockBlockStateErr := NewMockBlockState(ctrl)
	mockBlockStateErr.EXPECT().GetRuntime(&common.Hash{}).Return(nil, errDummyErr)

	runtimeMockOk := new(mocksruntime.Instance)
	mockBlockStateOk := NewMockBlockState(ctrl)
	mockBlockStateOk.EXPECT().GetRuntime(&common.Hash{}).Return(runtimeMockOk, nil)
	runtimeMockOk.On("SetContextStorage", ts)
	runtimeMockOk.On("Version").Return(rv, nil)

	tests := []struct {
		name      string
		service   *Service
		bhash     *common.Hash
		exp       runtime.Version
		expErr    error
		expErrMsg string
	}{
		{
			name: "get state root err",
			service: &Service{storageState: mockStorageStateGetRootErr},
			bhash: &common.Hash{},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "trie state err",
			service: &Service{storageState: mockStorageStateTrieErr},
			bhash: &common.Hash{},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "get runtime err",
			service: &Service{
				storageState: mockStorageStateOk,
				blockState: mockBlockStateErr,
			},
			bhash: &common.Hash{},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "happy path",
			service: &Service{
				storageState: mockStorageStateOk,
				blockState: mockBlockStateOk,
			},
			bhash: &common.Hash{},
			exp: rv,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			res, err := s.GetRuntimeVersion(tt.bhash)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestServiceHandleSubmittedExtrinsic(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorageStateErr := NewMockStorageState(ctrl)
	mockStorageStateErr.EXPECT().TrieState(nil).Return(nil, errDummyErr)

	mockStorageStateOk := NewMockStorageState(ctrl)
	mockBlockStateRuntimeErr := NewMockBlockState(ctrl)
	mockStorageStateOk.EXPECT().TrieState(nil).Return(&rtstorage.TrieState{}, nil).MaxTimes(3)
	mockBlockStateRuntimeErr.EXPECT().GetRuntime(nil).Return(nil, errDummyErr)

	ext := types.Extrinsic{}
	externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, ext...))
	runtimeMockErr := new(mocksruntime.Instance)
	mockBlockStateRuntimeOk := NewMockBlockState(ctrl)
	mockBlockStateRuntimeOk.EXPECT().GetRuntime(nil).Return(runtimeMockErr, nil).MaxTimes(2)
	runtimeMockErr.On("SetContextStorage", &rtstorage.TrieState{})
	runtimeMockErr.On("ValidateTransaction", externalExt).Return(nil, errDummyErr)

	runtimeMockOk := new(mocksruntime.Instance)
	mockBlockStateRuntimeOk2 := NewMockBlockState(ctrl)
	mockBlockStateRuntimeOk2.EXPECT().GetRuntime(nil).Return(runtimeMockOk, nil).MaxTimes(2)
	runtimeMockOk.On("SetContextStorage", &rtstorage.TrieState{})
	runtimeMockOk.On("ValidateTransaction", externalExt).
		Return(&transaction.Validity{Propagate: true}, nil)

	mockTxnState := NewMockTransactionState(ctrl)
	mockTxnState.EXPECT().AddToPool(transaction.NewValidTransaction(ext, &transaction.Validity{Propagate: true}))

	mockNetState := NewMockNetwork(ctrl)
	mockNetState.EXPECT().GossipMessage(&network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}})
	tests := []struct {
		name    string
		service   *Service
		ext types.Extrinsic
		expErr error
		expErrMsg string
	}{
		{
			name: "nil network",
			service: &Service{},
		},
		{
			name: "trie state err",
			service: &Service{
				storageState: mockStorageStateErr,
				net: NewMockNetwork(ctrl),
			},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "get runtime err",
			service: &Service{
				storageState: mockStorageStateOk,
				blockState: mockBlockStateRuntimeErr,
				net: NewMockNetwork(ctrl),
			},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "validate txn err",
			service: &Service{
				storageState: mockStorageStateOk,
				blockState: mockBlockStateRuntimeOk,
				net: NewMockNetwork(ctrl),
			},
			ext: types.Extrinsic{},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "happy path",
			service: &Service{
				storageState: mockStorageStateOk,
				blockState: mockBlockStateRuntimeOk2,
				transactionState: mockTxnState,
				net: mockNetState,
			},
			ext: types.Extrinsic{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			err := s.HandleSubmittedExtrinsic(tt.ext)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}

func TestServiceGetMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorageStateRootErr := NewMockStorageState(ctrl)
	mockStorageStateRootErr.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(nil, errDummyErr)

	mockStorageStateTrieErr := NewMockStorageState(ctrl)
	mockStorageStateTrieErr.EXPECT().TrieState(nil).Return(nil, errDummyErr)

	mockStorageStateOk := NewMockStorageState(ctrl)
	mockStorageStateOk.EXPECT().TrieState(nil).Return(&rtstorage.TrieState{}, nil).MaxTimes(2)
	mockBlockStateRuntimeErr := NewMockBlockState(ctrl)
	mockBlockStateRuntimeErr.EXPECT().GetRuntime(nil).Return(nil, errDummyErr)

	runtimeMockOk := new(mocksruntime.Instance)
	mockBlockStateRuntimeOk := NewMockBlockState(ctrl)
	mockBlockStateRuntimeOk.EXPECT().GetRuntime(nil).Return(runtimeMockOk, nil)
	runtimeMockOk.On("SetContextStorage", &rtstorage.TrieState{})
	runtimeMockOk.On("Metadata").Return([]byte{1, 2, 3}, nil)
	tests := []struct {
		name    string
		service   *Service
		bhash *common.Hash
		exp    []byte
		expErr error
		expErrMsg string
	}{
		{
			name: "get state root error",
			service: &Service{
				storageState: mockStorageStateRootErr,
			},
			bhash: &common.Hash{},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "trie state error",
			service: &Service{
				storageState: mockStorageStateTrieErr,
			},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "get runtime error",
			service: &Service{
				storageState: mockStorageStateOk,
				blockState: mockBlockStateRuntimeErr,
			},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "happy path",
			service: &Service{
				storageState: mockStorageStateOk,
				blockState: mockBlockStateRuntimeOk,
			},
			exp: []byte{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			res, err := s.GetMetadata(tt.bhash)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestService_tryQueryStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStorageStateRootErr := NewMockStorageState(ctrl)
	mockStorageStateRootErr.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(nil, errDummyErr)

	mockStorageStateErr := NewMockStorageState(ctrl)
	mockStorageStateErr.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)
	mockStorageStateErr.EXPECT().GetStorage(&common.Hash{}, common.MustHexToBytes("0x01")).Return(nil, errDummyErr)

	mockStorageStateOk := NewMockStorageState(ctrl)
	mockStorageStateOk.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)
	mockStorageStateOk.EXPECT().GetStorage(&common.Hash{}, common.MustHexToBytes("0x01")).
		Return([]byte{1, 2, 3}, nil)
	expChanges := make(QueryKeyValueChanges)
	expChanges["0x01"] = common.BytesToHex([]byte{1, 2, 3})
	type args struct {
		block common.Hash
		keys  []string
	}
	tests := []struct {
		name    string
		service *Service
		args    args
		exp    QueryKeyValueChanges
		expErr error
		expErrMsg string
	}{
		{
			name: "get state root error",
			service: &Service{storageState: mockStorageStateRootErr},
			args: args{block: common.Hash{}},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "get storage error",
			service: &Service{storageState: mockStorageStateErr},
			args: args{
				block: common.Hash{},
				keys: []string{"0x01"},
			},
			expErr: errDummyErr,
			expErrMsg: errDummyErr.Error(),
		},
		{
			name: "happy path",
			service: &Service{storageState: mockStorageStateOk},
			args: args{
				block: common.Hash{},
				keys: []string{"0x01"},
			},
			exp: expChanges,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.service
			res, err := s.tryQueryStorage(tt.args.block, tt.args.keys...)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

// This needs to be last function in this file
func TestCleanup(t *testing.T) {
	err := runtime.RemoveFiles(testWasmPaths)
	require.NoError(t, err)
}
