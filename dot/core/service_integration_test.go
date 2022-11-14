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
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
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
	genesisBlock, err := s.blockState.GetBlockByHash(s.blockState.GenesisHash())
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
	ext := createExtrinsic(t, rt, bs.GenesisHash(), nonce)

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
		Runtime: wasmer.NewTestInstance(t, runtime.NODE_RUNTIME),
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
	genesisFilePath, err := utils.GetGssmrGenesisRawPath()
	require.NoError(t, err)

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
	genesisFilePath, err := utils.GetGssmrGenesisRawPath()
	require.NoError(t, err)

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

	rtExpected := rt.Version()
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

	block := sync.BuildBlock(t, rt, genHeader, nil)

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
	const (
		updatedSpecVersion        = uint32(262)
		updateNodeRuntimeWasmPath = "../../tests/polkadotjs_test/test/node_runtime.compact.wasm"
	)
	s := NewTestService(t, nil)

	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	v := rt.Version()
	currSpecVersion := v.SpecVersion     // genesis runtime version.
	hash := s.blockState.BestBlockHash() // genesisHash

	digest := types.NewDigest()
	err = digest.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
	})
	require.NoError(t, err)

	newBlock1 := &types.Block{
		Header: types.Header{
			ParentHash: hash,
			Number:     1,
			Digest:     types.NewDigest()},
		Body: *types.NewBody([]types.Extrinsic{[]byte("Old Runtime")}),
	}

	newBlockRTUpdate := &types.Block{
		Header: types.Header{
			ParentHash: hash,
			Number:     1,
			Digest:     digest,
		},
		Body: *types.NewBody([]types.Extrinsic{[]byte("Updated Runtime")}),
	}

	ts, err := s.storageState.TrieState(nil) // Pass genesis root
	require.NoError(t, err)

	parentRt, err := s.blockState.GetRuntime(hash)
	require.NoError(t, err)

	v = parentRt.Version()
	require.Equal(t, v.SpecVersion, currSpecVersion)

	bhash1 := newBlock1.Header.Hash()
	err = s.blockState.HandleRuntimeChanges(ts, parentRt, bhash1)
	require.NoError(t, err)

	testRuntime, err := os.ReadFile(updateNodeRuntimeWasmPath)
	require.NoError(t, err)

	ts.Set(common.CodeKey, testRuntime)
	rtUpdateBhash := newBlockRTUpdate.Header.Hash()

	// update runtime for new block
	err = s.blockState.HandleRuntimeChanges(ts, parentRt, rtUpdateBhash)
	require.NoError(t, err)

	// bhash1 runtime should not be updated
	rt, err = s.blockState.GetRuntime(bhash1)
	require.NoError(t, err)

	v = rt.Version()
	require.Equal(t, v.SpecVersion, currSpecVersion)

	rt, err = s.blockState.GetRuntime(rtUpdateBhash)
	require.NoError(t, err)

	v = rt.Version()
	require.Equal(t, v.SpecVersion, updatedSpecVersion)
}

func TestService_HandleCodeSubstitutes(t *testing.T) {
	s := NewTestService(t, nil)

	runtimeFilepath, err := runtime.GetRuntime(context.Background(), runtime.POLKADOT_RUNTIME)
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

	ts := rtstorage.NewTrieState(trie.NewEmptyTrie())
	err = s.handleCodeSubstitution(blockHash, ts)
	require.NoError(t, err)
	codSub := s.codeSubstitutedState.LoadCodeSubstitutedBlockHash()
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

	ts := rtstorage.NewTrieState(trie.NewEmptyTrie())
	err = s.handleCodeSubstitution(blockHash, ts)
	require.NoError(t, err)
	require.Equal(t, codeHashBefore, parentRt.GetCodeHash()) // codeHash should remain unchanged after code substitute

	runtimeFilepath, err := runtime.GetRuntime(context.Background(), runtime.POLKADOT_RUNTIME)
	require.NoError(t, err)
	testRuntime, err := os.ReadFile(runtimeFilepath)
	require.NoError(t, err)

	ts, err = s.storageState.TrieState(nil)
	require.NoError(t, err)

	ts.Set(common.CodeKey, testRuntime)
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
