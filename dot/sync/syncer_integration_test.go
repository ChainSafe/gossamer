//go:build integration
// +build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newMockNetwork() *mocks.Network {
	m := new(mocks.Network)
	m.On("DoBlockRequest", mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.BlockRequestMessage")).Return(nil, nil)
	return m
}

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client
func newTestSyncer(t *testing.T) *Service {
	ctrl := gomock.NewController(t)

	mockTelemetryClient := NewMockClient(ctrl)
	mockTelemetryClient.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	wasmer.DefaultTestLogLvl = log.Warn

	cfg := &Config{}
	testDatadirPath := t.TempDir()

	scfg := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Info,
		Telemetry: mockTelemetryClient,
	}
	stateSrvc := state.NewService(scfg)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(gen, genHeader, genTrie)
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
	genState, err := rtstorage.NewTrieState(genTrie)
	require.NoError(t, err)

	rtCfg := &wasmer.Config{}
	rtCfg.Storage = genState
	rtCfg.LogLvl = 3
	rtCfg.NodeStorage = runtime.NodeStorage{}

	if stateSrvc != nil {
		rtCfg.NodeStorage.BaseDB = stateSrvc.Base
	} else {
		rtCfg.NodeStorage.BaseDB, err = utils.SetupDatabase(filepath.Join(testDatadirPath, "offline_storage"), false)
		require.NoError(t, err)
	}

	rtCfg.CodeHash, err = cfg.StorageState.LoadCodeHash(nil)
	require.NoError(t, err)

	instance, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	cfg.BlockState.StoreRuntime(cfg.BlockState.BestBlockHash(), instance)

	cfg.BlockImportHandler = new(mocks.BlockImportHandler)
	cfg.BlockImportHandler.(*mocks.BlockImportHandler).On(
		"HandleBlockImport", mock.AnythingOfType("*types.Block"), mock.AnythingOfType("*storage.TrieState")).
		Return(func(block *types.Block, ts *rtstorage.TrieState) error {
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
				block.Header.Hash(), ts.MustRoot())
			return nil
		})

	cfg.TransactionState = stateSrvc.Transaction
	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockBabeVerifier.EXPECT().VerifyBlock(gomock.AssignableToTypeOf(&types.Header{})).AnyTimes()
	cfg.BabeVerifier = mockBabeVerifier
	cfg.LogLvl = log.Trace
	mockFinalityGadget := NewMockFinalityGadget(ctrl)
	mockFinalityGadget.EXPECT().VerifyBlockJustification(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf([]byte{})).DoAndReturn(func(hash common.Hash, justification []byte) ([]byte, error) {
		return justification, nil
	}).AnyTimes()

	cfg.FinalityGadget = mockFinalityGadget
	cfg.Network = newMockNetwork()
	cfg.Telemetry = mockTelemetryClient
	syncer, err := NewService(cfg)
	require.NoError(t, err)
	return syncer
}

func newTestGenesisWithTrieAndHeader(t *testing.T) (*genesis.Genesis, *trie.Trie, *types.Header) {
	fp := utils.GetGssmrGenesisRawPathTest(t)
	gen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}),
		genTrie.MustHash(), trie.EmptyHash, 0, types.NewDigest())
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
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
			name: "when *chainSync.getHighestBlock() returns 0, error should return 0",
			in: input{
				highestBlock: 0,
				err:          errors.New("fake error"),
			},
			out: output{
				highestBlock: 0,
			},
		},
		{
			name: "when *chainSync.getHighestBlock() returns 0, nil should return 0",
			in: input{
				highestBlock: 0,
				err:          nil,
			},
			out: output{
				highestBlock: 0,
			},
		},
		{
			name: "when *chainSync.getHighestBlock() returns 50, nil should return 50",
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
