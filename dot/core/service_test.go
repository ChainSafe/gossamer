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
	"io/ioutil"
	"math/big"
	"os"
	//"sort"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/extrinsic"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/utils"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"

	coremocks "github.com/ChainSafe/gossamer/dot/core/mocks"
)

func addTestBlocksToState(t *testing.T, depth int, blockState BlockState) {
	_ = addTestBlocksToStateWithParent(t, blockState.BestBlockHash(), depth, blockState)
}

func addTestBlocksToStateWithParent(t *testing.T, previousHash common.Hash, depth int, blockState BlockState) []*types.Header {
	prevHeader, err := blockState.(*state.BlockState).GetHeader(previousHash)
	require.NoError(t, err)
	previousNum := prevHeader.Number

	headers := []*types.Header{}

	for i := 1; i <= depth; i++ {
		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)).Add(previousNum, big.NewInt(int64(i))),
				Digest:     types.Digest{},
			},
			Body: &types.Body{},
		}

		previousHash = block.Header.Hash()

		err := blockState.AddBlock(block)
		require.NoError(t, err)
		headers = append(headers, block.Header)
	}

	return headers
}

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

	// TODO: improve dot tests #687
	require.NotNil(t, s)

	err := s.Start()
	require.Nil(t, err)

	err = s.Stop()
	require.NoError(t, err)
}

func TestAnnounceBlock(t *testing.T) {
	net := new(coremocks.MockNetwork)
	cfg := &Config{
		Network: net,
	}

	s := NewTestService(t, cfg)
	err := s.Start()
	require.NoError(t, err)
	defer s.Stop()

	// simulate block sent from BABE session
	newBlock := &types.Block{
		Header: &types.Header{
			ParentHash: s.blockState.BestBlockHash(),
			Number:     big.NewInt(1),
			Digest:     types.Digest{types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()},
		},
		Body: &types.Body{},
	}

	expected := &network.BlockAnnounceMessage{
		ParentHash:     newBlock.Header.ParentHash,
		Number:         newBlock.Header.Number,
		StateRoot:      newBlock.Header.StateRoot,
		ExtrinsicsRoot: newBlock.Header.ExtrinsicsRoot,
		Digest:         newBlock.Header.Digest,
		BestBlock:      true,
	}

	net.On("GossipMessage", expected)

	state, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	err = s.HandleBlockProduced(newBlock, state)
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
	addTestBlocksToState(t, 4, s.blockState.(*state.BlockState))

	head, err := s.blockState.BestBlockHeader()
	require.NoError(t, err)

	err = s.handleChainReorg(head.ParentHash, head.Hash())
	require.NoError(t, err)
}

func TestHandleChainReorg_WithReorg_Trans(t *testing.T) {
	s := NewTestService(t, nil)

	bs := s.blockState

	parent, err := bs.BestBlockHeader()
	require.NoError(t, err)

	block1 := sync.BuildBlock(t, s.rt, parent, nil)
	err = bs.AddBlock(block1)
	require.NoError(t, err)

	block2 := sync.BuildBlock(t, s.rt, block1.Header, nil)
	err = bs.AddBlock(block2)
	require.NoError(t, err)

	block3 := sync.BuildBlock(t, s.rt, block2.Header, nil)
	err = bs.AddBlock(block3)
	require.NoError(t, err)

	block4 := sync.BuildBlock(t, s.rt, block3.Header, nil)
	err = bs.AddBlock(block4)
	require.NoError(t, err)

	block5 := sync.BuildBlock(t, s.rt, block4.Header, nil)
	err = bs.AddBlock(block5)
	require.NoError(t, err)

	block31 := sync.BuildBlock(t, s.rt, block2.Header, nil)
	err = bs.AddBlock(block31)
	require.NoError(t, err)

	nonce := uint64(1)

	// Add extrinsic to block `block31`
	ext := createExtrinsic(t, s.rt, bs.GenesisHash(), nonce)

	block41 := sync.BuildBlock(t, s.rt, block31.Header, ext)
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
	addTestBlocksToState(t, height, s.blockState.(*state.BlockState))

	// create extrinsic
	ext := extrinsic.NewIncludeDataExt([]byte("nootwashere"))
	tx, err := ext.Encode()
	require.NoError(t, err)

	validity, err := s.rt.ValidateTransaction(tx)
	require.NoError(t, err)

	// get common ancestor
	ancestor, err := s.blockState.(*state.BlockState).GetBlockByNumber(big.NewInt(int64(branch - 1)))
	require.NoError(t, err)

	// build "re-org" chain
	body, err := types.NewBodyFromExtrinsics([]types.Extrinsic{tx})
	require.NoError(t, err)

	block := &types.Block{
		Header: &types.Header{
			ParentHash: ancestor.Header.Hash(),
			Number:     big.NewInt(0).Add(ancestor.Header.Number, big.NewInt(1)),
			Digest: types.Digest{
				utils.NewMockDigestItem(1),
			},
		},
		Body: body,
	}

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
	// TODO" update these to real extrinsics on update to v0.8
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

	err := s.maintainTransactionPool(&types.Block{
		Body: types.NewBody([]byte{}),
	})
	require.NoError(t, err)

	// res := make([]*transaction.ValidTransaction, len(txs))
	// for i := range txs {
	// 	res[i] = ts.Pop()
	// }

	// sort.Slice(res, func(i, j int) bool {
	// 	return res[i].Extrinsic[0] < res[j].Extrinsic[0]
	// })
	// require.Equal(t, txs, res)

	// for _, tx := range txs {
	// 	ts.RemoveExtrinsic(tx.Extrinsic)
	// }
	// head := ts.Pop()
	// require.Nil(t, head)
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

	body, err := types.NewBodyFromExtrinsics([]types.Extrinsic{txs[0].Extrinsic})
	require.NoError(t, err)

	err = s.maintainTransactionPool(&types.Block{
		Body: body,
	})
	require.NoError(t, err)

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
	rtExpected, err := s.rt.Version()
	require.NoError(t, err)

	rtv, err := s.GetRuntimeVersion(nil)
	require.NoError(t, err)
	require.Equal(t, rtExpected, rtv)
}

func TestService_HandleSubmittedExtrinsic(t *testing.T) {
	s := NewTestService(t, nil)

	parentHash := common.MustHexToHash("0x35a28a7dbaf0ba07d1485b0f3da7757e3880509edc8c31d0850cb6dd6219361d")
	header, err := types.NewHeader(parentHash, common.Hash{}, common.Hash{}, big.NewInt(1), types.NewEmptyDigest())
	require.NoError(t, err)

	extBytes := createExtrinsic(t, s.rt, parentHash, 0)

	//initialise block header
	err = s.rt.InitializeBlock(header)
	require.NoError(t, err)

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
	codeHashBefore := s.codeHash

	testRuntime, err := ioutil.ReadFile(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)

	ts, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	ts.Set(common.CodeKey, testRuntime)
	err = s.handleRuntimeChanges(ts)
	require.NoError(t, err)
	codeHashAfter := s.codeHash
	require.NotEqualf(t, codeHashBefore, codeHashAfter, "expected different code hash after runtime update")
}

func TestService_HandleCodeSubstitutes(t *testing.T) {
	s := NewTestService(t, nil)

	testRuntime, err := ioutil.ReadFile(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)

	blockHash := common.MustHexToHash("0x86aa36a140dfc449c30dbce16ce0fea33d5c3786766baa764e33f336841b9e29") // hash for known test code substitution
	s.codeSubstitute = map[common.Hash]string{
		blockHash: common.BytesToHex(testRuntime),
	}

	err = s.handleCodeSubstitution(blockHash)
	require.NoError(t, err)
	codSub := s.codeSubstitutedState.LoadCodeSubstitutedBlockHash()
	require.Equal(t, blockHash, codSub)
}

func TestService_HandleRuntimeChangesAfterCodeSubstitutes(t *testing.T) {
	s := NewTestService(t, nil)
	codeHashBefore := s.codeHash
	blockHash := common.MustHexToHash("0x86aa36a140dfc449c30dbce16ce0fea33d5c3786766baa764e33f336841b9e29") // hash for known test code substitution

	err := s.handleCodeSubstitution(blockHash)
	require.NoError(t, err)
	require.Equal(t, codeHashBefore, s.codeHash) // codeHash should remain unchanged after code substitute

	testRuntime, err := ioutil.ReadFile(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)

	ts, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	ts.Set(common.CodeKey, testRuntime)
	err = s.handleRuntimeChanges(ts)
	require.NoError(t, err)
	require.NotEqualf(t, codeHashBefore, s.codeHash, "expected different code hash after runtime update") // codeHash should change after runtime change
}
