// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/babe/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

const (
	defaultTestLogLvl = log.Info
	testEpochIndex    = uint64(0)
)

var (
	emptyHash    = trie.EmptyHash
	maxThreshold = scale.MaxUint128

	emptyHeader = &types.Header{
		Number: 0,
		Digest: types.NewDigest(),
	}
)

// newTestCoreService creates a new test core service
func newTestCoreService(t *testing.T, cfg *core.Config, genesis genesis.Genesis,
	genesisTrie trie.Trie, genesisHeader types.Header) *core.Service {
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

		err := stateSrvc.Initialise(&genesis, &genesisHeader, &genesisTrie)
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

func createTestService(t *testing.T, cfg ServiceConfig, genesis genesis.Genesis,
	genesisTrie trie.Trie, genesisHeader types.Header, babeConfig *types.BabeConfiguration) *Service {
	wasmer.DefaultTestLogLvl = log.Error

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
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	cfg.Telemetry = telemetryMock

	testDatadirPath := t.TempDir()

	config := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}
	dbSrv := state.NewService(config)
	dbSrv.UseMemDB()

	dbSrv.Transaction = state.NewTransactionState(telemetryMock)

	err := dbSrv.Initialise(&genesis, &genesisHeader, &genesisTrie)
	require.NoError(t, err)

	err = dbSrv.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = dbSrv.Stop()
	})

	// Allow for epoch state to be made from custom babe config
	if babeConfig != nil {
		dbSrv.Epoch, err = state.NewEpochStateFromGenesis(dbSrv.DB(), dbSrv.Block, babeConfig)
		require.NoError(t, err)
	}
	cfg.BlockState = dbSrv.Block
	cfg.StorageState = dbSrv.Storage
	cfg.EpochState = dbSrv.Epoch
	cfg.TransactionState = dbSrv.Transaction

	var rtCfg wasmer.Config
	rtCfg.Storage = rtstorage.NewTrieState(&genesisTrie)

	storageState := cfg.StorageState.(*state.StorageState)
	rtCfg.CodeHash, err = storageState.LoadCodeHash(nil)
	require.NoError(t, err)

	nodeStorage := runtime.NodeStorage{}
	nodeStorage.BaseDB = dbSrv.Base

	rtCfg.NodeStorage = nodeStorage
	rtCfg.Transaction = dbSrv.Transaction
	runtime, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)
	cfg.BlockState.(*state.BlockState).StoreRuntime(cfg.BlockState.BestBlockHash(), runtime)

	cfg.Authority = true
	cfg.IsDev = true
	cfg.LogLvl = defaultTestLogLvl
	babeService, err := NewService(&cfg)
	require.NoError(t, err)

	if cfg.BlockImportHandler == nil {
		mockNetwork := mocks.NewMockNetwork(ctrl)
		mockNetwork.EXPECT().GossipMessage(gomock.Any()).AnyTimes()

		coreConfig := core.Config{
			BlockState:           dbSrv.Block,
			StorageState:         storageState,
			TransactionState:     dbSrv.Transaction,
			Runtime:              runtime,
			Keystore:             rtCfg.Keystore,
			Network:              mockNetwork,
			CodeSubstitutedState: dbSrv.Base,
			CodeSubstitutes:      make(map[common.Hash]string),
		}

		babeService.blockImportHandler = newTestCoreService(t, &coreConfig, genesis,
			genesisTrie, genesisHeader)
	}

	return babeService
}

func newTestServiceSetupParameters(t *testing.T, genesis genesis.Genesis,
	genesisTrie trie.Trie, genesisHeader types.Header) (*Service, *state.EpochState, *types.BabeConfiguration) {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	testDatadirPath := t.TempDir()

	config := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}
	dbSrv := state.NewService(config)
	dbSrv.UseMemDB()

	err := dbSrv.Initialise(&genesis, &genesisHeader, &genesisTrie)
	require.NoError(t, err)

	err = dbSrv.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = dbSrv.Stop()
	})

	rtCfg := wasmer.Config{
		Storage: rtstorage.NewTrieState(&genesisTrie),
	}

	rt, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	genCfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	s := &Service{
		epochState: dbSrv.Epoch,
	}

	return s, dbSrv.Epoch, genCfg
}

func createSecondaryVRFPreDigest(t *testing.T,
	keypair *sr25519.Keypair, index uint32,
	slot, epoch uint64, randomness Randomness,
) *types.BabeSecondaryVRFPreDigest {
	transcript := makeTranscript(randomness, slot, epoch)
	out, proof, err := keypair.VrfSign(transcript)
	require.NoError(t, err)

	return types.NewBabeSecondaryVRFPreDigest(index, slot, out, proof)
}

func buildLocalTransaction(t *testing.T, rt runtime.Instance, ext types.Extrinsic,
	bestBlockHash common.Hash) types.Extrinsic {
	runtimeVersion := rt.Version()
	txQueueVersion, err := runtimeVersion.TaggedTransactionQueueVersion()
	require.NoError(t, err)
	var extrinsicParts [][]byte
	switch txQueueVersion {
	case 3:
		extrinsicParts = [][]byte{{byte(types.TxnLocal)}, ext, bestBlockHash.ToBytes()}
	case 2:
		extrinsicParts = [][]byte{{byte(types.TxnLocal)}, ext}
	}
	return types.Extrinsic(bytes.Join(extrinsicParts, nil))
}

func getSlot(t *testing.T, rt runtime.Instance, timestamp time.Time) Slot {
	t.Helper()
	babeConfig, err := rt.BabeConfiguration()
	require.NoError(t, err)

	currentSlot := uint64(timestamp.UnixMilli()) / babeConfig.SlotDuration
	return Slot{
		start:    timestamp,
		duration: time.Duration(babeConfig.SlotDuration) * time.Millisecond,
		number:   currentSlot,
	}
}

func createTestBlockWithSlot(t *testing.T, babeService *Service, parent *types.Header,
	exts [][]byte, epoch uint64, epochData *epochData, slot Slot) *types.Block {
	for _, ext := range exts {
		validTransaction := transaction.NewValidTransaction(ext, &transaction.Validity{})
		_, err := babeService.transactionState.Push(validTransaction)
		require.NoError(t, err)
	}

	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	preRuntimeDigest, err := claimSlot(epoch, slot.number, epochData, babeService.keypair)
	require.NoError(t, err)

	block, err := babeService.buildBlock(parent, slot, rt, epochData.authorityIndex, preRuntimeDigest)
	require.NoError(t, err)

	babeService.blockState.(*state.BlockState).StoreRuntime(block.Header.Hash(), rt)
	return block
}

// newWestendLocalGenesisWithTrieAndHeader returns the westend genesis, genesis trie and genesis header
func newWestendLocalGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendLocalRawGenesisPath(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = wasmer.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader = *types.NewHeader(common.NewHash([]byte{0}),
		genesisTrie.MustHash(), trie.EmptyHash, 0, types.NewDigest())

	return gen, genesisTrie, genesisHeader
}

// newWestendDevGenesisWithTrieAndHeader returns the westend genesis, genesis trie and genesis header
func newWestendDevGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendDevRawGenesisPath(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = wasmer.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader = *types.NewHeader(common.NewHash([]byte{0}),
		genesisTrie.MustHash(), trie.EmptyHash, 0, types.NewDigest())

	return gen, genesisTrie, genesisHeader
}
