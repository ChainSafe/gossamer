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
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func newTestGenesisWithTrieAndHeader(t *testing.T) (*genesis.Genesis, *trie.Trie, *types.Header) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../chain/gssmr/genesis.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), genTrie.MustHash(), trie.EmptyHash, types.Digest{})
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
}

// helper method to create and start test state service
func newTestService(t *testing.T) (state *Service) {
	testDir := utils.NewTestDir(t)
	state = NewService(testDir, log.LvlTrace)
	return state
}

func newTestMemDBService() *Service {
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")
	state := NewService(testDatadirPath, log.LvlTrace)
	state.UseMemDB()
	return state
}

func TestService_Start(t *testing.T) {
	state := newTestService(t)
	defer utils.RemoveTestDir(t)

	genData, genTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)
	err := state.Initialize(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	err = state.Stop()
	require.NoError(t, err)
}

func TestService_Initialize(t *testing.T) {
	state := newTestService(t)
	defer utils.RemoveTestDir(t)

	genData, genTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)
	err := state.Initialize(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	genesisHeader, err = types.NewHeader(common.NewHash([]byte{77}), big.NewInt(0), genTrie.MustHash(), trie.EmptyHash, types.Digest{})
	require.NoError(t, err)

	err = state.Initialize(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	head, err := state.Block.BestBlockHeader()
	require.NoError(t, err)
	require.Equal(t, genesisHeader, head)
}

func TestMemDB_Start(t *testing.T) {
	state := newTestMemDBService()

	genData, genTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)
	err := state.Initialize(genData, genesisHeader, genTrie)
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

	stateA := NewService(testDir, log.LvlTrace)

	genData, genTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)
	err := stateA.Initialize(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = stateA.Start()
	require.NoError(t, err)

	// add blocks to state
	AddBlocksToState(t, stateA.Block, 10)

	err = stateA.Stop()
	require.NoError(t, err)

	stateB := NewService(testDir, log.LvlTrace)

	err = stateB.Start()
	require.NoError(t, err)

	err = stateB.Stop()
	require.NoError(t, err)
	require.Equal(t, stateA.Block.BestBlockHash(), stateB.Block.BestBlockHash())
}

func TestService_PruneStorage(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	serv := NewService(testDir, log.LvlTrace)
	serv.UseMemDB()

	genData, genTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)
	err := serv.Initialize(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = serv.Start()
	require.NoError(t, err)

	type prunedBlock struct {
		hash  common.Hash
		dbKey []byte
	}

	//var prunedArr []prunedBlock
	var toFinalize common.Hash

	for i := 0; i < 3; i++ {
		block, trieState := generateBlockWithRandomTrie(t, serv, nil)

		err = serv.Storage.blockState.AddBlock(block)
		require.NoError(t, err)

		err = serv.Storage.StoreTrie(trieState)
		require.NoError(t, err)

		// Only finalize a block at height 3
		if i == 2 {
			toFinalize = block.Header.Hash()
		}
	}

	// add some blocks to prune, on a different chain from the finalized block
	prunedArr := []prunedBlock{}
	parentHash := serv.Block.GenesisHash()
	for i := 0; i < 3; i++ {
		block, trieState := generateBlockWithRandomTrie(t, serv, &parentHash)

		err = serv.Storage.blockState.AddBlock(block)
		require.NoError(t, err)

		err = serv.Storage.StoreTrie(trieState)
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

	// finalize a block
	serv.Block.SetFinalizedHash(toFinalize, 0, 0)

	time.Sleep(1 * time.Second)

	for _, v := range prunedArr {
		serv.Storage.lock.Lock()
		_, ok := serv.Storage.tries[v.hash]
		serv.Storage.lock.Unlock()
		require.Equal(t, false, ok)
	}
}

func TestService_Rewind(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	serv := NewService(testDir, log.LvlTrace)
	serv.UseMemDB()

	genData, genTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)
	err := serv.Initialize(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = serv.Start()
	require.NoError(t, err)

	AddBlocksToState(t, serv.Block, 12)
	err = serv.Rewind(6)
	require.NoError(t, err)

	num, err := serv.Block.BestBlockNumber()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(6), num)
}

func TestService_Import(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	serv := NewService(testDir, log.LvlTrace)
	serv.UseMemDB()

	genData, genTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)
	err := serv.Initialize(genData, genesisHeader, genTrie)
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

	header := &types.Header{
		Number:    big.NewInt(77),
		StateRoot: tr.MustHash(),
		Digest:    types.Digest{types.NewBabeSecondaryPlainPreDigest(0, 177).ToPreRuntimeDigest()},
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

	require.Equal(t, firstSlot, serv.Epoch.firstSlot)
	skip, err := serv.Epoch.SkipVerify(header)
	require.NoError(t, err)
	require.True(t, skip)

	err = serv.Stop()
	require.NoError(t, err)
}
