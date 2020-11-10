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
	"math/big"
	"reflect"
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

var firstEpochInfo = &types.EpochInfo{
	Duration:   200,
	FirstBlock: 0,
}

// helper method to create and start test state service
func newTestService(t *testing.T) (state *Service) {
	testDir := utils.NewTestDir(t)
	state = NewService(testDir, log.LvlTrace)
	return state
}

func newTestMemDBService() *Service {
	state := NewService("", log.LvlTrace)
	state.UseMemDB()
	return state
}

func TestService_Start(t *testing.T) {
	state := newTestService(t)
	defer utils.RemoveTestDir(t)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), trie.EmptyHash, trie.EmptyHash, [][]byte{})
	require.NoError(t, err)

	tr := trie.NewEmptyTrie()

	genesisData := new(genesis.Data)

	err = state.Initialize(genesisData, genesisHeader, tr, firstEpochInfo)
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	err = state.Stop()
	require.NoError(t, err)
}

func TestMemDB_Start(t *testing.T) {
	state := newTestMemDBService()

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), trie.EmptyHash, trie.EmptyHash, [][]byte{})
	require.NoError(t, err)

	tr := trie.NewEmptyTrie()

	genesisData := new(genesis.Data)

	err = state.Initialize(genesisData, genesisHeader, tr, firstEpochInfo)
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

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), trie.EmptyHash, trie.EmptyHash, [][]byte{})
	require.NoError(t, err)

	genesisData := new(genesis.Data)

	tr := trie.NewEmptyTrie()
	err = stateA.Initialize(genesisData, genesisHeader, tr, firstEpochInfo)
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

	if !reflect.DeepEqual(stateA.Block.BestBlockHash(), stateB.Block.BestBlockHash()) {
		t.Fatalf("Fail: got %s expected %s", stateA.Block.BestBlockHash(), stateB.Block.BestBlockHash())
	}
}
func Test_ServicePruneStorage(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	serv := NewService(testDir, log.LvlTrace)
	serv.UseMemDB()

	genesisData := new(genesis.Data)

	tr := trie.NewEmptyTrie()
	err := serv.Initialize(genesisData, testGenesisHeader, tr, firstEpochInfo)
	require.NoError(t, err)

	err = serv.Start()
	require.NoError(t, err)

	type prunedBlock struct {
		hash  common.Hash
		dbKey []byte
	}

	var prunedArr []prunedBlock
	for i := 0; i < 3; i++ {
		block, trieState := generateBlockWithRandomTrie(t, serv)

		err = serv.Storage.blockState.AddBlock(block)
		require.NoError(t, err)

		err = serv.Storage.StoreTrie(block.Header.StateRoot, trieState)
		require.NoError(t, err)

		// Only finalize the head block.
		if i == 2 {
			serv.Block.SetFinalizedHash(block.Header.Hash(), 0, 0)
			break
		}

		// Store the other blocks that will be pruned.
		var trieVal *trie.Trie
		trieVal, err = trieState.t.DeepCopy()
		require.NoError(t, err)

		var rootHash common.Hash
		rootHash, err = trieVal.Hash()
		require.NoError(t, err)

		prunedArr = append(prunedArr, prunedBlock{hash: block.Header.StateRoot, dbKey: rootHash[:]})
	}

	time.Sleep(1 * time.Second)

	for _, v := range prunedArr {
		serv.Storage.lock.Lock()
		_, ok := serv.Storage.tries[v.hash]
		serv.Storage.lock.Unlock()
		require.Equal(t, false, ok)

		ok, err = serv.Storage.baseDB.Has(v.dbKey)
		require.NoError(t, err)
		require.Equal(t, false, ok)
	}

	err = serv.Stop()
	require.NoError(t, err)
}
