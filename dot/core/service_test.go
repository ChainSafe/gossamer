// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"io/ioutil"
	"math/big"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core/mocks"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	runtimemocks "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Error("failed to generate runtime wasm file", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()

	runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

func TestStartService(t *testing.T) {
	s := NewTestService(t, nil)
	require.NotNil(t, s)

	err := s.Start()
	require.NoError(t, err)

	err = s.Stop()
	require.NoError(t, err)
}

func TestAnnounceBlock(t *testing.T) {
	net := new(mocks.Network)
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

	newBlock := types.Block{
		Header: types.Header{
			Number:     big.NewInt(1),
			ParentHash: s.blockState.BestBlockHash(),
			Digest:     digest,
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

	net.On("GossipMessage", expected)

	state, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	err = s.HandleBlockProduced(&newBlock, state)
	require.NoError(t, err)

	time.Sleep(time.Second)
	net.AssertCalled(t, "GossipMessage", expected)
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

	res, err := s.HasKey(kr.Alice().Public().Hex(), "babe")
	require.NoError(t, err)
	require.True(t, res)
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
	require.EqualError(t, err, "unknown key type: xxxx")
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

	rt, err := s.blockState.GetRuntime(nil)
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
	height := 5
	branch := 3
	branches := map[int]int{branch: 1}
	state.AddBlocksToStateWithFixedBranches(t, s.blockState.(*state.BlockState), height, branches, 0)

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
	height := 5
	branch := 3
	state.AddBlocksToState(t, s.blockState.(*state.BlockState), height, false)

	// create extrinsic
	enc, err := scale.Marshal([]byte("nootwashere"))
	require.NoError(t, err)
	// we prefix with []byte{2} here since that's the enum index for the old IncludeDataExt extrinsic
	tx := append([]byte{2}, enc...)

	bhash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(&bhash)
	require.NoError(t, err)

	validity, err := rt.ValidateTransaction(tx)
	require.NoError(t, err)

	// get common ancestor
	ancestor, err := s.blockState.(*state.BlockState).GetBlockByNumber(big.NewInt(int64(branch - 1)))
	require.NoError(t, err)

	// build "re-org" chain

	digest := types.NewDigest()
	block := &types.Block{
		Header: types.Header{
			ParentHash: ancestor.Header.Hash(),
			Number:     big.NewInt(0).Add(ancestor.Header.Number, big.NewInt(1)),
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

func TestMaintainTransactionPool_EmptyBlock(t *testing.T) {
	// TODO: update these to real extrinsics on update to v0.9 (#904)
	txs := []*transaction.ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &transaction.Validity{Priority: 1},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &transaction.Validity{Priority: 4},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &transaction.Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &transaction.Validity{Priority: 17},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &transaction.Validity{Priority: 2},
		},
	}

	ts := state.NewTransactionState()
	hashes := make([]common.Hash, len(txs))

	for i, tx := range txs {
		h := ts.AddToPool(tx)
		hashes[i] = h
	}

	s := &Service{
		transactionState: ts,
	}

	s.maintainTransactionPool(&types.Block{
		Body: *types.NewBody([]types.Extrinsic{}),
	})

	res := make([]*transaction.ValidTransaction, len(txs))
	for i := range txs {
		res[i] = ts.Pop()
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Extrinsic[0] < res[j].Extrinsic[0]
	})
	require.Equal(t, txs, res)

	for _, tx := range txs {
		ts.RemoveExtrinsic(tx.Extrinsic)
	}
	head := ts.Pop()
	require.Nil(t, head)
}

func TestMaintainTransactionPool_BlockWithExtrinsics(t *testing.T) {
	txs := []*transaction.ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &transaction.Validity{Priority: 1},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &transaction.Validity{Priority: 4},
		},
	}

	ts := state.NewTransactionState()
	hashes := make([]common.Hash, len(txs))

	for i, tx := range txs {
		h := ts.AddToPool(tx)
		hashes[i] = h
	}

	s := &Service{
		transactionState: ts,
	}

	s.maintainTransactionPool(&types.Block{
		Body: types.Body([]types.Extrinsic{txs[0].Extrinsic}),
	})

	res := []*transaction.ValidTransaction{}
	for {
		tx := ts.Pop()
		if tx == nil {
			break
		}
		res = append(res, tx)
	}
	require.Equal(t, 1, len(res))
	require.Equal(t, res[0], txs[1])
}

func TestService_GetRuntimeVersion(t *testing.T) {
	s := NewTestService(t, nil)
	rt, err := s.blockState.GetRuntime(nil)
	require.NoError(t, err)

	rtExpected, err := rt.Version()
	require.NoError(t, err)

	rtv, err := s.GetRuntimeVersion(nil)
	require.NoError(t, err)
	require.Equal(t, rtExpected, rtv)
}

func TestService_HandleSubmittedExtrinsic(t *testing.T) {
	s := NewTestService(t, nil)

	genHeader, err := s.blockState.BestBlockHeader()
	require.NoError(t, err)

	rt, err := s.blockState.GetRuntime(nil)
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

	rt, err := s.blockState.GetRuntime(nil)
	require.NoError(t, err)

	v, err := rt.Version()
	require.NoError(t, err)

	currSpecVersion := v.SpecVersion()   // genesis runtime version.
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
			Number:     big.NewInt(1),
			Digest:     types.NewDigest()},
		Body: *types.NewBody([]types.Extrinsic{[]byte("Old Runtime")}),
	}

	newBlockRTUpdate := &types.Block{
		Header: types.Header{
			ParentHash: hash,
			Number:     big.NewInt(1),
			Digest:     digest,
		},
		Body: *types.NewBody([]types.Extrinsic{[]byte("Updated Runtime")}),
	}

	ts, err := s.storageState.TrieState(nil) // Pass genesis root
	require.NoError(t, err)

	parentRt, err := s.blockState.GetRuntime(&hash)
	require.NoError(t, err)

	v, err = parentRt.Version()
	require.NoError(t, err)
	require.Equal(t, v.SpecVersion(), currSpecVersion)

	bhash1 := newBlock1.Header.Hash()
	err = s.blockState.HandleRuntimeChanges(ts, parentRt, bhash1)
	require.NoError(t, err)

	testRuntime, err := ioutil.ReadFile(updateNodeRuntimeWasmPath)
	require.NoError(t, err)

	ts.Set(common.CodeKey, testRuntime)
	rtUpdateBhash := newBlockRTUpdate.Header.Hash()

	// update runtime for new block
	err = s.blockState.HandleRuntimeChanges(ts, parentRt, rtUpdateBhash)
	require.NoError(t, err)

	// bhash1 runtime should not be updated
	rt, err = s.blockState.GetRuntime(&bhash1)
	require.NoError(t, err)

	v, err = rt.Version()
	require.NoError(t, err)
	require.Equal(t, v.SpecVersion(), currSpecVersion)

	rt, err = s.blockState.GetRuntime(&rtUpdateBhash)
	require.NoError(t, err)

	v, err = rt.Version()
	require.NoError(t, err)
	require.Equal(t, v.SpecVersion(), updatedSpecVersion)
}

func TestService_HandleCodeSubstitutes(t *testing.T) {
	s := NewTestService(t, nil)

	testRuntime, err := ioutil.ReadFile(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)

	blockHash := common.MustHexToHash("0x86aa36a140dfc449c30dbce16ce0fea33d5c3786766baa764e33f336841b9e29") // hash for known test code substitution
	s.codeSubstitute = map[common.Hash]string{
		blockHash: common.BytesToHex(testRuntime),
	}

	rt, err := s.blockState.GetRuntime(nil)
	require.NoError(t, err)

	s.blockState.StoreRuntime(blockHash, rt)

	ts, err := rtstorage.NewTrieState(trie.NewEmptyTrie())
	require.NoError(t, err)

	err = s.handleCodeSubstitution(blockHash, ts)
	require.NoError(t, err)
	codSub := s.codeSubstitutedState.LoadCodeSubstitutedBlockHash()
	require.Equal(t, blockHash, codSub)
}

func TestService_HandleRuntimeChangesAfterCodeSubstitutes(t *testing.T) {
	s := NewTestService(t, nil)

	parentRt, err := s.blockState.GetRuntime(nil)
	require.NoError(t, err)

	codeHashBefore := parentRt.GetCodeHash()
	blockHash := common.MustHexToHash("0x86aa36a140dfc449c30dbce16ce0fea33d5c3786766baa764e33f336841b9e29") // hash for known test code substitution

	body := types.NewBody([]types.Extrinsic{[]byte("Updated Runtime")})
	newBlock := &types.Block{
		Header: types.Header{
			ParentHash: blockHash,
			Number:     big.NewInt(1),
			Digest:     types.NewDigest(),
		},
		Body: *body,
	}

	ts, err := rtstorage.NewTrieState(trie.NewEmptyTrie())
	require.NoError(t, err)

	err = s.handleCodeSubstitution(blockHash, ts)
	require.NoError(t, err)
	require.Equal(t, codeHashBefore, parentRt.GetCodeHash()) // codeHash should remain unchanged after code substitute

	testRuntime, err := ioutil.ReadFile(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)

	ts, err = s.storageState.TrieState(nil)
	require.NoError(t, err)

	ts.Set(common.CodeKey, testRuntime)
	rtUpdateBhash := newBlock.Header.Hash()

	// update runtime for new block
	err = s.blockState.HandleRuntimeChanges(ts, parentRt, rtUpdateBhash)
	require.NoError(t, err)

	rt, err := s.blockState.GetRuntime(&rtUpdateBhash)
	require.NoError(t, err)

	require.NotEqualf(t, codeHashBefore, rt.GetCodeHash(), "expected different code hash after runtime update") // codeHash should change after runtime change
}

func TestTryQueryStore_WhenThereIsDataToRetrieve(t *testing.T) {
	s := NewTestService(t, nil)
	storageStateTrie, err := rtstorage.NewTrieState(trie.NewTrie(nil))

	testKey, testValue := []byte("to"), []byte("0x1723712318238AB12312")
	storageStateTrie.Set(testKey, testValue)
	require.NoError(t, err)

	header, err := types.NewHeader(s.blockState.GenesisHash(), storageStateTrie.MustRoot(),
		common.Hash{}, big.NewInt(1), types.NewDigest())
	require.NoError(t, err)

	err = s.storageState.StoreTrie(storageStateTrie, header)
	require.NoError(t, err)

	testBlock := &types.Block{
		Header: *header,
		Body:   *types.NewBody([]types.Extrinsic{}),
	}

	err = s.blockState.AddBlock(testBlock)
	require.NoError(t, err)

	blockhash := testBlock.Header.Hash()
	hexKey := common.BytesToHex(testKey)
	keys := []string{hexKey}

	changes, err := s.tryQueryStorage(blockhash, keys...)
	require.NoError(t, err)

	require.Equal(t, changes[hexKey], common.BytesToHex(testValue))
}

func TestTryQueryStore_WhenDoesNotHaveDataToRetrieve(t *testing.T) {
	s := NewTestService(t, nil)
	storageStateTrie, err := rtstorage.NewTrieState(trie.NewTrie(nil))
	require.NoError(t, err)

	header, err := types.NewHeader(s.blockState.GenesisHash(), storageStateTrie.MustRoot(),
		common.Hash{}, big.NewInt(1), types.NewDigest())
	require.NoError(t, err)

	err = s.storageState.StoreTrie(storageStateTrie, header)
	require.NoError(t, err)

	testBlock := &types.Block{
		Header: *header,
		Body:   *types.NewBody([]types.Extrinsic{}),
	}

	err = s.blockState.AddBlock(testBlock)
	require.NoError(t, err)

	testKey := []byte("to")
	blockhash := testBlock.Header.Hash()
	hexKey := common.BytesToHex(testKey)
	keys := []string{hexKey}

	changes, err := s.tryQueryStorage(blockhash, keys...)
	require.NoError(t, err)

	require.Empty(t, changes)
}

func TestTryQueryState_WhenDoesNotHaveStateRoot(t *testing.T) {
	s := NewTestService(t, nil)

	header, err := types.NewHeader(s.blockState.GenesisHash(), common.Hash{}, common.Hash{}, big.NewInt(1), types.NewDigest())
	require.NoError(t, err)

	testBlock := &types.Block{
		Header: *header,
		Body:   *types.NewBody([]types.Extrinsic{}),
	}

	err = s.blockState.AddBlock(testBlock)
	require.NoError(t, err)

	testKey := []byte("to")
	blockhash := testBlock.Header.Hash()
	hexKey := common.BytesToHex(testKey)
	keys := []string{hexKey}

	changes, err := s.tryQueryStorage(blockhash, keys...)
	require.Error(t, err)
	require.Nil(t, changes)
}

func TestQueryStorate_WhenBlocksHasData(t *testing.T) {
	keys := []string{
		common.BytesToHex([]byte("transfer.to")),
		common.BytesToHex([]byte("transfer.from")),
		common.BytesToHex([]byte("transfer.value")),
	}

	s := NewTestService(t, nil)

	firstKey, firstValue := []byte("transfer.to"), []byte("some-address-herer")
	firstBlock := createNewBlockAndStoreDataAtBlock(
		t, s, firstKey, firstValue, s.blockState.GenesisHash(), 1,
	)

	secondKey, secondValue := []byte("transfer.from"), []byte("another-address-here")
	secondBlock := createNewBlockAndStoreDataAtBlock(
		t, s, secondKey, secondValue, firstBlock.Header.Hash(), 2,
	)

	thirdKey, thirdValue := []byte("transfer.value"), []byte("value-gigamegablaster")
	thirdBlock := createNewBlockAndStoreDataAtBlock(
		t, s, thirdKey, thirdValue, secondBlock.Header.Hash(), 3,
	)

	from := firstBlock.Header.Hash()
	data, err := s.QueryStorage(from, common.Hash{}, keys...)
	require.NoError(t, err)
	require.Len(t, data, 3)

	require.Equal(t, data[firstBlock.Header.Hash()], QueryKeyValueChanges(
		map[string]string{
			common.BytesToHex(firstKey): common.BytesToHex(firstValue),
		},
	))

	from = secondBlock.Header.Hash()
	to := thirdBlock.Header.Hash()

	data, err = s.QueryStorage(from, to, keys...)
	require.NoError(t, err)
	require.Len(t, data, 2)

	require.Equal(t, data[secondBlock.Header.Hash()], QueryKeyValueChanges(
		map[string]string{
			common.BytesToHex(secondKey): common.BytesToHex(secondValue),
		},
	))
	require.Equal(t, data[thirdBlock.Header.Hash()], QueryKeyValueChanges(
		map[string]string{
			common.BytesToHex(thirdKey): common.BytesToHex(thirdValue),
		},
	))
}

func createNewBlockAndStoreDataAtBlock(t *testing.T, s *Service, key, value []byte, parentHash common.Hash, number int64) *types.Block {
	t.Helper()

	storageStateTrie, err := rtstorage.NewTrieState(trie.NewTrie(nil))
	storageStateTrie.Set(key, value)
	require.NoError(t, err)

	header, err := types.NewHeader(parentHash, storageStateTrie.MustRoot(), common.Hash{}, big.NewInt(number), types.NewDigest())
	require.NoError(t, err)

	err = s.storageState.StoreTrie(storageStateTrie, header)
	require.NoError(t, err)

	testBlock := &types.Block{
		Header: *header,
		Body:   *types.NewBody([]types.Extrinsic{}),
	}

	err = s.blockState.AddBlock(testBlock)
	require.NoError(t, err)

	return testBlock
}

func TestDecodeSessionKeys(t *testing.T) {
	mockInstance := new(runtimemocks.Instance)
	mockInstance.On("DecodeSessionKeys", mock.AnythingOfType("[]uint8")).Return([]byte{}, nil).Once()

	mockBlockState := new(mocks.BlockState)
	mockBlockState.On("GetRuntime", mock.AnythingOfType("*common.Hash")).Return(mockInstance, nil).Once()

	coreservice := new(Service)
	coreservice.blockState = mockBlockState

	b, err := coreservice.DecodeSessionKeys([]byte{})

	mockBlockState.AssertCalled(t, "GetRuntime", mock.AnythingOfType("*common.Hash"))
	mockInstance.AssertCalled(t, "DecodeSessionKeys", []uint8{})

	require.NoError(t, err)
	require.Equal(t, b, []byte{})
}

func TestDecodeSessionKeys_WhenGetRuntimeReturnError(t *testing.T) {
	mockBlockState := new(mocks.BlockState)
	mockBlockState.On("GetRuntime", mock.AnythingOfType("*common.Hash")).Return(nil, errors.New("problems")).Once()

	coreservice := new(Service)
	coreservice.blockState = mockBlockState

	b, err := coreservice.DecodeSessionKeys([]byte{})

	mockBlockState.AssertCalled(t, "GetRuntime", mock.AnythingOfType("*common.Hash"))
	require.Error(t, err, "problems")
	require.Nil(t, b)
}

func TestGetReadProofAt(t *testing.T) {
	keysToProof := [][]byte{[]byte("first_key"), []byte("another_key")}
	mockedProofs := [][]byte{[]byte("proof01"), []byte("proof02")}

	t.Run("When Has Block Is Empty", func(t *testing.T) {
		mockedStateRootHash := common.NewHash([]byte("state root hash"))
		expectedBlockHash := common.NewHash([]byte("expected block hash"))

		mockBlockState := new(mocks.BlockState)
		mockBlockState.On("BestBlockHash").Return(expectedBlockHash)
		mockBlockState.On("GetBlockStateRoot", expectedBlockHash).
			Return(mockedStateRootHash, nil)

		mockStorageStage := new(mocks.StorageState)
		mockStorageStage.On("GenerateTrieProof", mockedStateRootHash, keysToProof).
			Return(mockedProofs, nil)

		s := &Service{
			blockState:   mockBlockState,
			storageState: mockStorageStage,
		}

		b, p, err := s.GetReadProofAt(common.Hash{}, keysToProof)
		require.NoError(t, err)
		require.Equal(t, p, mockedProofs)
		require.Equal(t, expectedBlockHash, b)

		mockBlockState.AssertCalled(t, "BestBlockHash")
		mockBlockState.AssertCalled(t, "GetBlockStateRoot", expectedBlockHash)
		mockStorageStage.AssertCalled(t, "GenerateTrieProof", mockedStateRootHash, keysToProof)
	})

	t.Run("When GetStateRoot fails", func(t *testing.T) {
		mockedBlockHash := common.NewHash([]byte("fake block hash"))

		mockBlockState := new(mocks.BlockState)
		mockBlockState.On("GetBlockStateRoot", mockedBlockHash).
			Return(common.Hash{}, errors.New("problems while getting state root"))

		s := &Service{
			blockState: mockBlockState,
		}

		b, p, err := s.GetReadProofAt(mockedBlockHash, keysToProof)
		require.True(t, b.IsEmpty())
		require.Nil(t, p)
		require.Error(t, err)
	})

	t.Run("When GenerateTrieProof fails", func(t *testing.T) {
		mockedBlockHash := common.NewHash([]byte("fake block hash"))
		mockedStateRootHash := common.NewHash([]byte("state root hash"))

		mockBlockState := new(mocks.BlockState)
		mockBlockState.On("GetBlockStateRoot", mockedBlockHash).
			Return(mockedStateRootHash, nil)

		mockStorageStage := new(mocks.StorageState)
		mockStorageStage.On("GenerateTrieProof", mockedStateRootHash, keysToProof).
			Return(nil, errors.New("problems to generate trie proof"))

		s := &Service{
			blockState:   mockBlockState,
			storageState: mockStorageStage,
		}

		b, p, err := s.GetReadProofAt(mockedBlockHash, keysToProof)
		require.True(t, b.IsEmpty())
		require.Nil(t, p)
		require.Error(t, err)
	})
}
