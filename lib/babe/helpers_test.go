// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// NewTestService creates a new test core service
func NewTestService(t *testing.T, cfg *core.Config) *core.Service {
	t.Helper()
	ctrl := gomock.NewController(t)

	if cfg == nil {
		cfg = &core.Config{}
	}

	if cfg.Keystore == nil {
		cfg.Keystore = keystore.NewGlobalKeystore()
		kp, err := sr25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		err = cfg.Keystore.Acco.Insert(kp)
		require.NoError(t, err)
	}

	cfg.LogLvl = 3

	var stateSrvc *state.Service
	testDatadirPath := t.TempDir()

	gen, genesisTrie, genesisHeader := newTestGenesisWithTrieAndHeader(t)

	if cfg.BlockState == nil || cfg.StorageState == nil ||
		cfg.TransactionState == nil || cfg.CodeSubstitutedState == nil {
		telemetryMock := NewMockTelemetry(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

		config := state.Config{
			Path:      testDatadirPath,
			LogLevel:  log.Info,
			Telemetry: telemetryMock,
		}

		stateSrvc = state.NewService(config)
		stateSrvc.UseMemDB()

		err := stateSrvc.Initialise(&gen, &genesisHeader, &genesisTrie)
		require.NoError(t, err)

		err = stateSrvc.Start()
		require.NoError(t, err)
	}

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	if cfg.TransactionState == nil {
		cfg.TransactionState = stateSrvc.Transaction
	}

	if cfg.CodeSubstitutedState == nil {
		cfg.CodeSubstitutedState = stateSrvc.Base
	}

	if cfg.Runtime == nil {
		var rtCfg wasmer.Config

		rtCfg.Storage = rtstorage.NewTrieState(&genesisTrie)

		var err error
		rtCfg.CodeHash, err = cfg.StorageState.(*state.StorageState).LoadCodeHash(nil)
		require.NoError(t, err)

		nodeStorage := runtime.NodeStorage{}

		if stateSrvc != nil {
			nodeStorage.BaseDB = stateSrvc.Base
		} else {
			nodeStorage.BaseDB, err = utils.SetupDatabase(filepath.Join(testDatadirPath, "offline_storage"), false)
			require.NoError(t, err)
		}

		rtCfg.NodeStorage = nodeStorage

		cfg.Runtime, err = wasmer.NewRuntimeFromGenesis(rtCfg)
		require.NoError(t, err)
	}
	cfg.BlockState.StoreRuntime(cfg.BlockState.BestBlockHash(), cfg.Runtime)

	if cfg.CodeSubstitutes == nil {
		cfg.CodeSubstitutes = make(map[common.Hash]string)

		genesisData, err := cfg.CodeSubstitutedState.(*state.BaseState).LoadGenesisData()
		require.NoError(t, err)

		for k, v := range genesisData.CodeSubstitutes {
			cfg.CodeSubstitutes[common.MustHexToHash(k)] = v
		}
	}

	if cfg.CodeSubstitutedState == nil {
		cfg.CodeSubstitutedState = stateSrvc.Base
	}

	s, err := core.NewService(cfg)
	require.NoError(t, err)

	return s
}

// newDevGenesisWithTrieAndHeader generates test dev genesis, genesis trie and genesis header
func newDevGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header) {
	t.Helper()
	genesisPath := utils.GetDevV3SubstrateGenesisPath(t)

	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = wasmer.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader = *types.NewHeader(common.NewHash([]byte{0}),
		genesisTrie.MustHash(), trie.EmptyHash, 0, types.NewDigest())

	return gen, genesisTrie, genesisHeader
}

func newTestGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetGssmrV3SubstrateGenesisRawPathTest(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = wasmer.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := genesisTrie.MustHash()
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeader = *types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)

	return gen, genesisTrie, genesisHeader
}
