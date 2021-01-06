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
	"io/ioutil"
	"math"
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

var emptyHash = trie.EmptyHash
var maxThreshold = big.NewInt(0).SetBytes([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
var testTimeout = time.Second * 5

var genesisHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: emptyHash,
}

var emptyHeader = &types.Header{
	Number: big.NewInt(0),
}

var testEpochLength = uint64(10)

var genesisBABEConfig = &types.BabeConfiguration{
	SlotDuration:       1000,
	EpochLength:        200,
	C1:                 1,
	C2:                 4,
	GenesisAuthorities: []*types.AuthorityRaw{},
	Randomness:         [32]byte{},
	SecondarySlots:     false,
}

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
		dbSrv := state.NewService("", log.LvlInfo)
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

func TestCalculateThreshold(t *testing.T) {
	// C = 1
	var C1 uint64 = 1
	var C2 uint64 = 1

	expected := new(big.Int).Lsh(big.NewInt(1), 256)

	threshold, err := CalculateThreshold(C1, C2, 3)
	require.NoError(t, err)

	if threshold.Cmp(expected) != 0 {
		t.Fatalf("Fail: got %d expected %d", threshold, expected)
	}

	// C = 1/2
	C2 = 2

	theta := float64(1) / float64(3)
	c := float64(C1) / float64(C2)
	pp := 1 - c
	pp_exp := math.Pow(pp, theta)
	p := 1 - pp_exp
	p_rat := new(big.Rat).SetFloat64(p)
	q := new(big.Int).Lsh(big.NewInt(1), 256)
	expected = q.Mul(q, p_rat.Num()).Div(q, p_rat.Denom())

	threshold, err = CalculateThreshold(C1, C2, 3)
	require.NoError(t, err)

	if threshold.Cmp(expected) != 0 {
		t.Fatalf("Fail: got %d expected %d", threshold, expected)
	}
}

func TestCalculateThreshold_Failing(t *testing.T) {
	var C1 uint64 = 5
	var C2 uint64 = 4

	_, err := CalculateThreshold(C1, C2, 3)
	if err == nil {
		t.Fatal("Fail: did not err for c>1")
	}
}

func TestRunLottery(t *testing.T) {
	babeService := createTestService(t, nil)

	babeService.epochData.threshold = maxThreshold

	outAndProof, err := babeService.runLottery(0)
	if err != nil {
		t.Fatal(err)
	}

	if outAndProof == nil {
		t.Fatal("proof was nil when over threshold")
	}
}

func TestRunLottery_False(t *testing.T) {
	babeService := createTestService(t, nil)
	babeService.epochData.threshold = big.NewInt(0)

	outAndProof, err := babeService.runLottery(0)
	if err != nil {
		t.Fatal(err)
	}

	if outAndProof != nil {
		t.Fatal("proof was not nil when under threshold")
	}
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
		logger:    log.New("BABE"),
		authority: true,
	}

	idx, err := bs.getAuthorityIndex(authData)
	require.NoError(t, err)
	require.Equal(t, uint64(0), idx)

	bs = &Service{
		keypair:   kpB,
		logger:    log.New("BABE"),
		authority: true,
	}

	idx, err = bs.getAuthorityIndex(authData)
	require.NoError(t, err)
	require.Equal(t, uint64(1), idx)
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
