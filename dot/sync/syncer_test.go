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

package sync

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"

	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
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

func newMockFinalityGadget() *syncmocks.FinalityGadget {
	m := new(syncmocks.FinalityGadget)
	// using []uint8 instead of []byte: https://github.com/stretchr/testify/pull/969
	m.On("VerifyBlockJustification", mock.AnythingOfType("common.Hash"), mock.AnythingOfType("[]uint8")).Return(nil)
	return m
}

func newMockVerifier() *syncmocks.MockVerifier {
	m := new(syncmocks.MockVerifier)
	m.On("VerifyBlock", mock.AnythingOfType("*types.Header")).Return(nil)
	return m
}

func newMockNetwork() *syncmocks.MockNetwork {
	m := new(syncmocks.MockNetwork)
	m.On("DoBlockRequest", mock.AnythingOfType("peer.ID"), mock.AnythingOfType("*BlockRequestMessage")).Return(nil, nil)
	return m
}

func newTestSyncer(t *testing.T, usePolkadotGenesis bool) *Service {
	wasmer.DefaultTestLogLvl = 3

	cfg := &Config{}
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")

	scfg := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.LvlInfo,
	}
	stateSrvc := state.NewService(scfg)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t, usePolkadotGenesis)
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

	// initialize runtime
	genState, err := rtstorage.NewTrieState(genTrie) //nolint
	require.NoError(t, err)

	rtCfg := &wasmer.Config{}
	rtCfg.Storage = genState
	rtCfg.LogLvl = 3

	rtCfg.CodeHash, err = cfg.StorageState.LoadCodeHash(nil)
	require.NoError(t, err)

	instance, err := wasmer.NewRuntimeFromGenesis(gen, rtCfg)
	require.NoError(t, err)

	cfg.BlockState.StoreRuntime(cfg.BlockState.BestBlockHash(), instance)

	cfg.BlockImportHandler = new(syncmocks.MockBlockImportHandler)
	cfg.BlockImportHandler.(*syncmocks.MockBlockImportHandler).On("HandleBlockImport", mock.AnythingOfType("*types.Block"), mock.AnythingOfType("*storage.TrieState")).Return(func(block *types.Block, ts *rtstorage.TrieState) error {
		// store updates state trie nodes in database
		err := stateSrvc.Storage.StoreTrie(ts, block.Header)
		if err != nil {
			logger.Warn("failed to store state trie for imported block", "block", block.Header.Hash(), "error", err)
			return err
		}

		// store block in database
		if err = stateSrvc.Block.AddBlock(block); err != nil {
			if err == blocktree.ErrParentNotFound && block.Header.Number.Cmp(big.NewInt(0)) != 0 {
				return err
			} else if err == blocktree.ErrBlockExists || block.Header.Number.Cmp(big.NewInt(0)) == 0 {
				// this is fine
			} else {
				return err
			}
		}

		stateSrvc.Block.StoreRuntime(block.Header.Hash(), instance)
		logger.Debug("imported block and stored state trie", "block", block.Header.Hash(), "state root", ts.MustRoot())
		return nil
	})

	cfg.TransactionState = stateSrvc.Transaction
	cfg.BabeVerifier = newMockVerifier()
	cfg.LogLvl = log.LvlTrace
	cfg.FinalityGadget = newMockFinalityGadget()
	cfg.Network = newMockNetwork()

	syncer, err := NewService(cfg)
	require.NoError(t, err)
	return syncer
}

func newTestGenesisWithTrieAndHeader(t *testing.T, usePolkadotGenesis bool) (*genesis.Genesis, *trie.Trie, *types.Header) {
	fp := "../../chain/gssmr/genesis.json"
	if usePolkadotGenesis {
		fp = "../../chain/polkadot/genesis.json"
	}

	gen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), genTrie.MustHash(), trie.EmptyHash, big.NewInt(0), types.Digest{})
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
}
