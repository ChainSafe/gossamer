// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package core

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	inmemory_storage "github.com/ChainSafe/gossamer/lib/runtime/storage/inmemory"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStartService(t *testing.T) {
	s := NewTestService(t, nil)

	err := s.Start()
	require.NoError(t, err)

	err = s.Stop()
	require.NoError(t, err)
}

func TestAnnounceBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	net := NewMockNetwork(ctrl)

	cfg := &Config{
		Network: net,
	}

	s := NewTestService(t, cfg)
	err := s.Start()
	require.NoError(t, err)
	defer s.Stop()

	// simulate block sent from BABE session
	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	// Used to define the state root of new block for testing
	genesisHash := s.blockState.(*state.BlockState).GenesisHash()
	genesisBlock, err := s.blockState.(*state.BlockState).GetBlockByHash(genesisHash)
	require.NoError(t, err)

	newBlock := types.Block{
		Header: types.Header{
			Number:     1,
			ParentHash: s.blockState.BestBlockHash(),
			Digest:     digest,
			StateRoot:  genesisBlock.Header.StateRoot,
		},
		Body: *types.NewBody([]types.Extrinsic{}),
	}

	onBlockImportHandleMock := NewMockBlockImportDigestHandler(ctrl)
	onBlockImportHandleMock.EXPECT().HandleDigests(&newBlock.Header).Return(nil)
	mockGrandpaState := NewMockGrandpaState(ctrl)
	mockGrandpaState.EXPECT().ApplyForcedChanges(&newBlock.Header).Return(nil)
	s.onBlockImport = onBlockImportHandleMock
	s.grandpaState = mockGrandpaState

	expected := &network.BlockAnnounceMessage{
		ParentHash:     newBlock.Header.ParentHash,
		Number:         newBlock.Header.Number,
		StateRoot:      newBlock.Header.StateRoot,
		ExtrinsicsRoot: newBlock.Header.ExtrinsicsRoot,
		Digest:         digest,
		BestBlock:      true,
	}

	net.EXPECT().GossipMessage(expected)

	state, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	err = s.HandleBlockProduced(&newBlock, state)
	require.NoError(t, err)

	time.Sleep(time.Second)
}

func TestService_InsertKey(t *testing.T) {
	t.Parallel()
	ks := keystore.NewGlobalKeystore()

	cfg := &Config{
		Keystore: ks,
	}
	s := NewTestService(t, cfg)

	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	testCases := []struct {
		description  string
		keystoreType string
		err          error
	}{
		{
			description:  "Test that insertKey fails when keystore type is invalid ",
			keystoreType: "some-invalid-type",
			err:          keystore.ErrInvalidKeystoreName,
		},
		{
			description:  "Test that insertKey fails when keystore type is valid but inappropriate",
			keystoreType: "gran",
			err: fmt.Errorf(
				"%v, passed key type: sr25519, acceptable key type: ed25519",
				keystore.ErrKeyTypeNotSupported),
		},
		{
			description:  "Test that insertKey succeeds when keystore type is valid and appropriate ",
			keystoreType: "acco",
			err:          nil,
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.description, func(t *testing.T) {
			t.Parallel()

			err := s.InsertKey(kr.Alice(), c.keystoreType)

			if c.err == nil {
				require.NoError(t, err)
				res, err := s.HasKey(kr.Alice().Public().Hex(), c.keystoreType)
				require.NoError(t, err)
				require.True(t, res)
			} else {
				require.NotNil(t, err)
				require.Equal(t, err.Error(), c.err.Error())
			}
		})
	}
}

func TestService_HasKey(t *testing.T) {
	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Acco.Insert(kr.Alice())

	cfg := &Config{
		Keystore: ks,
	}
	s := NewTestService(t, cfg)

	res, err := s.HasKey(kr.Alice().Public().Hex(), "acco")
	require.NoError(t, err)
	require.True(t, res)

	res, err = s.HasKey(kr.Alice().Public().Hex(), "babe")
	require.NoError(t, err)
	require.False(t, res)

	res, err = s.HasKey(kr.Alice().Public().Hex(), "gran")
	require.NoError(t, err)
	require.False(t, res)
}

func TestService_HasKey_UnknownType(t *testing.T) {
	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Acco.Insert(kr.Alice())

	cfg := &Config{
		Keystore: ks,
	}

	s := NewTestService(t, cfg)
	res, err := s.HasKey(kr.Alice().Public().Hex(), "xxxx")
	require.EqualError(t, err, "invalid keystore name")
	require.False(t, res)
}

func TestHandleChainReorg_NoReorg(t *testing.T) {
	s := NewTestService(t, nil)
	state.AddBlocksToState(t, s.blockState.(*state.BlockState), 4, false)

	head, err := s.blockState.BestBlockHeader()
	require.NoError(t, err)

	err = s.handleChainReorg(head.ParentHash, head.Hash())
	require.NoError(t, err)
}

func TestHandleChainReorg_WithReorg_Trans(t *testing.T) {
	t.Skip() // TODO: tx fails to validate in handleChainReorg() with "Invalid transaction" (#1026)
	s := NewTestService(t, nil)
	bs := s.blockState

	parent, err := bs.BestBlockHeader()
	require.NoError(t, err)

	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	block1 := sync.BuildBlock(t, rt, parent, nil)
	bs.StoreRuntime(block1.Header.Hash(), rt)
	err = bs.AddBlock(block1)
	require.NoError(t, err)

	block2 := sync.BuildBlock(t, rt, &block1.Header, nil)
	bs.StoreRuntime(block2.Header.Hash(), rt)
	err = bs.AddBlock(block2)
	require.NoError(t, err)

	block3 := sync.BuildBlock(t, rt, &block2.Header, nil)
	bs.StoreRuntime(block3.Header.Hash(), rt)
	err = bs.AddBlock(block3)
	require.NoError(t, err)

	block4 := sync.BuildBlock(t, rt, &block3.Header, nil)
	bs.StoreRuntime(block4.Header.Hash(), rt)
	err = bs.AddBlock(block4)
	require.NoError(t, err)

	block5 := sync.BuildBlock(t, rt, &block4.Header, nil)
	bs.StoreRuntime(block5.Header.Hash(), rt)
	err = bs.AddBlock(block5)
	require.NoError(t, err)

	block31 := sync.BuildBlock(t, rt, &block2.Header, nil)
	bs.StoreRuntime(block31.Header.Hash(), rt)
	err = bs.AddBlock(block31)
	require.NoError(t, err)

	nonce := uint64(0)

	// Add extrinsic to block `block41`
	ext := createExtrinsic(t, rt, bs.(*state.BlockState).GenesisHash(), nonce)

	block41 := sync.BuildBlock(t, rt, &block31.Header, ext)
	bs.StoreRuntime(block41.Header.Hash(), rt)
	err = bs.AddBlock(block41)
	require.NoError(t, err)

	err = s.handleChainReorg(block41.Header.Hash(), block5.Header.Hash())
	require.NoError(t, err)

	pending := s.transactionState.(*state.TransactionState).Pending()
	require.Equal(t, 1, len(pending))
}

func TestHandleChainReorg_WithReorg_NoTransactions(t *testing.T) {
	s := NewTestService(t, nil)
	const height = 5
	const branch = 3
	branches := map[uint]int{branch: 1}
	state.AddBlocksToStateWithFixedBranches(t, s.blockState.(*state.BlockState), height, branches)

	leaves := s.blockState.(*state.BlockState).Leaves()
	require.Equal(t, 2, len(leaves))

	head := s.blockState.BestBlockHash()
	var other common.Hash
	if leaves[0] == head {
		other = leaves[1]
	} else {
		other = leaves[0]
	}

	err := s.handleChainReorg(other, head)
	require.NoError(t, err)
}

func TestHandleChainReorg_WithReorg_Transactions(t *testing.T) {
	t.Skip() // need to update this test to use a valid transaction

	cfg := &Config{
		Runtime: wazero_runtime.NewTestInstance(t, runtime.WESTEND_RUNTIME_v0929),
	}

	s := NewTestService(t, cfg)
	const height = 5
	const branch = 3
	state.AddBlocksToState(t, s.blockState.(*state.BlockState), height, false)

	// create extrinsic
	enc, err := scale.Marshal([]byte("nootwashere"))
	require.NoError(t, err)
	// we prefix with []byte{2} here since that's the enum index for the old IncludeDataExt extrinsic
	tx := append([]byte{2}, enc...)

	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	validity, err := rt.ValidateTransaction(tx)
	require.NoError(t, err)

	// get common ancestor
	ancestor, err := s.blockState.(*state.BlockState).GetBlockByNumber(branch - 1)
	require.NoError(t, err)

	// build "re-org" chain

	digest := types.NewDigest()
	block := &types.Block{
		Header: types.Header{
			ParentHash: ancestor.Header.Hash(),
			Number:     ancestor.Header.Number + 1,
			Digest:     digest,
		},
		Body: types.Body([]types.Extrinsic{tx}),
	}

	s.blockState.StoreRuntime(block.Header.Hash(), rt)
	err = s.blockState.AddBlock(block)
	require.NoError(t, err)

	leaves := s.blockState.(*state.BlockState).Leaves()
	require.Equal(t, 2, len(leaves))

	head := s.blockState.BestBlockHash()
	var other common.Hash
	if leaves[0] == head {
		other = leaves[1]
	} else {
		other = leaves[0]
	}

	err = s.handleChainReorg(other, head)
	require.NoError(t, err)

	pending := s.transactionState.(*state.TransactionState).Pending()
	require.Equal(t, 1, len(pending))
	require.Equal(t, transaction.NewValidTransaction(tx, validity), pending[0])
}

func TestMaintainTransactionPoolLatestTxnQueue_EmptyBlock(t *testing.T) {
	accountInfo := types.AccountInfo{
		Nonce: 0,
		Data: types.AccountData{
			Free:       scale.MustNewUint128(big.NewInt(1152921504606846976)),
			Reserved:   scale.MustNewUint128(big.NewInt(0)),
			MiscFrozen: scale.MustNewUint128(big.NewInt(0)),
			FreeFrozen: scale.MustNewUint128(big.NewInt(0)),
		},
	}
	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	alicePub := common.MustHexToBytes(keyring.Alice().Public().Hex())
	genesisFilePath := utils.GetWestendDevRawGenesisPath(t)

	ctrl := gomock.NewController(t)
	service, encExt := createTestService(t, genesisFilePath, alicePub, accountInfo, ctrl)

	tx := &transaction.ValidTransaction{
		Extrinsic: types.Extrinsic(encExt),
		Validity:  &transaction.Validity{Priority: 1},
	}
	_ = service.transactionState.AddToPool(tx)

	// provides is a list of transaction hashes that depend on this tx, see:
	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/sr-primitives/src/transaction_validity.rs#L195
	provides := common.MustHexToBytes("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d00000000")
	txnValidity := &transaction.Validity{
		Priority:  36074,
		Provides:  [][]byte{provides},
		Longevity: 18446744073709551614,
		Propagate: true,
	}

	expectedTx := transaction.NewValidTransaction(tx.Extrinsic, txnValidity)

	bestBlockHash := service.blockState.BestBlockHash()
	err = service.maintainTransactionPool(&types.Block{
		Body: *types.NewBody([]types.Extrinsic{}),
	}, bestBlockHash)
	require.NoError(t, err)

	resultTx := service.transactionState.(*state.TransactionState).Pop()
	require.Equal(t, expectedTx, resultTx)

	service.transactionState.RemoveExtrinsic(tx.Extrinsic)
	head := service.transactionState.(*state.TransactionState).Pop()
	require.Nil(t, head)
}

func TestMaintainTransactionPoolLatestTxnQueue_BlockWithExtrinsics(t *testing.T) {
	accountInfo := types.AccountInfo{
		Nonce: 0,
		Data: types.AccountData{
			Free:       scale.MustNewUint128(big.NewInt(1152921504606846976)),
			Reserved:   scale.MustNewUint128(big.NewInt(0)),
			MiscFrozen: scale.MustNewUint128(big.NewInt(0)),
			FreeFrozen: scale.MustNewUint128(big.NewInt(0)),
		},
	}
	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	alicePub := common.MustHexToBytes(keyring.Alice().Public().Hex())
	genesisFilePath := utils.GetWestendDevRawGenesisPath(t)

	ctrl := gomock.NewController(t)
	service, encodedExtrinsic := createTestService(t, genesisFilePath, alicePub, accountInfo, ctrl)

	tx := &transaction.ValidTransaction{
		Extrinsic: types.Extrinsic(encodedExtrinsic),
		Validity:  &transaction.Validity{Priority: 1},
	}
	_ = service.transactionState.AddToPool(tx)

	bestBlockHash := service.blockState.BestBlockHash()
	err = service.maintainTransactionPool(&types.Block{
		Body: types.Body([]types.Extrinsic{encodedExtrinsic}),
	}, bestBlockHash)
	require.NoError(t, err)

	res := []*transaction.ValidTransaction{}
	for {
		tx := service.transactionState.(*state.TransactionState).Pop()
		if tx == nil {
			break
		}
		res = append(res, tx)
	}
	require.Empty(t, res)
}

func TestService_GetRuntimeVersion(t *testing.T) {
	s := NewTestService(t, nil)
	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	rtExpected, err := rt.Version()
	require.NoError(t, err)

	rtv, err := s.GetRuntimeVersion(nil)
	require.NoError(t, err)
	require.Equal(t, rtExpected, rtv)
}

func TestService_HandleSubmittedExtrinsic(t *testing.T) {
	cfg := &Config{}
	ctrl := gomock.NewController(t)

	net := NewMockNetwork(ctrl)
	net.EXPECT().GossipMessage(gomock.AssignableToTypeOf(new(network.TransactionMessage)))
	cfg.Network = net
	s := NewTestService(t, cfg)

	genHeader, err := s.blockState.BestBlockHeader()
	require.NoError(t, err)

	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	ts, err := s.storageState.TrieState(nil)
	require.NoError(t, err)
	rt.SetContextStorage(ts)

	babeConfig, err := rt.BabeConfiguration()
	require.NoError(t, err)

	currentTimestamp := uint64(time.Now().UnixMilli())
	currentSlotNumber := currentTimestamp / babeConfig.SlotDuration

	block := buildTestBlockWithoutExtrinsics(t, rt, genHeader, currentSlotNumber, currentTimestamp)

	onBlockImportHandlerMock := NewMockBlockImportDigestHandler(ctrl)
	onBlockImportHandlerMock.EXPECT().HandleDigests(&block.Header).Return(nil)
	mockGrandpaState := NewMockGrandpaState(ctrl)
	mockGrandpaState.EXPECT().ApplyForcedChanges(&block.Header).Return(nil)
	s.onBlockImport = onBlockImportHandlerMock
	s.grandpaState = mockGrandpaState

	err = s.handleBlock(block, ts)
	require.NoError(t, err)

	extBytes := createExtrinsic(t, rt, genHeader.Hash(), 0)
	err = s.HandleSubmittedExtrinsic(extBytes)
	require.NoError(t, err)
}

func TestService_GetMetadata(t *testing.T) {
	s := NewTestService(t, nil)
	res, err := s.GetMetadata(nil)
	require.NoError(t, err)
	require.Greater(t, len(res), 10000)
}

func TestService_HandleRuntimeChanges(t *testing.T) {
	s := NewTestService(t, nil)
	genesisHeader, err := s.blockState.BestBlockHeader()
	require.NoError(t, err)

	genesisBlockHash := genesisHeader.Hash()
	genesisStateRoot := genesisHeader.StateRoot

	ts, err := s.storageState.TrieState(&genesisStateRoot) // Pass genesis root
	require.NoError(t, err)

	firstBlockHash := createBlockUsingOldRuntime(t, genesisBlockHash, ts, s.blockState)
	updateNodeRuntimeWasmPath, err := runtime.GetRuntime(context.Background(), runtime.WESTEND_RUNTIME_v0929)
	require.NoError(t, err)

	secondBlockHash := createBlockUsingNewRuntime(t, genesisBlockHash, updateNodeRuntimeWasmPath, ts, s.blockState)

	// firstBlockHash runtime should not be updated
	genesisRuntime, err := s.blockState.GetRuntime(genesisBlockHash)
	require.NoError(t, err)

	firstBlockRuntime, err := s.blockState.GetRuntime(firstBlockHash)
	require.NoError(t, err)

	genesisRuntimeVersion, err := genesisRuntime.Version()
	require.NoError(t, err)

	firstBlockRuntimeVersion, err := firstBlockRuntime.Version()
	require.NoError(t, err)

	require.Equal(t, genesisRuntimeVersion, firstBlockRuntimeVersion)

	secondBlockRuntime, err := s.blockState.GetRuntime(secondBlockHash)
	require.NoError(t, err)

	const updatedSpecVersion = uint32(9290)
	secondBlockRuntimeVersion, err := secondBlockRuntime.Version()
	require.NoError(t, err)

	require.Equal(t, updatedSpecVersion, secondBlockRuntimeVersion.SpecVersion)
}

func createBlockUsingOldRuntime(t *testing.T, bestBlockHash common.Hash, trieState *inmemory_storage.InMemoryTrieState,
	blockState BlockState) (blockHash common.Hash) {
	parentRt, err := blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	primaryDigestData := types.NewBabePrimaryPreDigest(0, uint64(0), [32]byte{}, [64]byte{})
	digest := types.NewDigest()
	preRuntimeDigest, err := primaryDigestData.ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*preRuntimeDigest)
	require.NoError(t, err)

	newBlock := &types.Block{
		Header: types.Header{
			ParentHash: bestBlockHash,
			Number:     1,
			Digest:     digest,
		},
		Body: *types.NewBody([]types.Extrinsic{[]byte("Old Runtime")}),
	}
	err = blockState.AddBlock(newBlock)
	require.NoError(t, err)

	newBlockHash := newBlock.Header.Hash()
	err = blockState.HandleRuntimeChanges(trieState, parentRt, newBlockHash)
	require.NoError(t, err)

	return newBlockHash
}

func createBlockUsingNewRuntime(t *testing.T, bestBlockHash common.Hash, newRuntimePath string,
	trieState *inmemory_storage.InMemoryTrieState, blockState BlockState) (blockHash common.Hash) {
	parentRt, err := blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	testRuntime, err := os.ReadFile(newRuntimePath)
	require.NoError(t, err)

	trieState.Put(common.CodeKey, testRuntime)

	primaryDigestData := types.NewBabePrimaryPreDigest(0, uint64(1), [32]byte{}, [64]byte{})
	digest := types.NewDigest()
	preRuntimeDigest, err := primaryDigestData.ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*preRuntimeDigest)
	require.NoError(t, err)

	newBlockRuntimeUpdate := &types.Block{
		Header: types.Header{
			ParentHash: bestBlockHash,
			Number:     1,
			Digest:     digest,
		},
		Body: *types.NewBody([]types.Extrinsic{[]byte("Updated Runtime")}),
	}

	err = blockState.AddBlock(newBlockRuntimeUpdate)
	require.NoError(t, err)

	newBlockRTUpdateHash := newBlockRuntimeUpdate.Header.Hash()
	err = blockState.HandleRuntimeChanges(trieState, parentRt, newBlockRTUpdateHash)
	require.NoError(t, err)

	return newBlockRTUpdateHash
}

func TestService_HandleCodeSubstitutes(t *testing.T) {
	s := NewTestService(t, nil)

	runtimeFilepath, err := runtime.GetRuntime(context.Background(), runtime.POLKADOT_RUNTIME_v0929)
	require.NoError(t, err)
	testRuntime, err := os.ReadFile(runtimeFilepath)
	require.NoError(t, err)

	// hash for known test code substitution
	blockHash := common.MustHexToHash("0x86aa36a140dfc449c30dbce16ce0fea33d5c3786766baa764e33f336841b9e29")
	s.codeSubstitute = map[common.Hash]string{
		blockHash: common.BytesToHex(testRuntime),
	}

	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	s.blockState.StoreRuntime(blockHash, rt)

	ts := inmemory_storage.NewTrieState(trie.NewEmptyInmemoryTrie())
	err = s.handleCodeSubstitution(blockHash, ts)
	require.NoError(t, err)
	codSub := s.codeSubstitutedState.(*state.BaseState).LoadCodeSubstitutedBlockHash()
	require.Equal(t, blockHash, codSub)
}

func TestService_HandleRuntimeChangesAfterCodeSubstitutes(t *testing.T) {
	s := NewTestService(t, nil)

	bestBlockHash := s.blockState.BestBlockHash()
	parentRt, err := s.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	codeHashBefore := parentRt.GetCodeHash()
	// hash for known test code substitution
	blockHash := common.MustHexToHash("0x86aa36a140dfc449c30dbce16ce0fea33d5c3786766baa764e33f336841b9e29")

	body := types.NewBody([]types.Extrinsic{[]byte("Updated Runtime")})
	newBlock := &types.Block{
		Header: types.Header{
			ParentHash: blockHash,
			Number:     1,
			Digest:     types.NewDigest(),
		},
		Body: *body,
	}

	ts := inmemory_storage.NewTrieState(trie.NewEmptyInmemoryTrie())
	err = s.handleCodeSubstitution(blockHash, ts)
	require.NoError(t, err)
	require.Equal(t, codeHashBefore, parentRt.GetCodeHash()) // codeHash should remain unchanged after code substitute

	runtimeFilepath, err := runtime.GetRuntime(context.Background(), runtime.POLKADOT_RUNTIME_v0929)
	require.NoError(t, err)
	testRuntime, err := os.ReadFile(runtimeFilepath)
	require.NoError(t, err)

	ts, err = s.storageState.TrieState(nil)
	require.NoError(t, err)

	ts.Put(common.CodeKey, testRuntime)
	rtUpdateBhash := newBlock.Header.Hash()

	// update runtime for new block
	err = s.blockState.HandleRuntimeChanges(ts, parentRt, rtUpdateBhash)
	require.NoError(t, err)

	rt, err := s.blockState.GetRuntime(rtUpdateBhash)
	require.NoError(t, err)

	// codeHash should change after runtime change
	require.NotEqualf(t,
		codeHashBefore,
		rt.GetCodeHash(),
		"expected different code hash after runtime update")
}

func buildTestBlockWithoutExtrinsics(t *testing.T, instance runtime.Instance,
	parentHeader *types.Header, slotNumber, timestamp uint64) *types.Block {
	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, slotNumber).ToPreRuntimeDigest()
	require.NoError(t, err)

	err = digest.Add(*prd)
	require.NoError(t, err)
	header := &types.Header{
		ParentHash: parentHeader.Hash(),
		Number:     parentHeader.Number + 1,
		Digest:     digest,
	}

	err = instance.InitializeBlock(header)
	require.NoError(t, err)

	inherentData := types.NewInherentData()
	err = inherentData.SetInherent(types.Timstap0, timestamp)
	require.NoError(t, err)

	err = inherentData.SetInherent(types.Babeslot, uint64(1))
	require.NoError(t, err)

	parachainInherent := inherents.ParachainInherentData{
		ParentHeader: *parentHeader,
	}

	err = inherentData.SetInherent(types.Parachn0, parachainInherent)
	require.NoError(t, err)

	err = inherentData.SetInherent(types.Newheads, []byte{0})
	require.NoError(t, err)

	encodedInherents, err := inherentData.Encode()
	require.NoError(t, err)

	inherentExts, err := instance.InherentExtrinsics(encodedInherents)
	require.NoError(t, err)

	var decodedInherents [][]byte
	err = scale.Unmarshal(inherentExts, &decodedInherents)
	require.NoError(t, err)

	for _, inherent := range decodedInherents {
		encoded, err := scale.Marshal(inherent)
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(encoded)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}

	finalisedHeader, err := instance.FinalizeBlock()
	require.NoError(t, err)

	finalisedHeader.Number = header.Number
	finalisedHeader.Hash()
	return &types.Block{
		Header: *finalisedHeader,
		Body:   types.Body(types.BytesArrayToExtrinsics(decodedInherents)),
	}
}
