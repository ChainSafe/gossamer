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

package babe

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

var (
	emptyHash       = trie.EmptyHash
	testTimeout     = time.Second * 5
	testEpochIndex  = uint64(0)
	testEpochLength = uint64(10)

	maxThreshold = common.MaxUint128
	minThreshold = &common.Uint128{}

	genesisHeader = &types.Header{
		Number:    big.NewInt(0),
		StateRoot: emptyHash,
	}

	emptyHeader = &types.Header{
		Number: big.NewInt(0),
	}

	genesisBABEConfig = &types.BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: []*types.AuthorityRaw{},
		Randomness:         [32]byte{},
		SecondarySlots:     false,
	}
)

func createTestService(t *testing.T, cfg *ServiceConfig) *Service {
	wasmer.DefaultTestLogLvl = 1

	var err error
	tt := trie.NewEmptyTrie()
	rt := wasmer.NewTestInstanceWithTrie(t, runtime.NODE_RUNTIME, tt, log.LvlCrit)

	if cfg == nil {
		cfg = &ServiceConfig{
			Runtime:   rt,
			Authority: true,
		}
	}

	if cfg.Runtime == nil {
		cfg.Runtime = rt
	}

	if cfg.Keypair == nil {
		cfg.Keypair, err = sr25519.GenerateKeypair()
		require.NoError(t, err)
	}

	if cfg.AuthData == nil {
		auth := &types.Authority{
			Key:    cfg.Keypair.Public().(*sr25519.PublicKey),
			Weight: 1,
		}
		cfg.AuthData = []*types.Authority{auth}
	}

	if cfg.TransactionState == nil {
		cfg.TransactionState = state.NewTransactionState()
	}

	if cfg.BlockState == nil || cfg.StorageState == nil || cfg.EpochState == nil {
		testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*") //nolint
		require.NoError(t, err)
		dbSrv := state.NewService(testDatadirPath, log.LvlInfo)
		dbSrv.UseMemDB()

		genesisData := new(genesis.Data)

		if cfg.EpochLength > 0 {
			genesisBABEConfig.EpochLength = cfg.EpochLength
		}

		err = dbSrv.Initialize(genesisData, genesisHeader, tt, genesisBABEConfig)
		require.NoError(t, err)

		err = dbSrv.Start()
		require.NoError(t, err)

		cfg.BlockState = dbSrv.Block
		cfg.StorageState = dbSrv.Storage
		cfg.EpochState = dbSrv.Epoch
	}

	if cfg.StartSlot == 0 {
		cfg.StartSlot = 1
	}

	babeService, err := NewService(cfg)
	require.NoError(t, err)
	return babeService
}

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

func TestRunEpochLengthConfig(t *testing.T) {
	cfg := &ServiceConfig{
		EpochLength: 5,
	}

	babeService := createTestService(t, cfg)

	if babeService.epochLength != 5 {
		t.Fatal("epoch length not set")
	}
}

func TestSlotDuration(t *testing.T) {
	bs := &Service{
		slotDuration: 1000,
	}

	dur := bs.getSlotDuration()
	require.Equal(t, dur.Milliseconds(), int64(1000))
}

func TestBabeAnnounceMessage(t *testing.T) {
	// this test uses a real database because it sets the runtime Storage context, which
	// must be backed by a non-memory database
	datadir, _ := ioutil.TempDir("/tmp", "test-datadir-*")
	dbSrv := state.NewService(datadir, log.LvlInfo)
	genesisData := new(genesis.Data)
	err := dbSrv.Initialize(genesisData, genesisHeader, trie.NewEmptyTrie(), genesisBABEConfig)
	require.NoError(t, err)
	err = dbSrv.Start()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		BlockState:       dbSrv.Block,
		StorageState:     dbSrv.Storage,
		EpochState:       dbSrv.Epoch,
		TransactionState: dbSrv.Transaction,
		LogLvl:           log.LvlTrace,
		Authority:        true,
	}

	babeService := createTestService(t, cfg)
	t.Cleanup(func() {
		_ = dbSrv.Stop()
		os.RemoveAll(datadir)
		_ = babeService.Stop()
	})

	babeService.epochData.authorityIndex = 0
	babeService.epochData.authorities = []*types.Authority{
		{Key: nil, Weight: 1},
		{Key: nil, Weight: 1},
		{Key: nil, Weight: 1},
	}

	babeService.epochData.threshold = maxThreshold
	blockNumber := big.NewInt(int64(1))

	err = babeService.Start()
	require.NoError(t, err)

	newBlocks := babeService.GetBlockChannel()
	select {
	case block := <-newBlocks:
		require.Equal(t, blockNumber, block.Header.Number)
	case <-time.After(testTimeout):
		t.Fatal("did not receive block")
	}
}

func TestGetAuthorityIndex(t *testing.T) {
	kpA, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	kpB, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	pubA := kpA.Public().(*sr25519.PublicKey)
	pubB := kpB.Public().(*sr25519.PublicKey)

	authData := []*types.Authority{
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
		LogLvl: log.LvlCrit,
	})
	err := bs.Start()
	require.NoError(t, err)
	err = bs.Stop()
	require.NoError(t, err)
}
