//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestSyncer(t *testing.T) *Service {
	ctrl := gomock.NewController(t)

	mockTelemetryClient := NewMockTelemetry(ctrl)
	mockTelemetryClient.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	wazero_runtime.DefaultTestLogLvl = log.Warn

	cfg := &Config{}
	testDatadirPath := t.TempDir()

	scfg := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Info,
		Telemetry: mockTelemetryClient,
	}
	stateSrvc := state.NewService(scfg)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(&gen, &genHeader, &genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	// initialise runtime
	genState := rtstorage.NewTransactionalTrieState(&genTrie)

	rtCfg := wazero_runtime.Config{
		Storage: genState,
		LogLvl:  log.Critical,
	}

	if stateSrvc != nil {
		rtCfg.NodeStorage.BaseDB = stateSrvc.Base
	} else {
		rtCfg.NodeStorage.BaseDB, err = database.LoadDatabase(filepath.Join(testDatadirPath, "offline_storage"), false)
		require.NoError(t, err)
	}

	rtCfg.CodeHash, err = cfg.StorageState.(*state.StorageState).LoadCodeHash(nil)
	require.NoError(t, err)

	instance, err := wazero_runtime.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	bestBlockHash := cfg.BlockState.(*state.BlockState).BestBlockHash()
	cfg.BlockState.(*state.BlockState).StoreRuntime(bestBlockHash, instance)
	blockImportHandler := NewMockBlockImportHandler(ctrl)
	blockImportHandler.EXPECT().HandleBlockImport(gomock.AssignableToTypeOf(&types.Block{}),
		gomock.AssignableToTypeOf(&rtstorage.TransactionalTrieState{}), false).DoAndReturn(
		func(block *types.Block, ts *rtstorage.TransactionalTrieState, _ bool) error {
			// store updates state trie nodes in database
			if err = stateSrvc.Storage.StoreTrie(ts, &block.Header); err != nil {
				logger.Warnf("failed to store state trie for imported block %s: %s", block.Header.Hash(), err)
				return err
			}

			// store block in database
			err = stateSrvc.Block.AddBlock(block)
			require.NoError(t, err)

			stateSrvc.Block.StoreRuntime(block.Header.Hash(), instance)
			logger.Debugf("imported block %s and stored state trie with root %s",
				block.Header.Hash(), ts.MustRoot(trie.NoMaxInlineValueSize))
			return nil
		}).AnyTimes()
	cfg.BlockImportHandler = blockImportHandler

	cfg.TransactionState = stateSrvc.Transaction
	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockBabeVerifier.EXPECT().VerifyBlock(gomock.AssignableToTypeOf(&types.Header{})).AnyTimes()
	cfg.BabeVerifier = mockBabeVerifier
	cfg.LogLvl = log.Trace
	mockFinalityGadget := NewMockFinalityGadget(ctrl)
	mockFinalityGadget.EXPECT().VerifyBlockJustification(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf([]byte{})).DoAndReturn(func(hash common.Hash, justification []byte) error {
		return nil
	}).AnyTimes()

	cfg.FinalityGadget = mockFinalityGadget
	cfg.Network = NewMockNetwork(ctrl)
	cfg.Telemetry = mockTelemetryClient
	cfg.RequestMaker = NewMockRequestMaker(ctrl)
	syncer, err := NewService(cfg)
	require.NoError(t, err)
	return syncer
}

func newWestendDevGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendDevRawGenesisPath(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = runtime.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := genesisTrie.MustHash(trie.NoMaxInlineValueSize)
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeaderPtr := types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)
	genesisHeader = *genesisHeaderPtr

	return gen, genesisTrie, genesisHeader
}

func TestHighestBlock(t *testing.T) {
	type input struct {
		highestBlock uint
		err          error
	}
	type output struct {
		highestBlock uint
	}
	type test struct {
		name string
		in   input
		out  output
	}
	tests := []test{
		{
			name: "when_*chainSync.getHighestBlock()_returns_0,_error_should_return_0",
			in: input{
				highestBlock: 0,
				err:          errors.New("fake error"),
			},
			out: output{
				highestBlock: 0,
			},
		},
		{
			name: "when_*chainSync.getHighestBlock()_returns_0,_nil_should_return_0",
			in: input{
				highestBlock: 0,
				err:          nil,
			},
			out: output{
				highestBlock: 0,
			},
		},
		{
			name: "when_*chainSync.getHighestBlock()_returns_50,_nil_should_return_50",
			in: input{
				highestBlock: 50,
				err:          nil,
			},
			out: output{
				highestBlock: 50,
			},
		},
	}
	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			s := newTestSyncer(t)

			ctrl := gomock.NewController(t)
			chainSync := NewMockChainSync(ctrl)
			chainSync.EXPECT().getHighestBlock().Return(ts.in.highestBlock, ts.in.err)

			s.chainSync = chainSync

			result := s.HighestBlock()
			require.Equal(t, result, ts.out.highestBlock)
		})
	}
}
