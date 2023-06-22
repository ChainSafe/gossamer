// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	testdata "github.com/ChainSafe/gossamer/dot/rpc/modules/test_data"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	cscale "github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestDummyError = errors.New("test dummy error")

const (
	authoringVersion   = 0
	specVersion        = 25
	implVersion        = 0
	transactionVersion = 0
	stateVersion       = 0
)

func generateTestCentrifugeMetadata(t *testing.T) *ctypes.Metadata {
	t.Helper()
	metadataHex := testdata.NewTestMetadata()
	rawMeta, err := common.HexToBytes(metadataHex)
	require.NoError(t, err)
	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = codec.Decode(decoded, meta)
	require.NoError(t, err)
	return meta
}

func generateExtrinsic(t *testing.T) (extrinsic, externalExtrinsic types.Extrinsic, body *types.Body) {
	t.Helper()
	meta := generateTestCentrifugeMetadata(t)

	rv := runtime.Version{
		SpecName:         []byte("polkadot"),
		ImplName:         []byte("parity-polkadot"),
		AuthoringVersion: authoringVersion,
		SpecVersion:      specVersion,
		ImplVersion:      implVersion,
		APIItems: []runtime.APIItem{{
			Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Ver:  99,
		}},
		TransactionVersion: transactionVersion,
	}

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	bobPub := keyring.Bob().Public().Hex()

	bob, err := ctypes.NewMultiAddressFromHexAccountID(bobPub)
	require.NoError(t, err)

	const balanceTransfer = "Balances.transfer"
	call, err := ctypes.NewCall(meta, balanceTransfer, bob, ctypes.NewUCompactFromUInt(12345))
	require.NoError(t, err)

	// Create the extrinsic
	centrifugeExtrinsic := ctypes.NewExtrinsic(call)
	testGenHash := ctypes.NewHash(common.Hash{}.ToBytes())
	require.NoError(t, err)
	o := ctypes.SignatureOptions{
		BlockHash:          testGenHash,
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        testGenHash,
		Nonce:              ctypes.NewUCompactFromUInt(uint64(0)),
		SpecVersion:        ctypes.U32(rv.SpecVersion),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion),
	}

	// Sign the transaction using Alice's default account
	err = centrifugeExtrinsic.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	// Encode the signed extrinsic
	extEnc := bytes.Buffer{}
	encoder := cscale.NewEncoder(&extEnc)
	err = centrifugeExtrinsic.Encode(*encoder)
	require.NoError(t, err)

	encExt := []types.Extrinsic{extEnc.Bytes()}
	testHeader := types.NewEmptyHeader()
	testExternalExt := types.Extrinsic(bytes.Join([][]byte{
		{byte(types.TxnExternal)},
		encExt[0],
		testHeader.StateRoot.ToBytes(),
	}, nil))
	testUnencryptedBody := types.NewBody(encExt)
	return encExt[0], testExternalExt, testUnencryptedBody
}

func Test_Service_StorageRoot(t *testing.T) {
	t.Parallel()

	ts := rtstorage.NewTrieState(nil)

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := tt.service
			if tt.trieStateCall {
				ctrl := gomock.NewController(t)
				mockStorageState := NewMockStorageState(ctrl)
				mockStorageState.EXPECT().TrieState(nil).Return(tt.retTrieState, tt.retErr)
				service.storageState = mockStorageState
			}

			res, err := service.StorageRoot()
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

	errTest := errors.New("test error")
	validRuntimeCode := getWestendDevRuntimeCode(t)

	testCases := map[string]struct {
		serviceBuilder func(ctrl *gomock.Controller) *Service
		blockHash      common.Hash
		trieState      *rtstorage.TrieState
		errWrapped     error
		errMessage     string
	}{
		"non_existent_block_hash_substitute": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				return &Service{
					codeSubstitute: map[common.Hash]string{
						{0x02}: "0x02",
					},
				}
			},
			blockHash: common.Hash{0x01},
		},
		"empty_runtime_code_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				return &Service{
					codeSubstitute: map[common.Hash]string{
						{0x01}: "0x",
					},
				}
			},
			blockHash:  common.Hash{0x01},
			errWrapped: ErrEmptyRuntimeCode,
			errMessage: "new :code is empty: for hash " +
				"0x0100000000000000000000000000000000000000000000000000000000000000",
		},
		"block_state_get_runtime_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetRuntime(common.Hash{0x01}).
					Return(nil, errTest)
				return &Service{
					blockState: blockState,
					codeSubstitute: map[common.Hash]string{
						{0x01}: "0x00",
					},
				}
			},
			blockHash:  common.Hash{0x01},
			errWrapped: errTest,
			errMessage: "getting runtime from block state: test error",
		},
		"instance_creation_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				storedRuntime := NewMockInstance(ctrl)
				storedRuntime.EXPECT().Keystore().Return(nil)
				storedRuntime.EXPECT().NodeStorage().Return(runtime.NodeStorage{})
				storedRuntime.EXPECT().NetworkService().Return(nil)
				storedRuntime.EXPECT().Validator().Return(false)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetRuntime(common.Hash{0x01}).
					Return(storedRuntime, nil)

				return &Service{
					blockState: blockState,
					codeSubstitute: map[common.Hash]string{
						{0x01}: "0x" +
							// compression header
							hex.EncodeToString([]byte{82, 188, 83, 118, 70, 219, 142, 5}) +
							"01", // bad byte
					},
				}
			},
			blockHash:  common.Hash{0x01},
			errMessage: "creating new runtime instance: unexpected EOF",
		},
		"store_code_substitution_block_hash_error": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				storedRuntime := NewMockInstance(ctrl)
				storedRuntime.EXPECT().Keystore().Return(nil)
				storedRuntime.EXPECT().NodeStorage().Return(runtime.NodeStorage{})
				storedRuntime.EXPECT().NetworkService().Return(nil)
				storedRuntime.EXPECT().Validator().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetRuntime(common.Hash{0x01}).
					Return(storedRuntime, nil)

				codeSubstitutedState := NewMockCodeSubstitutedState(ctrl)
				codeSubstitutedState.EXPECT().
					StoreCodeSubstitutedBlockHash(common.Hash{0x01}).
					Return(errTest)

				return &Service{
					blockState: blockState,
					codeSubstitute: map[common.Hash]string{
						{0x01}: common.BytesToHex(validRuntimeCode),
					},
					codeSubstitutedState: codeSubstitutedState,
				}
			},
			blockHash:  common.Hash{0x01},
			errWrapped: errTest,
			errMessage: "storing code substituted block hash: test error",
		},
		"success": {
			serviceBuilder: func(ctrl *gomock.Controller) *Service {
				storedRuntime := NewMockInstance(ctrl)
				storedRuntime.EXPECT().Keystore().Return(nil)
				storedRuntime.EXPECT().NodeStorage().Return(runtime.NodeStorage{})
				storedRuntime.EXPECT().NetworkService().Return(nil)
				storedRuntime.EXPECT().Validator().Return(true)

				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().GetRuntime(common.Hash{0x01}).
					Return(storedRuntime, nil)

				codeSubstitutedState := NewMockCodeSubstitutedState(ctrl)
				codeSubstitutedState.EXPECT().
					StoreCodeSubstitutedBlockHash(common.Hash{0x01}).
					Return(nil)

				blockState.EXPECT().StoreRuntime(common.Hash{0x01},
					gomock.AssignableToTypeOf(&wazero_runtime.Instance{}))

				return &Service{
					blockState: blockState,
					codeSubstitute: map[common.Hash]string{
						{0x01}: common.BytesToHex(validRuntimeCode),
					},
					codeSubstitutedState: codeSubstitutedState,
				}
			},
			blockHash: common.Hash{0x01},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			service := testCase.serviceBuilder(ctrl)

			err := service.handleCodeSubstitution(testCase.blockHash, testCase.trieState)
			if testCase.errWrapped != nil {
				assert.ErrorIs(t, err, testCase.errWrapped)
			}
			if testCase.errMessage != "" {
				assert.ErrorContains(t, err, testCase.errMessage)
			}
		})
	}
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

	t.Run("nil_input", func(t *testing.T) {
		t.Parallel()
		service := &Service{}
		execTest(t, service, nil, nil, ErrNilBlockHandlerParameter)
	})

	t.Run("storeTrie_error", func(t *testing.T) {
		t.Parallel()
		trieState := rtstorage.NewTrieState(nil)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(errTestDummyError)

		service := &Service{storageState: mockStorageState}
		execTest(t, service, &block, trieState, errTestDummyError)
	})

	t.Run("addBlock_quit_error", func(t *testing.T) {
		t.Parallel()
		trieState := rtstorage.NewTrieState(nil)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(errTestDummyError)

		service := &Service{
			storageState: mockStorageState,
			blockState:   mockBlockState,
		}
		execTest(t, service, &block, trieState, errTestDummyError)
	})

	t.Run("addBlock_parent_not_found_error", func(t *testing.T) {
		t.Parallel()
		trieState := rtstorage.NewTrieState(nil)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrParentNotFound)

		service := &Service{
			storageState: mockStorageState,
			blockState:   mockBlockState,
		}
		execTest(t, service, &block, trieState, blocktree.ErrParentNotFound)
	})

	t.Run("addBlock_error_continue", func(t *testing.T) {
		t.Parallel()
		trieState := rtstorage.NewTrieState(nil)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
		mockBlockState.EXPECT().GetRuntime(block.Header.ParentHash).Return(nil, errTestDummyError)

		onBlockImportHandlerMock := NewMockBlockImportDigestHandler(ctrl)
		onBlockImportHandlerMock.EXPECT().Handle(&block.Header).Return(nil)

		service := &Service{
			storageState:  mockStorageState,
			blockState:    mockBlockState,
			onBlockImport: onBlockImportHandlerMock,
		}
		execTest(t, service, &block, trieState, errTestDummyError)
	})

	t.Run("handle_runtime_changes_error", func(t *testing.T) {
		t.Parallel()
		trieState := rtstorage.NewTrieState(nil)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

		ctrl := gomock.NewController(t)
		runtimeMock := NewMockInstance(ctrl)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
		mockBlockState.EXPECT().GetRuntime(block.Header.ParentHash).Return(runtimeMock, nil)
		mockBlockState.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).
			Return(errTestDummyError)

		onBlockImportHandlerMock := NewMockBlockImportDigestHandler(ctrl)
		onBlockImportHandlerMock.EXPECT().Handle(&block.Header).Return(nil)

		service := &Service{
			storageState:  mockStorageState,
			blockState:    mockBlockState,
			onBlockImport: onBlockImportHandlerMock,
		}
		execTest(t, service, &block, trieState, errTestDummyError)
	})

	t.Run("code_substitution_ok", func(t *testing.T) {
		t.Parallel()
		trieState := rtstorage.NewTrieState(nil)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

		ctrl := gomock.NewController(t)
		runtimeMock := NewMockInstance(ctrl)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
		mockBlockState.EXPECT().GetRuntime(block.Header.ParentHash).Return(runtimeMock, nil)
		mockBlockState.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).Return(nil)

		onBlockImportHandlerMock := NewMockBlockImportDigestHandler(ctrl)
		onBlockImportHandlerMock.EXPECT().Handle(&block.Header).Return(nil)
		service := &Service{
			storageState:  mockStorageState,
			blockState:    mockBlockState,
			ctx:           context.Background(),
			onBlockImport: onBlockImportHandlerMock,
		}
		execTest(t, service, &block, trieState, nil)
	})
}

func Test_Service_HandleBlockProduced(t *testing.T) {
	t.Parallel()
	execTest := func(t *testing.T, s *Service, block *types.Block, trieState *rtstorage.TrieState, expErr error) {
		err := s.HandleBlockProduced(block, trieState)
		require.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, "handling block: "+expErr.Error())
		}
	}
	t.Run("nil_input", func(t *testing.T) {
		t.Parallel()
		service := &Service{}
		execTest(t, service, nil, nil, ErrNilBlockHandlerParameter)
	})

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		trieState := rtstorage.NewTrieState(nil)

		digest := types.NewDigest()
		err := digest.Add(
			types.PreRuntimeDigest{
				ConsensusEngineID: types.BabeEngineID,
				Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
			})
		require.NoError(t, err)

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21
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
		runtimeMock := NewMockInstance(ctrl)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().StoreTrie(trieState, &block.Header).Return(nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().AddBlock(&block).Return(blocktree.ErrBlockExists)
		mockBlockState.EXPECT().GetRuntime(block.Header.ParentHash).Return(runtimeMock, nil)
		mockBlockState.EXPECT().HandleRuntimeChanges(trieState, runtimeMock, block.Header.Hash()).Return(nil)
		mockNetwork := NewMockNetwork(ctrl)
		mockNetwork.EXPECT().GossipMessage(msg)
		onBlockImportHandlerMock := NewMockBlockImportDigestHandler(ctrl)
		onBlockImportHandlerMock.EXPECT().Handle(&block.Header).Return(nil)

		service := &Service{
			storageState:  mockStorageState,
			blockState:    mockBlockState,
			net:           mockNetwork,
			ctx:           context.Background(),
			onBlockImport: onBlockImportHandlerMock,
		}
		execTest(t, service, &block, trieState, nil)
	})
}

func Test_Service_maintainTransactionPool(t *testing.T) {
	t.Parallel()
	t.Run("Validate_Transaction_err", func(t *testing.T) {
		t.Parallel()
		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

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
		externalExt := types.Extrinsic(bytes.Join([][]byte{
			{byte(types.TxnExternal)},
			extrinsic,
			testHeader.StateRoot.ToBytes(),
		}, nil))
		vt := transaction.NewValidTransaction(extrinsic, validity)

		ctrl := gomock.NewController(t)
		runtimeMock := NewMockInstance(ctrl)
		runtimeMock.EXPECT().ValidateTransaction(externalExt).Return(nil, errTestDummyError)
		runtimeMock.EXPECT().Version().Return(runtime.Version{
			SpecName:         []byte("polkadot"),
			ImplName:         []byte("parity-polkadot"),
			AuthoringVersion: authoringVersion,
			SpecVersion:      specVersion,
			ImplVersion:      implVersion,
			APIItems: []runtime.APIItem{{
				Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
				Ver:  3,
			}},
			TransactionVersion: transactionVersion,
			StateVersion:       stateVersion,
		}, nil)
		runtimeMock.EXPECT().SetContextStorage(&rtstorage.TrieState{})

		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().RemoveExtrinsic(types.Extrinsic{21}).Times(2)
		mockTxnState.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{vt})
		mockBlockState := NewMockBlockState(ctrl)
		runtimeBlockHashCall := mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockState.EXPECT().GetRuntime(common.Hash{1}).
			Return(runtimeMock, nil).After(runtimeBlockHashCall)
		mockBlockState.EXPECT().BestBlockHash().
			Return(common.Hash{}).After(runtimeBlockHashCall)

		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(&common.Hash{1}).Return(&rtstorage.TrieState{}, nil)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{1}).Return(&common.Hash{1}, nil)
		service := &Service{
			transactionState: mockTxnState,
			blockState:       mockBlockState,
			storageState:     mockStorageState,
		}
		err := service.maintainTransactionPool(&block, common.Hash{1})
		require.NoError(t, err)
	})

	t.Run("Validate_Transaction_ok", func(t *testing.T) {
		t.Parallel()
		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

		validity := &transaction.Validity{
			Priority: 0x3e8,
			Requires: [][]byte{{0xb5, 0x47, 0xb1, 0x90, 0x37, 0x10, 0x7e, 0x1f, 0x79, 0x4c,
				0xa8, 0x69, 0x0, 0xa1, 0xb5, 0x98}},
			Provides: [][]byte{{0xe4, 0x80, 0x7d, 0x1b, 0x67, 0x49, 0x37, 0xbf, 0xc7, 0x89,
				0xbb, 0xdd, 0x88, 0x6a, 0xdd, 0xd6}},
			Longevity: 0x40,
			Propagate: true,
		}

		ext := types.Extrinsic{21}
		externalExt := types.Extrinsic(bytes.Join([][]byte{
			{byte(types.TxnExternal)},
			ext,
			testHeader.StateRoot.ToBytes(),
		}, nil))
		vt := transaction.NewValidTransaction(ext, validity)
		tx := transaction.NewValidTransaction(ext, &transaction.Validity{Propagate: true})

		ctrl := gomock.NewController(t)
		runtimeMock := NewMockInstance(ctrl)
		runtimeMock.EXPECT().ValidateTransaction(externalExt).Return(&transaction.Validity{Propagate: true}, nil)
		runtimeMock.EXPECT().Version().Return(runtime.Version{
			SpecName:         []byte("polkadot"),
			ImplName:         []byte("parity-polkadot"),
			AuthoringVersion: authoringVersion,
			SpecVersion:      specVersion,
			ImplVersion:      implVersion,
			APIItems: []runtime.APIItem{{
				Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
				Ver:  3,
			}},
			TransactionVersion: transactionVersion,
			StateVersion:       stateVersion,
		}, nil)
		runtimeMock.EXPECT().SetContextStorage(&rtstorage.TrieState{})
		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().RemoveExtrinsic(types.Extrinsic{21})
		mockTxnState.EXPECT().PendingInPool().Return([]*transaction.ValidTransaction{vt})
		mockTxnState.EXPECT().Push(tx).Return(common.Hash{}, nil)
		mockTxnState.EXPECT().RemoveExtrinsicFromPool(types.Extrinsic{21})

		mockBlockStateOk := NewMockBlockState(ctrl)
		runtimeBlockHashCall := mockBlockStateOk.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockStateOk.EXPECT().GetRuntime(common.Hash{1}).
			Return(runtimeMock, nil).After(runtimeBlockHashCall)
		mockBlockStateOk.EXPECT().BestBlockHash().
			Return(common.Hash{}).After(runtimeBlockHashCall)

		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(&common.Hash{1}).Return(&rtstorage.TrieState{}, nil)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{1}).Return(&common.Hash{1}, nil)
		service := &Service{
			transactionState: mockTxnState,
			blockState:       mockBlockStateOk,
			storageState:     mockStorageState,
		}
		err := service.maintainTransactionPool(&block, common.Hash{1})
		require.NoError(t, err)
	})
}

func Test_Service_handleBlocksAsync(t *testing.T) {
	t.Parallel()
	t.Run("cancelled_context", func(t *testing.T) {
		t.Parallel()
		blockAddChan := make(chan *types.Block)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		service := &Service{
			blockAddCh: blockAddChan,
			ctx:        ctx,
		}
		service.handleBlocksAsync()
	})

	t.Run("channel_not_ok", func(t *testing.T) {
		t.Parallel()
		blockAddChan := make(chan *types.Block)
		close(blockAddChan)
		service := &Service{
			blockAddCh: blockAddChan,
			ctx:        context.Background(),
		}
		service.handleBlocksAsync()
	})

	t.Run("nil_block", func(t *testing.T) {
		t.Parallel()
		blockAddChan := make(chan *types.Block)
		go func() {
			blockAddChan <- nil
			close(blockAddChan)
		}()
		service := &Service{
			blockAddCh: blockAddChan,
			ctx:        context.Background(),
		}
		service.handleBlocksAsync()
	})

	t.Run("handleChainReorg_error", func(t *testing.T) {
		t.Parallel()

		testHeader := types.NewEmptyHeader()
		block := types.NewBlock(*testHeader, *types.NewBody([]types.Extrinsic{[]byte{21}}))
		block.Header.Number = 21

		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		mockBlockState.EXPECT().LowestCommonAncestor(common.Hash{}, block.Header.Hash()).
			Return(common.Hash{}, errTestDummyError)

		blockAddChan := make(chan *types.Block)
		go func() {
			blockAddChan <- &block
			close(blockAddChan)
		}()
		service := &Service{
			blockState: mockBlockState,
			blockAddCh: blockAddChan,
			ctx:        context.Background(),
		}

		assert.PanicsWithError(t, "failed to re-add transactions to chain upon re-org: test dummy error",
			service.handleBlocksAsync)
	})
}

func TestService_handleChainReorg(t *testing.T) {
	t.Parallel()
	execTest := func(t *testing.T, s *Service, prevHash common.Hash, currHash common.Hash, expErr error) {
		err := s.handleChainReorg(prevHash, currHash)
		if expErr != nil {
			assert.EqualError(t, err, expErr.Error())
		}
	}

	testPrevHash := common.MustHexToHash("0x01")
	testCurrentHash := common.MustHexToHash("0x02")
	testAncestorHash := common.MustHexToHash("0x03")
	testSubChain := []common.Hash{testPrevHash, testCurrentHash, testAncestorHash}

	// A valid extrinsic is needed since it will be validated in handleChainReorg
	ext, externExt, body := generateExtrinsic(t)
	testValidity := &transaction.Validity{Propagate: true}
	vtx := transaction.NewValidTransaction(ext, testValidity)

	t.Run("highest_common_ancestor_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().LowestCommonAncestor(testPrevHash, testCurrentHash).
			Return(common.Hash{}, errDummyErr)

		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, testPrevHash, testCurrentHash, errDummyErr)
	})

	t.Run("highest_common_ancestor_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().LowestCommonAncestor(testPrevHash, testCurrentHash).
			Return(common.Hash{}, errDummyErr)

		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, testPrevHash, testCurrentHash, errDummyErr)
	})

	t.Run("ancestor_eq_priv", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().LowestCommonAncestor(testPrevHash, testCurrentHash).
			Return(testPrevHash, nil)

		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, testPrevHash, testCurrentHash, nil)
	})

	t.Run("subchain_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().LowestCommonAncestor(testPrevHash, testCurrentHash).
			Return(testAncestorHash, nil)
		mockBlockState.EXPECT().RangeInMemory(testAncestorHash, testPrevHash).Return([]common.Hash{}, errDummyErr)

		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, testPrevHash, testCurrentHash, errDummyErr)
	})

	t.Run("empty_subchain", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().LowestCommonAncestor(testPrevHash, testCurrentHash).
			Return(testAncestorHash, nil)
		mockBlockState.EXPECT().RangeInMemory(testAncestorHash, testPrevHash).Return([]common.Hash{}, nil)

		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, testPrevHash, testCurrentHash, nil)
	})

	t.Run("get_runtime_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().LowestCommonAncestor(testPrevHash, testCurrentHash).
			Return(testAncestorHash, nil)
		mockBlockState.EXPECT().RangeInMemory(testAncestorHash, testPrevHash).Return(testSubChain, nil)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockState.EXPECT().GetRuntime(common.Hash{1}).Return(nil, errDummyErr)

		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, testPrevHash, testCurrentHash, errDummyErr)
	})

	t.Run("invalid_transaction", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		runtimeMockErr := NewMockInstance(ctrl)
		runtimeMockErr.EXPECT().ValidateTransaction(externExt).Return(nil, errTestDummyError)
		runtimeMockErr.EXPECT().Version().Return(runtime.Version{
			SpecName:         []byte("polkadot"),
			ImplName:         []byte("parity-polkadot"),
			AuthoringVersion: authoringVersion,
			SpecVersion:      specVersion,
			ImplVersion:      implVersion,
			APIItems: []runtime.APIItem{{
				Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
				Ver:  3,
			}},
			TransactionVersion: transactionVersion,
			StateVersion:       stateVersion,
		}, nil)

		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().LowestCommonAncestor(testPrevHash, testCurrentHash).
			Return(testAncestorHash, nil)
		mockBlockState.EXPECT().RangeInMemory(testAncestorHash, testPrevHash).Return(testSubChain, nil)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockState.EXPECT().GetRuntime(common.Hash{1}).Return(runtimeMockErr, nil)
		mockBlockState.EXPECT().GetBlockBody(testCurrentHash).Return(nil, errDummyErr)
		mockBlockState.EXPECT().GetBlockBody(testAncestorHash).Return(body, nil)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().RemoveExtrinsic(ext)

		service := &Service{
			blockState:       mockBlockState,
			transactionState: mockTxnState,
		}

		execTest(t, service, testPrevHash, testCurrentHash, nil)
	})

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		runtimeMockOk := NewMockInstance(ctrl)
		runtimeMockOk.EXPECT().ValidateTransaction(externExt).Return(testValidity, nil)
		runtimeMockOk.EXPECT().Version().Return(runtime.Version{
			SpecName:         []byte("polkadot"),
			ImplName:         []byte("parity-polkadot"),
			AuthoringVersion: authoringVersion,
			SpecVersion:      specVersion,
			ImplVersion:      implVersion,
			APIItems: []runtime.APIItem{{
				Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
				Ver:  3,
			}},
			TransactionVersion: transactionVersion,
			StateVersion:       stateVersion,
		}, nil)

		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().LowestCommonAncestor(testPrevHash, testCurrentHash).
			Return(testAncestorHash, nil)
		mockBlockState.EXPECT().RangeInMemory(testAncestorHash, testPrevHash).Return(testSubChain, nil)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockState.EXPECT().GetRuntime(common.Hash{1}).Return(runtimeMockOk, nil)
		mockBlockState.EXPECT().GetBlockBody(testCurrentHash).Return(nil, errDummyErr)
		mockBlockState.EXPECT().GetBlockBody(testAncestorHash).Return(body, nil)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		mockTxnStateOk := NewMockTransactionState(ctrl)
		mockTxnStateOk.EXPECT().AddToPool(vtx).Return(common.Hash{})

		service := &Service{
			blockState:       mockBlockState,
			transactionState: mockTxnStateOk,
		}
		execTest(t, service, testPrevHash, testCurrentHash, nil)
	})
}

func TestServiceInsertKey(t *testing.T) {
	t.Parallel()
	keyStore := keystore.GlobalKeystore{
		Babe: keystore.NewBasicKeystore(keystore.BabeName, crypto.Sr25519Type),
	}

	keyring, _ := keystore.NewSr25519Keyring()
	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	type args struct {
		kp           KeyPair
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
			name: "ok_case",
			service: &Service{
				keys: &keyStore,
			},
			args: args{
				kp:           aliceKeypair,
				keystoreType: string(keystore.BabeName),
			},
		},
		{
			name: "err_case",
			service: &Service{
				keys: &keyStore,
			},
			args: args{
				kp:           aliceKeypair,
				keystoreType: "test",
			},
			expErr:    keystore.ErrInvalidKeystoreName,
			expErrMsg: keystore.ErrInvalidKeystoreName.Error(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := tt.service
			err := service.InsertKey(tt.args.kp, tt.args.keystoreType)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
		})
	}
}

func TestServiceHasKey(t *testing.T) {
	t.Parallel()
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
			name: "ok_case",
			service: &Service{
				keys: &keyStore,
			},
			args: args{
				pubKeyStr:    aliceKeypair.Public().Hex(),
				keystoreType: string(keystore.BabeName),
			},
		},
		{
			name: "err_case",
			service: &Service{
				keys: &keyStore,
			},
			args: args{
				pubKeyStr:    aliceKeypair.Public().Hex(),
				keystoreType: "test",
			},
			expErr:    keystore.ErrInvalidKeystoreName,
			expErrMsg: keystore.ErrInvalidKeystoreName.Error(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := tt.service
			res, err := service.HasKey(tt.args.pubKeyStr, tt.args.keystoreType)
			assert.ErrorIs(t, err, tt.expErr)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErrMsg)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestService_DecodeSessionKeys(t *testing.T) {
	t.Parallel()
	testEncKeys := []byte{1, 2, 3, 4}
	execTest := func(t *testing.T, s *Service, enc []byte, exp []byte, expErr error) {
		res, err := s.DecodeSessionKeys(enc)
		assert.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, expErr.Error())
		}
		assert.Equal(t, exp, res)
	}

	t.Run("ok_case", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		runtimeMock := NewMockInstance(ctrl)
		runtimeMock.EXPECT().DecodeSessionKeys(testEncKeys).Return(testEncKeys, nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockState.EXPECT().GetRuntime(common.Hash{1}).Return(runtimeMock, nil)
		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, testEncKeys, testEncKeys, nil)
	})

	t.Run("err_case", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockState.EXPECT().GetRuntime(common.Hash{1}).Return(nil, errDummyErr)
		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, testEncKeys, nil, errDummyErr)
	})
}

func TestServiceGetRuntimeVersion(t *testing.T) {
	t.Parallel()
	rv := runtime.Version{
		SpecName:         []byte("polkadot"),
		ImplName:         []byte("parity-polkadot"),
		AuthoringVersion: authoringVersion,
		SpecVersion:      specVersion,
		ImplVersion:      implVersion,
		APIItems: []runtime.APIItem{{
			Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Ver:  99,
		}},
		TransactionVersion: transactionVersion,
	}
	emptyTrie := trie.NewEmptyTrie()
	ts := rtstorage.NewTrieState(emptyTrie)

	execTest := func(t *testing.T, s *Service, bhash *common.Hash, exp runtime.Version,
		expErr error, expectedErrMessage string) {
		res, err := s.GetRuntimeVersion(bhash)
		assert.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, expectedErrMessage)
		}
		assert.Equal(t, exp, res)
	}

	t.Run("get_state_root_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(nil, errDummyErr)
		service := &Service{
			storageState: mockStorageState,
		}
		const expectedErrMessage = "setting up runtime: getting state root from block hash: dummy error for testing"
		execTest(t, service, &common.Hash{}, runtime.Version{}, errDummyErr, expectedErrMessage)
	})

	t.Run("trie_state_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)
		mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, errDummyErr)
		service := &Service{
			storageState: mockStorageState,
		}
		const expectedErrMessage = "setting up runtime: getting trie state: dummy error for testing"
		execTest(t, service, &common.Hash{}, runtime.Version{}, errDummyErr, expectedErrMessage)
	})

	t.Run("get_runtime_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)
		mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(ts, nil).MaxTimes(2)

		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().GetRuntime(common.Hash{}).Return(nil, errDummyErr)
		service := &Service{
			storageState: mockStorageState,
			blockState:   mockBlockState,
		}
		const expectedErrMessage = "setting up runtime: getting runtime: dummy error for testing"
		execTest(t, service, &common.Hash{}, runtime.Version{}, errDummyErr, expectedErrMessage)
	})

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil).MaxTimes(2)
		mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(ts, nil).MaxTimes(2)

		runtimeMock := NewMockInstance(ctrl)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().GetRuntime(common.Hash{}).Return(runtimeMock, nil)
		runtimeMock.EXPECT().SetContextStorage(ts)
		runtimeMock.EXPECT().Version().Return(rv, nil)
		service := &Service{
			storageState: mockStorageState,
			blockState:   mockBlockState,
		}
		execTest(t, service, &common.Hash{}, rv, nil, "")
	})
}

func TestServiceHandleSubmittedExtrinsic(t *testing.T) {
	t.Parallel()
	ext := types.Extrinsic{}
	testHeader := types.NewEmptyHeader()
	externalExt := types.Extrinsic(bytes.Join([][]byte{
		{byte(types.TxnExternal)},
		ext,
		testHeader.StateRoot.ToBytes(),
	}, nil))
	execTest := func(t *testing.T, s *Service, ext types.Extrinsic, expErr error) {
		err := s.HandleSubmittedExtrinsic(ext)
		assert.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, expErr.Error())
		}
	}

	t.Run("nil_network", func(t *testing.T) {
		t.Parallel()
		service := &Service{}
		execTest(t, service, nil, nil)
	})

	t.Run("trie_state_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(nil, errDummyErr)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)

		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().Exists(nil)
		service := &Service{
			blockState:       mockBlockState,
			storageState:     mockStorageState,
			transactionState: mockTxnState,
			net:              NewMockNetwork(ctrl),
		}
		execTest(t, service, nil, errDummyErr)
	})

	t.Run("get_runtime_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		mockBlockState.EXPECT().GetRuntime(common.Hash{}).Return(nil, errDummyErr)

		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(&rtstorage.TrieState{}, nil)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)

		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().Exists(nil).MaxTimes(2)
		service := &Service{
			storageState:     mockStorageState,
			transactionState: mockTxnState,
			blockState:       mockBlockState,
			net:              NewMockNetwork(ctrl),
		}
		execTest(t, service, nil, errDummyErr)
	})

	t.Run("validate_txn_err", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		runtimeMockErr := NewMockInstance(ctrl)
		mockBlockState.EXPECT().GetRuntime(common.Hash{}).Return(runtimeMockErr, nil).MaxTimes(2)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})

		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(&rtstorage.TrieState{}, nil)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)

		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().Exists(types.Extrinsic{})

		runtimeMockErr.EXPECT().ValidateTransaction(externalExt).Return(nil, errDummyErr)
		runtimeMockErr.EXPECT().Version().Return(runtime.Version{
			SpecName:         []byte("polkadot"),
			ImplName:         []byte("parity-polkadot"),
			AuthoringVersion: authoringVersion,
			SpecVersion:      specVersion,
			ImplVersion:      implVersion,
			APIItems: []runtime.APIItem{{
				Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
				Ver:  3,
			}},
			TransactionVersion: transactionVersion,
			StateVersion:       stateVersion,
		}, nil)
		runtimeMockErr.EXPECT().SetContextStorage(&rtstorage.TrieState{})
		service := &Service{
			storageState:     mockStorageState,
			transactionState: mockTxnState,
			blockState:       mockBlockState,
			net:              NewMockNetwork(ctrl),
		}
		execTest(t, service, types.Extrinsic{}, errDummyErr)
	})

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		runtimeMock := NewMockInstance(ctrl)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})
		mockBlockState.EXPECT().GetRuntime(common.Hash{}).Return(runtimeMock, nil).MaxTimes(2)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{})

		runtimeMock.EXPECT().ValidateTransaction(externalExt).Return(&transaction.Validity{Propagate: true}, nil)
		runtimeMock.EXPECT().Version().Return(runtime.Version{
			SpecName:         []byte("polkadot"),
			ImplName:         []byte("parity-polkadot"),
			AuthoringVersion: authoringVersion,
			SpecVersion:      specVersion,
			ImplVersion:      implVersion,
			APIItems: []runtime.APIItem{{
				Name: common.MustBlake2b8([]byte("TaggedTransactionQueue")),
				Ver:  3,
			}},
			TransactionVersion: transactionVersion,
			StateVersion:       stateVersion,
		}, nil)
		runtimeMock.EXPECT().SetContextStorage(&rtstorage.TrieState{})

		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(&common.Hash{}).Return(&rtstorage.TrieState{}, nil)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(&common.Hash{}, nil)

		mockTxnState := NewMockTransactionState(ctrl)
		mockTxnState.EXPECT().Exists(types.Extrinsic{}).MaxTimes(2)
		mockTxnState.EXPECT().AddToPool(transaction.NewValidTransaction(ext, &transaction.Validity{Propagate: true}))
		mockNetState := NewMockNetwork(ctrl)
		mockNetState.EXPECT().GossipMessage(&network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}})
		service := &Service{
			storageState:     mockStorageState,
			transactionState: mockTxnState,
			blockState:       mockBlockState,
			net:              mockNetState,
		}
		execTest(t, service, types.Extrinsic{}, nil)
	})
}

func TestServiceGetMetadata(t *testing.T) {
	t.Parallel()
	execTest := func(t *testing.T, s *Service, bhash *common.Hash, exp []byte,
		expErr error, expectedErrMessage string) {
		res, err := s.GetMetadata(bhash)
		assert.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, expectedErrMessage)
		}
		assert.Equal(t, exp, res)
	}

	t.Run("get_state_root_error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().GetStateRootFromBlock(&common.Hash{}).Return(nil, errDummyErr)
		service := &Service{
			storageState: mockStorageState,
		}
		const expectedErrMessage = "setting up runtime: getting state root from block hash: dummy error for testing"
		execTest(t, service, &common.Hash{}, nil, errDummyErr, expectedErrMessage)
	})

	t.Run("trie_state_error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(nil).Return(nil, errDummyErr)
		service := &Service{
			storageState: mockStorageState,
		}
		const expectedErrMessage = "setting up runtime: getting trie state: dummy error for testing"
		execTest(t, service, nil, nil, errDummyErr, expectedErrMessage)
	})

	t.Run("get_runtime_error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(nil).Return(&rtstorage.TrieState{}, nil)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockState.EXPECT().GetRuntime(common.Hash{1}).Return(nil, errDummyErr)
		service := &Service{
			storageState: mockStorageState,
			blockState:   mockBlockState,
		}
		const expectedErrMessage = "setting up runtime: getting runtime: dummy error for testing"
		execTest(t, service, nil, nil, errDummyErr, expectedErrMessage)
	})

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().TrieState(nil).Return(&rtstorage.TrieState{}, nil)
		runtimeMockOk := NewMockInstance(ctrl)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{1})
		mockBlockState.EXPECT().GetRuntime(common.Hash{1}).Return(runtimeMockOk, nil)
		runtimeMockOk.EXPECT().SetContextStorage(&rtstorage.TrieState{})
		runtimeMockOk.EXPECT().Metadata().Return([]byte{1, 2, 3}, nil)
		service := &Service{
			storageState: mockStorageState,
			blockState:   mockBlockState,
		}
		const expectedErrMessage = "setting up runtime: getting state root from block hash: dummy error for testing"
		execTest(t, service, nil, []byte{1, 2, 3}, nil, expectedErrMessage)
	})
}

func TestService_GetReadProofAt(t *testing.T) {
	t.Parallel()
	execTest := func(t *testing.T, s *Service, block common.Hash, keys [][]byte,
		expHash common.Hash, expProofForKeys [][]byte, expErr error) {
		resHash, resProofForKeys, err := s.GetReadProofAt(block, keys)
		assert.ErrorIs(t, err, expErr)
		if expErr != nil {
			assert.EqualError(t, err, expErr.Error())
		}
		assert.Equal(t, expHash, resHash)
		assert.Equal(t, expProofForKeys, resProofForKeys)
	}

	t.Run("get_block_state_root_error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{2})
		mockBlockState.EXPECT().GetBlockStateRoot(common.Hash{2}).Return(common.Hash{}, errDummyErr)
		service := &Service{
			blockState: mockBlockState,
		}
		execTest(t, service, common.Hash{}, nil, common.Hash{}, nil, errDummyErr)
	})

	t.Run("generate_trie_proof_error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{2})
		mockBlockState.EXPECT().GetBlockStateRoot(common.Hash{2}).Return(common.Hash{3}, nil)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().GenerateTrieProof(common.Hash{3}, [][]byte{{1}}).
			Return([][]byte{}, errDummyErr)
		service := &Service{
			blockState:   mockBlockState,
			storageState: mockStorageState,
		}
		execTest(t, service, common.Hash{}, [][]byte{{1}}, common.Hash{}, nil, errDummyErr)
	})

	t.Run("happy_path", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mockBlockState := NewMockBlockState(ctrl)
		mockBlockState.EXPECT().BestBlockHash().Return(common.Hash{2})
		mockBlockState.EXPECT().GetBlockStateRoot(common.Hash{2}).Return(common.Hash{3}, nil)
		mockStorageState := NewMockStorageState(ctrl)
		mockStorageState.EXPECT().GenerateTrieProof(common.Hash{3}, [][]byte{{1}}).
			Return([][]byte{{2}}, nil)
		service := &Service{
			blockState:   mockBlockState,
			storageState: mockStorageState,
		}
		execTest(t, service, common.Hash{}, [][]byte{{1}}, common.Hash{2}, [][]byte{{2}}, nil)
	})
}
