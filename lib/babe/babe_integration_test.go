// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration
// +build integration

package babe

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/babe/mocks"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"

	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	defaultTestLogLvl = log.Info
	emptyHash         = trie.EmptyHash
	testEpochIndex    = uint64(0)
	maxThreshold      = scale.MaxUint128

	genesisHeader *types.Header
	emptyHeader   = &types.Header{
		Number: big.NewInt(0),
		Digest: types.NewDigest(),
	}

	genesisBABEConfig = &types.BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: []types.AuthorityRaw{},
		Randomness:         [32]byte{},
		SecondarySlots:     0,
	}
)

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client

func createTestService(t *testing.T, cfg *ServiceConfig) *Service {
	wasmer.DefaultTestLogLvl = 1

	gen, genTrie, genHeader := genesis.NewDevGenesisWithTrieAndHeader(t)
	genesisHeader = genHeader

	var err error

	if cfg == nil {
		cfg = &ServiceConfig{
			Authority: true,
		}
	}

	cfg.BlockImportHandler = new(mocks.BlockImportHandler)
	cfg.BlockImportHandler.(*mocks.BlockImportHandler).
		On("HandleBlockProduced",
			mock.AnythingOfType("*types.Block"), mock.AnythingOfType("*storage.TrieState")).
		Return(nil)

	if cfg.Keypair == nil {
		cfg.Keypair = keyring.Alice().(*sr25519.Keypair)
	}

	if cfg.AuthData == nil {
		auth := types.Authority{
			Key:    cfg.Keypair.Public().(*sr25519.PublicKey),
			Weight: 1,
		}
		cfg.AuthData = []types.Authority{auth}
	}

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	cfg.Telemetry = telemetryMock

	if cfg.TransactionState == nil {
		cfg.TransactionState = state.NewTransactionState(telemetryMock)
	}

	testDatadirPath := t.TempDir()
	require.NoError(t, err)

	var dbSrv *state.Service
	if cfg.BlockState == nil || cfg.StorageState == nil || cfg.EpochState == nil {
		config := state.Config{
			Path:      testDatadirPath,
			LogLevel:  log.Info,
			Telemetry: telemetryMock,
		}
		dbSrv = state.NewService(config)
		dbSrv.UseMemDB()

		err = dbSrv.Initialise(gen, genHeader, genTrie)
		require.NoError(t, err)

		err = dbSrv.Start()
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = dbSrv.Stop()
		})

		cfg.BlockState = dbSrv.Block
		cfg.StorageState = dbSrv.Storage
		cfg.EpochState = dbSrv.Epoch
	}

	rtCfg := &wasmer.Config{}
	rtCfg.Storage, err = rtstorage.NewTrieState(genTrie)
	require.NoError(t, err)

	storageState := cfg.StorageState.(core.StorageState)
	rtCfg.CodeHash, err = storageState.LoadCodeHash(nil)
	require.NoError(t, err)

	nodeStorage := runtime.NodeStorage{}
	if dbSrv != nil {
		nodeStorage.BaseDB = dbSrv.Base
	} else {
		nodeStorage.BaseDB, err = utils.SetupDatabase(filepath.Join(testDatadirPath, "offline_storage"), false)
		require.NoError(t, err)
	}

	rtCfg.NodeStorage = nodeStorage
	rt, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)
	cfg.BlockState.StoreRuntime(cfg.BlockState.BestBlockHash(), rt)

	cfg.IsDev = true
	cfg.LogLvl = defaultTestLogLvl
	babeService, err := NewService(cfg)
	require.NoError(t, err)
	return babeService
}

func TestMain(m *testing.M) {
	wasmFilePaths, err := runtime.GenerateRuntimeWasmFile()
	if err != nil {
		log.Errorf("failed to generate runtime wasm file: %s", err)
		os.Exit(1)
	}

	logger = log.NewFromGlobal(log.SetLevel(defaultTestLogLvl))

	// Start all tests
	code := m.Run()

	runtime.RemoveFiles(wasmFilePaths)
	os.Exit(code)
}

func newTestServiceSetupParameters(t *testing.T) (*Service, *state.EpochState, *types.BabeConfiguration) {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	testDatadirPath := t.TempDir()

	config := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}
	dbSrv := state.NewService(config)
	dbSrv.UseMemDB()

	gen, genTrie, genHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := dbSrv.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = dbSrv.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = dbSrv.Stop()
	})

	rtCfg := &wasmer.Config{}
	rtCfg.Storage, err = rtstorage.NewTrieState(genTrie)
	require.NoError(t, err)
	rt, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	genCfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	s := &Service{
		epochState: dbSrv.Epoch,
	}

	return s, dbSrv.Epoch, genCfg
}

func TestService_SlotDuration(t *testing.T) {
	duration, err := time.ParseDuration("1000ms")
	require.NoError(t, err)

	bs := &Service{
		constants: &constants{
			slotDuration: duration,
		},
	}

	dur := bs.constants.slotDuration
	require.Equal(t, dur.Milliseconds(), int64(1000))
}

func TestService_ProducesBlocks(t *testing.T) {
	babeService := createTestService(t, nil)
	babeService.lead = true

	err := babeService.Start()
	require.NoError(t, err)
	defer func() {
		_ = babeService.Stop()
	}()

	time.Sleep(babeService.constants.slotDuration * 2)
	babeService.blockImportHandler.(*mocks.BlockImportHandler).
		AssertCalled(t, "HandleBlockProduced",
			mock.AnythingOfType("*types.Block"),
			mock.AnythingOfType("*storage.TrieState"))
}

func TestService_GetAuthorityIndex(t *testing.T) {
	kpA, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	kpB, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	pubA := kpA.Public().(*sr25519.PublicKey)
	pubB := kpB.Public().(*sr25519.PublicKey)

	authData := []types.Authority{
		{Key: pubA, Weight: 1},
		{Key: pubB, Weight: 1},
	}

	bs := &Service{
		keypair:   kpA,
		authority: true,
	}

	idx, err := bs.getAuthorityIndex(authData)
	require.NoError(t, err)
	require.Equal(t, uint32(0), idx)

	bs = &Service{
		keypair:   kpB,
		authority: true,
	}

	idx, err = bs.getAuthorityIndex(authData)
	require.NoError(t, err)
	require.Equal(t, uint32(1), idx)
}

func TestStartAndStop(t *testing.T) {
	bs := createTestService(t, &ServiceConfig{
		LogLvl: log.Critical,
	})
	err := bs.Start()
	require.NoError(t, err)
	err = bs.Stop()
	require.NoError(t, err)
}

func TestService_PauseAndResume(t *testing.T) {
	bs := createTestService(t, &ServiceConfig{
		LogLvl: log.Critical,
	})
	err := bs.Start()
	require.NoError(t, err)
	time.Sleep(time.Second)

	go func() {
		_ = bs.Pause()
	}()

	go func() {
		_ = bs.Pause()
	}()

	go func() {
		err := bs.Resume()
		require.NoError(t, err)
	}()

	go func() {
		err := bs.Resume()
		require.NoError(t, err)
	}()

	err = bs.Stop()
	require.NoError(t, err)
}
