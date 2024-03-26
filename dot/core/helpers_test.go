// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func balanceKey(t *testing.T, pub []byte) (bKey []byte) {
	t.Helper()

	h0, err := common.Twox128Hash([]byte("System"))
	require.NoError(t, err)
	h1, err := common.Twox128Hash([]byte("Account"))
	require.NoError(t, err)
	h2, err := common.Blake2b128(pub)
	require.NoError(t, err)
	return bytes.Join([][]byte{h0, h1, h2, pub}, nil)
}

// Creates test service, used now for testing txnPool but can be used elsewhere when needed
func createTestService(t *testing.T, genesisFilePath string,
	pubKey []byte, accountInfo types.AccountInfo, ctrl *gomock.Controller) (service *Service, encodedExtrinsic []byte) {
	t.Helper()

	gen, err := genesis.NewGenesisFromJSONRaw(genesisFilePath)
	require.NoError(t, err)

	genesisTrie, err := runtime.NewInMemoryTrieFromGenesis(*gen)
	require.NoError(t, err)

	// Extrinsic and context related stuff
	aliceBalanceKey := balanceKey(t, pubKey)
	encodedAccountInfo, err := scale.Marshal(accountInfo)
	require.NoError(t, err)

	genesisHeader := &types.Header{
		StateRoot: trie.V0.MustHash(genesisTrie),
		Number:    0,
	}

	cfgKeystore := keystore.NewGlobalKeystore()
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)
	err = cfgKeystore.Acco.Insert(kp)
	require.NoError(t, err)

	// Create state service
	var stateSrvc *state.Service
	testDatadirPath := t.TempDir()

	// Set up block and storage state
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	stateConfig := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Critical,
		Telemetry: telemetryMock,
	}

	stateSrvc = state.NewService(stateConfig)
	stateSrvc.UseMemDB()

	err = stateSrvc.Initialise(gen, genesisHeader, genesisTrie)
	require.NoError(t, err)

	// Start state service
	err = stateSrvc.Start()
	require.NoError(t, err)

	cfgBlockState := stateSrvc.Block
	cfgStorageState := stateSrvc.Storage
	cfgCodeSubstitutedState := stateSrvc.Base

	var rtCfg wazero_runtime.Config
	rtCfg.Storage = rtstorage.NewTrieState(genesisTrie)

	rtCfg.CodeHash, err = cfgStorageState.LoadCodeHash(nil)
	require.NoError(t, err)

	nodeStorage := runtime.NodeStorage{}
	nodeStorage.BaseDB = stateSrvc.Base

	rtCfg.NodeStorage = nodeStorage

	cfgRuntime, err := wazero_runtime.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	cfgRuntime.Context.Storage.Put(aliceBalanceKey, encodedAccountInfo)
	// this key is System.UpgradedToDualRefCount -> set to true since all accounts have been upgraded to v0.9 format
	cfgRuntime.Context.Storage.Put(common.UpgradedToDualRefKey, []byte{1})

	cfgBlockState.StoreRuntime(cfgBlockState.BestBlockHash(), cfgRuntime)

	// Hash of encrypted centrifuge extrinsic
	testCallArguments := []byte{0xab, 0xcd}
	extHex := runtime.NewTestExtrinsic(t, cfgRuntime, genesisHeader.Hash(), cfgBlockState.BestBlockHash(),
		0, signature.TestKeyringPairAlice, "System.remark", testCallArguments)
	encodedExtrinsic = common.MustHexToBytes(extHex)

	cfgCodeSubstitutes := make(map[common.Hash]string)

	genesisData, err := cfgCodeSubstitutedState.LoadGenesisData()
	require.NoError(t, err)

	for k, v := range genesisData.CodeSubstitutes {
		cfgCodeSubstitutes[common.MustHexToHash(k)] = v
	}

	cfgCodeSubstitutedState = stateSrvc.Base

	cfg := &Config{
		Keystore:             cfgKeystore,
		LogLvl:               log.Critical,
		BlockState:           cfgBlockState,
		StorageState:         cfgStorageState,
		TransactionState:     stateSrvc.Transaction,
		CodeSubstitutedState: cfgCodeSubstitutedState,
		Runtime:              cfgRuntime,
		Network:              new(network.Service),
		CodeSubstitutes:      cfgCodeSubstitutes,
	}
	service, err = NewService(cfg)
	require.NoError(t, err)

	return service, encodedExtrinsic
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

	gen, genesisTrie, genesisHeader := newWestendLocalWithTrieAndHeader(t)

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

		err := stateSrvc.Initialise(&gen, &genesisHeader, genesisTrie)
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
		var rtCfg wazero_runtime.Config

		rtCfg.Storage = rtstorage.NewTrieState(genesisTrie)

		var err error
		rtCfg.CodeHash, err = cfg.StorageState.(*state.InmemoryStorageState).LoadCodeHash(nil)
		require.NoError(t, err)

		nodeStorage := runtime.NodeStorage{}

		if stateSrvc != nil {
			nodeStorage.BaseDB = stateSrvc.Base
		} else {
			nodeStorage.BaseDB, err = database.LoadDatabase(filepath.Join(testDatadirPath, "offline_storage"), false)
			require.NoError(t, err)
		}

		rtCfg.NodeStorage = nodeStorage

		cfg.Runtime, err = wazero_runtime.NewRuntimeFromGenesis(rtCfg)
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

func newWestendLocalWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie *trie.InMemoryTrie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendLocalRawGenesisPath(t)
	genPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genPtr

	genesisTrie, err = runtime.NewInMemoryTrieFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := trie.V0.MustHash(genesisTrie)
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeader = *types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)

	return gen, genesisTrie, genesisHeader
}

func getWestendDevRuntimeCode(t *testing.T) (code []byte) {
	t.Helper()

	path := utils.GetWestendDevRawGenesisPath(t)
	westendDevGenesis, err := genesis.NewGenesisFromJSONRaw(path)
	require.NoError(t, err)

	genesisTrie, err := runtime.NewInMemoryTrieFromGenesis(*westendDevGenesis)
	require.NoError(t, err)

	trieState := rtstorage.NewTrieState(genesisTrie)

	return trieState.LoadCode()
}
