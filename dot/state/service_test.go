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

package state

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/metrics"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/ChainSafe/chaindb"
	log "github.com/ChainSafe/log15"
	ethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/stretchr/testify/require"
)

// helper method to create and start test state service
func newTestService(t *testing.T) (state *Service) {
	testDir := utils.NewTestDir(t)
	config := Config{
		Path:     testDir,
		LogLevel: log.LvlInfo,
	}
	state = NewService(config)
	return state
}

func newTestMemDBService() *Service {
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")
	config := Config{
		Path:     testDatadirPath,
		LogLevel: log.LvlInfo,
	}
	state := NewService(config)
	state.UseMemDB()
	return state
}

func TestService_Start(t *testing.T) {
	state := newTestService(t)
	defer utils.RemoveTestDir(t)

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := state.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	err = state.Stop()
	require.NoError(t, err)
}

func TestService_Initialise(t *testing.T) {
	state := newTestService(t)
	defer utils.RemoveTestDir(t)

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := state.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	genesisHeader, err = types.NewHeader(common.NewHash([]byte{77}), genTrie.MustHash(), trie.EmptyHash, big.NewInt(0), types.NewDigest())
	require.NoError(t, err)

	err = state.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	head, err := state.Block.BestBlockHeader()
	require.NoError(t, err)
	require.Equal(t, genesisHeader, head)
}

func TestMemDB_Start(t *testing.T) {
	state := newTestMemDBService()

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := state.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	err = state.Stop()
	require.NoError(t, err)
}

func TestService_BlockTree(t *testing.T) {
	testDir := utils.NewTestDir(t)

	// removes all data directories created within test directory
	defer utils.RemoveTestDir(t)

	config := Config{
		Path:     testDir,
		LogLevel: log.LvlInfo,
	}
	stateA := NewService(config)

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := stateA.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = stateA.Start()
	require.NoError(t, err)

	// add blocks to state
	AddBlocksToState(t, stateA.Block, 10, false)
	head := stateA.Block.BestBlockHash()

	err = stateA.Block.SetFinalisedHash(head, 1, 1)
	require.NoError(t, err)

	err = stateA.Stop()
	require.NoError(t, err)

	stateB := NewService(config)

	err = stateB.Start()
	require.NoError(t, err)

	err = stateB.Stop()
	require.NoError(t, err)
	require.Equal(t, stateA.Block.BestBlockHash(), stateB.Block.BestBlockHash())
}

func TestService_StorageTriePruning(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	retainBlocks := 2
	config := Config{
		Path:     testDir,
		LogLevel: log.LvlInfo,
		PrunerCfg: pruner.Config{
			Mode:           pruner.Full,
			RetainedBlocks: int64(retainBlocks),
		},
	}
	serv := NewService(config)
	serv.UseMemDB()

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := serv.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = serv.Start()
	require.NoError(t, err)

	var blocks []*types.Block
	parentHash := serv.Block.GenesisHash()

	totalBlock := 10
	for i := 1; i < totalBlock; i++ {
		block, trieState := generateBlockWithRandomTrie(t, serv, &parentHash, int64(i))

		err = serv.Storage.blockState.AddBlock(block)
		require.NoError(t, err)

		err = serv.Storage.StoreTrie(trieState, &block.Header)
		require.NoError(t, err)

		blocks = append(blocks, block)
		parentHash = block.Header.Hash()
	}

	time.Sleep(2 * time.Second)

	for _, b := range blocks {
		_, err := serv.Storage.LoadFromDB(b.Header.StateRoot)
		if b.Header.Number.Int64() >= int64(totalBlock-retainBlocks-1) {
			require.NoError(t, err, fmt.Sprintf("Got error for block %d", b.Header.Number.Int64()))
			continue
		}
		require.ErrorIs(t, err, chaindb.ErrKeyNotFound, fmt.Sprintf("Expected error for block %d", b.Header.Number.Int64()))
	}
}

func TestService_PruneStorage(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	config := Config{
		Path:     testDir,
		LogLevel: log.LvlInfo,
	}
	serv := NewService(config)
	serv.UseMemDB()

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := serv.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = serv.Start()
	require.NoError(t, err)

	type prunedBlock struct {
		hash  common.Hash
		dbKey []byte
	}

	var toFinalize common.Hash
	for i := 0; i < 3; i++ {
		block, trieState := generateBlockWithRandomTrie(t, serv, nil, int64(i+1))
		digest := types.NewDigest()
		digest.Add(*types.NewBabeSecondaryPlainPreDigest(0, uint64(i+1)).ToPreRuntimeDigest())
		block.Header.Digest = digest

		err = serv.Storage.blockState.AddBlock(block)
		require.NoError(t, err)

		err = serv.Storage.StoreTrie(trieState, nil)
		require.NoError(t, err)

		// Only finalise a block at height 3
		if i == 2 {
			toFinalize = block.Header.Hash()
		}
	}

	// add some blocks to prune, on a different chain from the finalised block
	var prunedArr []prunedBlock
	parentHash := serv.Block.GenesisHash()
	for i := 0; i < 3; i++ {
		block, trieState := generateBlockWithRandomTrie(t, serv, &parentHash, int64(i+1))

		err = serv.Storage.blockState.AddBlock(block)
		require.NoError(t, err)

		err = serv.Storage.StoreTrie(trieState, nil)
		require.NoError(t, err)

		// Store the other blocks that will be pruned.
		var trieVal *trie.Trie
		trieVal, err = trieState.Trie().DeepCopy()
		require.NoError(t, err)

		var rootHash common.Hash
		rootHash, err = trieVal.Hash()
		require.NoError(t, err)

		prunedArr = append(prunedArr, prunedBlock{hash: block.Header.StateRoot, dbKey: rootHash[:]})
		parentHash = block.Header.Hash()
	}

	// finalise a block
	serv.Block.SetFinalisedHash(toFinalize, 0, 0)

	time.Sleep(1 * time.Second)

	for _, v := range prunedArr {
		_, has := serv.Storage.tries.Load(v.hash)
		require.Equal(t, false, has)
	}
}

func TestService_Rewind(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	config := Config{
		Path:     testDir,
		LogLevel: log.LvlInfo,
	}
	serv := NewService(config)
	serv.UseMemDB()

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := serv.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = serv.Start()
	require.NoError(t, err)

	err = serv.Grandpa.setCurrentSetID(3)
	require.NoError(t, err)

	err = serv.Grandpa.setSetIDChangeAtBlock(1, big.NewInt(5))
	require.NoError(t, err)

	err = serv.Grandpa.setSetIDChangeAtBlock(2, big.NewInt(8))
	require.NoError(t, err)

	err = serv.Grandpa.setSetIDChangeAtBlock(3, big.NewInt(10))
	require.NoError(t, err)

	AddBlocksToState(t, serv.Block, 12, false)
	head := serv.Block.BestBlockHash()
	err = serv.Block.SetFinalisedHash(head, 0, 0)
	require.NoError(t, err)

	err = serv.Rewind(6)
	require.NoError(t, err)

	num, err := serv.Block.BestBlockNumber()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(6), num)

	setID, err := serv.Grandpa.GetCurrentSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), setID)

	_, err = serv.Grandpa.GetSetIDChange(1)
	require.NoError(t, err)

	_, err = serv.Grandpa.GetSetIDChange(2)
	require.Equal(t, chaindb.ErrKeyNotFound, err)

	_, err = serv.Grandpa.GetSetIDChange(3)
	require.Equal(t, chaindb.ErrKeyNotFound, err)
}

func TestService_Import(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	config := Config{
		Path:     testDir,
		LogLevel: log.LvlInfo,
	}
	serv := NewService(config)
	serv.UseMemDB()

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := serv.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)
	err = serv.db.Close()
	require.NoError(t, err)

	tr := trie.NewEmptyTrie()
	var testCases = []string{
		"asdf",
		"ghjk",
		"qwerty",
		"uiopl",
		"zxcv",
		"bnm",
	}
	for _, tc := range testCases {
		tr.Put([]byte(tc), []byte(tc))
	}

	digest := types.NewDigest()
	digest.Add(*types.NewBabeSecondaryPlainPreDigest(0, 177).ToPreRuntimeDigest())
	header := &types.Header{
		Number:    big.NewInt(77),
		StateRoot: tr.MustHash(),
		Digest:    digest,
	}

	firstSlot := uint64(100)

	err = serv.Import(header, tr, firstSlot)
	require.NoError(t, err)

	err = serv.Start()
	require.NoError(t, err)

	bestBlockHeader, err := serv.Block.BestBlockHeader()
	require.NoError(t, err)
	require.Equal(t, header, bestBlockHeader)

	root, err := serv.Storage.StorageRoot()
	require.NoError(t, err)
	require.Equal(t, header.StateRoot, root)

	skip, err := serv.Epoch.SkipVerify(header)
	require.NoError(t, err)
	require.True(t, skip)

	err = serv.Stop()
	require.NoError(t, err)
}

func TestStateServiceMetrics(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	config := Config{
		Path:     testDir,
		LogLevel: log.LvlInfo,
	}
	ethmetrics.Enabled = true
	serv := NewService(config)
	serv.Transaction = NewTransactionState()

	m := metrics.NewCollector(context.Background())
	m.AddGauge(serv)
	go m.Start()

	vtxs := []*transaction.ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &transaction.Validity{Priority: 1},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &transaction.Validity{Priority: 4},
		},
	}

	hashes := make([]common.Hash, len(vtxs))
	for i, v := range vtxs {
		h := serv.Transaction.pool.Insert(v)
		serv.Transaction.queue.Push(v)

		hashes[i] = h
	}

	time.Sleep(time.Second + metrics.Refresh)
	gpool := ethmetrics.GetOrRegisterGauge(readyPoolTransactionsMetrics, nil)
	gqueue := ethmetrics.GetOrRegisterGauge(readyPriorityQueueTransactions, nil)

	require.Equal(t, int64(2), gpool.Value())
	require.Equal(t, int64(2), gqueue.Value())

	serv.Transaction.pool.Remove(hashes[0])
	serv.Transaction.queue.Pop()

	time.Sleep(time.Second + metrics.Refresh)
	require.Equal(t, int64(1), gpool.Value())
	require.Equal(t, int64(1), gqueue.Value())
}
