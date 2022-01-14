// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/mock"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/sync/mocks"
)

func TestMain(m *testing.M) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Errorf("failed to generate runtime wasm file: %s", err)
		os.Exit(1)
	}

	// Start all tests
	code := m.Run()

	runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

type syncAPIMock struct {
	fnGetHighestBlock func() (int64, error)
}

func (_m syncAPIMock) start()                                                         {}
func (_m syncAPIMock) stop()                                                          {}
func (_m syncAPIMock) setBlockAnnounce(from peer.ID, header *types.Header) error      { return nil }
func (_m syncAPIMock) setPeerHead(p peer.ID, hash common.Hash, number *big.Int) error { return nil }
func (_m syncAPIMock) syncState() chainSyncState                                      { return 0 }
func (_m syncAPIMock) getHighestBlock() (int64, error)                                { return _m.fnGetHighestBlock() }

func newMockFinalityGadget() *mocks.FinalityGadget {
	m := new(mocks.FinalityGadget)
	// using []uint8 instead of []byte: https://github.com/stretchr/testify/pull/969
	m.On("VerifyBlockJustification", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil)
	return m
}

func newMockBabeVerifier() *mocks.BabeVerifier {
	m := new(mocks.BabeVerifier)
	m.On("VerifyBlock", mock.AnythingOfType("*types.Header")).Return(nil)
	return m
}

func newMockNetwork() *mocks.Network {
	m := new(mocks.Network)
	m.On("DoBlockRequest", mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.BlockRequestMessage")).Return(nil, nil)
	return m
}

func newTestSyncer(t *testing.T) *Service {
	wasmer.DefaultTestLogLvl = 3

	cfg := &Config{}
	testDatadirPath := t.TempDir()

	scfg := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.Info,
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
	cfg.BabeVerifier = newMockBabeVerifier()
	cfg.LogLvl = log.Trace
	cfg.FinalityGadget = newMockFinalityGadget()
	cfg.Network = newMockNetwork()

	syncer, err := NewService(cfg)
	require.NoError(t, err)
	return syncer
}

func newTestGenesisWithTrieAndHeader(t *testing.T) (*genesis.Genesis, *trie.Trie, *types.Header) {
	fp := "../../chain/gssmr/genesis.json"
	gen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}),
		genTrie.MustHash(), trie.EmptyHash, big.NewInt(0), types.NewDigest())
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
}

func TestHighestBlock(t *testing.T) {
	type input struct {
		highestBlock int64
		err          error
	}
	type output struct {
		highestBlock int64
	}
	type test struct {
		name string
		in   input
		out  output
	}
	tests := []test{
		{
			name: "when *chainSync.getHighestBlock() returns error should return 0",
			in: input{
				highestBlock: 0,
				err:          errors.New("fake error"),
			},
			out: output{
				highestBlock: 0,
			},
		},
		{
			name: "when *chainSync.getHighestBlock() returns 0 should return 0",
			in: input{
				highestBlock: 0,
				err:          nil,
			},
			out: output{
				highestBlock: 0,
			},
		},
		{
			name: "when *chainSync.getHighestBlock() returns 50 should return 50",
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

			chainSync := syncAPIMock{}
			chainSync.fnGetHighestBlock = func() (int64, error) {
				return ts.in.highestBlock, ts.in.err
			}

			s.chainSync = chainSync
			result := s.HighestBlock()
			require.Equalf(t, result, ts.out.highestBlock, ts.name)
		})
	}
}
