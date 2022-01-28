// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"

	"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/require"
)

// helper method to create and start test state service
func newTestService(t *testing.T) (state *Service) {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := Config{
		Path:      t.TempDir(),
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}
	state = NewService(config)
	return state
}

func newTestMemDBService(t *testing.T) *Service {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	testDatadirPath := t.TempDir()
	config := Config{
		Path:      testDatadirPath,
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}
	state := NewService(config)
	state.UseMemDB()
	return state
}

func TestService_Start(t *testing.T) {
	state := newTestService(t)

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := state.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state.SetupBase()
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	err = state.Stop()
	require.NoError(t, err)
}

func TestService_Initialise(t *testing.T) {
	state := newTestService(t)

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := state.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	genesisHeader, err = types.NewHeader(common.NewHash([]byte{77}),
		genTrie.MustHash(), trie.EmptyHash, big.NewInt(0), types.NewDigest())
	require.NoError(t, err)

	err = state.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state.SetupBase()
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	head, err := state.Block.BestBlockHeader()
	require.NoError(t, err)
	require.Equal(t, genesisHeader, head)
}

func TestMemDB_Start(t *testing.T) {
	state := newTestMemDBService(t)

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := state.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = state.Start()
	require.NoError(t, err)

	err = state.Stop()
	require.NoError(t, err)
}

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client

func TestService_BlockTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().
		SendMessage(gomock.AssignableToTypeOf(&telemetry.NotifyFinalized{})).
		MaxTimes(2)

	config := Config{
		Path:      t.TempDir(),
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}

	stateA := NewService(config)

	genData, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := stateA.Initialise(genData, genesisHeader, genTrie)
	require.NoError(t, err)

	err = stateA.SetupBase()
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

	err = stateB.SetupBase()
	require.NoError(t, err)

	err = stateB.Start()
	require.NoError(t, err)

	err = stateB.Stop()
	require.NoError(t, err)
	require.Equal(t, stateA.Block.BestBlockHash(), stateB.Block.BestBlockHash())
}

func TestService_StorageTriePruning(t *testing.T) {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	retainBlocks := 2
	config := Config{
		Path:     t.TempDir(),
		LogLevel: log.Info,
		PrunerCfg: pruner.Config{
			Mode:           pruner.Full,
			RetainedBlocks: int64(retainBlocks),
		},
		Telemetry: telemetryMock,
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
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := Config{
		Path:      t.TempDir(),
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
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
		prd, err := types.NewBabeSecondaryPlainPreDigest(0, uint64(i+1)).ToPreRuntimeDigest()
		require.NoError(t, err)
		err = digest.Add(*prd)
		require.NoError(t, err)
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
		copiedTrie := trieState.Trie().DeepCopy()

		var rootHash common.Hash
		rootHash, err = copiedTrie.Hash()
		require.NoError(t, err)

		prunedArr = append(prunedArr, prunedBlock{hash: block.Header.StateRoot, dbKey: rootHash[:]})
		parentHash = block.Header.Hash()
	}

	// finalise a block
	serv.Block.SetFinalisedHash(toFinalize, 0, 0)

	time.Sleep(1 * time.Second)

	for _, v := range prunedArr {
		tr := serv.Storage.tries.get(v.hash)
		require.Nil(t, tr)
	}
}

func TestService_Rewind(t *testing.T) {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := Config{
		Path:      t.TempDir(),
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
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
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := Config{
		Path:      t.TempDir(),
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
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
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 177).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)
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
