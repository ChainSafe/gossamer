// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
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
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func balanceKey(t *testing.T, pub []byte) (bKey []byte) {
	t.Helper()

	h0, err := common.Twox128Hash([]byte("System"))
	require.NoError(t, err)
	h1, err := common.Twox128Hash([]byte("Account"))
	require.NoError(t, err)
	h2, err := common.Blake2b128(pub)
	require.NoError(t, err)
	bytes := [][]byte{h0, h1, h2, pub}
	return concatenateByteSlices(bytes)
}

// Note might need to pass in Config but will see
// CreateTestService is a new helper function to clean up orchestration for testing the txnPool
func CreateTestService(t *testing.T, genesisFilePath string,
	pubKey []byte, accInfo types.AccountInfo, ctrl *gomock.Controller) (s *Service, encExtrinsic []byte) {
	t.Helper()

	// Create genesis
	gen, err := genesis.NewGenesisFromJSONRaw(genesisFilePath)
	require.NoError(t, err)

	// Trie Created
	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	// Extrinsic and context related stuff
	aliceBalanceKey := balanceKey(t, pubKey)
	encodedAccountInfo, err := scale.Marshal(accInfo)
	require.NoError(t, err)

	genesisHeader := &types.Header{
		StateRoot: genTrie.MustHash(),
		Number:    0,
	}

	cfg := &Config{}
	if cfg.Keystore == nil {
		cfg.Keystore = keystore.NewGlobalKeystore()
		kp, err := sr25519.GenerateKeypair()
		require.NoError(t, err)
		err = cfg.Keystore.Acco.Insert(kp)
		require.NoError(t, err)
	}

	cfg.LogLvl = log.Critical

	// Create state service
	var stateSrvc *state.Service
	testDatadirPath := t.TempDir()

	// Set up block and storage state
	if cfg.BlockState == nil || cfg.StorageState == nil ||
		cfg.TransactionState == nil || cfg.EpochState == nil ||
		cfg.CodeSubstitutedState == nil {
		telemetryMock := NewMockClient(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

		config := state.Config{
			Path:      testDatadirPath,
			LogLevel:  log.Info,
			Telemetry: telemetryMock,
		}

		stateSrvc = state.NewService(config)
		stateSrvc.UseMemDB()

		err := stateSrvc.Initialise(gen, genesisHeader, genTrie)
		require.NoError(t, err)

		// Start state service
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

	if cfg.EpochState == nil {
		cfg.EpochState = stateSrvc.Epoch
	}

	if cfg.CodeSubstitutedState == nil {
		cfg.CodeSubstitutedState = stateSrvc.Base
	}

	// Runtime stuff
	if cfg.Runtime == nil {
		// Okay no errors, but test doesn't pass
		var rtCfg wasmer.Config
		rtCfg.Storage = rtstorage.NewTrieState(genTrie)

		var err error
		rtCfg.CodeHash, err = cfg.StorageState.LoadCodeHash(nil)
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

		cfg.Runtime.(*wasmer.Instance).GetContext().Storage.Set(aliceBalanceKey, encodedAccountInfo)
		// this key is System.UpgradedToDualRefCount -> set to true since all accounts have been upgraded to v0.9 format
		cfg.Runtime.(*wasmer.Instance).GetContext().Storage.Set(common.UpgradedToDualRefKey, []byte{1})
	}
	cfg.BlockState.StoreRuntime(cfg.BlockState.BestBlockHash(), cfg.Runtime)

	// Hash of encrypted centrifuge extrinsic
	testCallArguments := []byte{0xab, 0xcd}
	extHex := runtime.NewTestExtrinsic(t, cfg.Runtime, genesisHeader.Hash(), cfg.BlockState.BestBlockHash(),
		0, "System.remark", testCallArguments)
	encExtrinsic = common.MustHexToBytes(extHex)

	if cfg.Network == nil {
		cfg.Network = new(network.Service) // only for nil check in NewService
	}

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

	s, err = NewService(cfg)
	require.NoError(t, err)

	return s, encExtrinsic
}

// NewTestService creates a new test core service
func NewTestService(t *testing.T, cfg *Config) *Service {
	t.Helper()
	ctrl := gomock.NewController(t)

	if cfg == nil {
		cfg = &Config{}
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

	gen, genTrie, genHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	if cfg.BlockState == nil || cfg.StorageState == nil ||
		cfg.TransactionState == nil || cfg.EpochState == nil ||
		cfg.CodeSubstitutedState == nil {
		telemetryMock := NewMockClient(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

		config := state.Config{
			Path:      testDatadirPath,
			LogLevel:  log.Info,
			Telemetry: telemetryMock,
		}

		stateSrvc = state.NewService(config)
		stateSrvc.UseMemDB()

		err := stateSrvc.Initialise(gen, genHeader, genTrie)
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

	if cfg.EpochState == nil {
		cfg.EpochState = stateSrvc.Epoch
	}

	if cfg.CodeSubstitutedState == nil {
		cfg.CodeSubstitutedState = stateSrvc.Base
	}

	if cfg.Runtime == nil {
		var rtCfg wasmer.Config

		rtCfg.Storage = rtstorage.NewTrieState(genTrie)

		var err error
		rtCfg.CodeHash, err = cfg.StorageState.LoadCodeHash(nil)
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

	s, err := NewService(cfg)
	require.NoError(t, err)

	return s
}

func getGssmrRuntimeCode(t *testing.T) (code []byte) {
	t.Helper()

	path, err := utils.GetGssmrGenesisRawPath()
	require.NoError(t, err)

	gssmrGenesis, err := genesis.NewGenesisFromJSONRaw(path)
	require.NoError(t, err)

	trie, err := genesis.NewTrieFromGenesis(gssmrGenesis)
	require.NoError(t, err)

	trieState := rtstorage.NewTrieState(trie)

	return trieState.LoadCode()
}

func hashPtr(h common.Hash) *common.Hash { return &h }
