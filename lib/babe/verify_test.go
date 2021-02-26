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
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	log "github.com/ChainSafe/log15"
)

func newTestVerificationManager(t *testing.T, genCfg *types.BabeConfiguration) *VerificationManager {
	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	dbSrv := state.NewService(testDatadirPath, log.LvlInfo)
	dbSrv.UseMemDB()

	if genCfg == nil {
		genCfg = genesisBABEConfig
	}

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t)
	err = dbSrv.Initialize(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = dbSrv.Start()
	require.NoError(t, err)
	dbSrv.Epoch, err = state.NewEpochStateFromGenesis(dbSrv.DB(), genCfg)
	require.NoError(t, err)

	logger = log.New("pkg", "babe")
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	h = log.CallerFileHandler(h)
	logger.SetHandler(log.LvlFilterHandler(defaultTestLogLvl, h))

	vm, err := NewVerificationManager(dbSrv.Block, dbSrv.Epoch)
	require.NoError(t, err)
	return vm
}

func TestVerificationManager_OnDisabled_InvalidIndex(t *testing.T) {
	vm := newTestVerificationManager(t, nil)

	babeService := createTestService(t, &ServiceConfig{
		ThresholdNumerator:   1,
		ThresholdDenominator: 1,
	})
	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	err := vm.SetOnDisabled(1, block.Header)
	require.Equal(t, err, ErrInvalidBlockProducerIndex)
}

func TestVerificationManager_OnDisabled_NewDigest(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair:              kp,
		ThresholdNumerator:   1,
		ThresholdDenominator: 1,
	}

	babeService := createTestService(t, cfg)

	vm := newTestVerificationManager(t, nil)
	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	}

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	block, _ = createTestBlock(t, babeService, genesisHeader, [][]byte{}, 2, testEpochIndex)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_OnDisabled_DuplicateDigest(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair:              kp,
		ThresholdNumerator:   1,
		ThresholdDenominator: 1,
	}

	babeService := createTestService(t, cfg)

	vm := newTestVerificationManager(t, nil)
	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	}

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	block2, _ := createTestBlock(t, babeService, block.Header, [][]byte{}, 2, testEpochIndex)
	err = vm.blockState.AddBlock(block2)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, block2.Header)
	require.Equal(t, ErrAuthorityAlreadyDisabled, err)
}

func TestVerificationManager_VerifyBlock_IsDisabled(t *testing.T) {
	t.Skip() // TODO: fix OnDisabled digests and re-enable this

	babeService := createTestService(t, &ServiceConfig{
		ThresholdNumerator:   1,
		ThresholdDenominator: 1,
	})
	cfg, err := babeService.rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(babeService.epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)
	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)

	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, block.Header)
	require.NoError(t, err)

	// a block that we created, that disables ourselves, should still be accepted
	err = vm.VerifyBlock(block.Header)
	require.NoError(t, err)

	block, _ = createTestBlock(t, babeService, block.Header, [][]byte{}, 2, testEpochIndex)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	// any blocks following the one where we are disabled should reject
	err = vm.VerifyBlock(block.Header)
	require.Equal(t, ErrAuthorityDisabled, err)

	// let's try a block on a different chain, it shouldn't reject
	parentHeader := genesisHeader
	for slot := 77; slot < 80; slot++ {
		block, _ = createTestBlock(t, babeService, parentHeader, [][]byte{}, uint64(slot), testEpochIndex)
		err = vm.blockState.AddBlock(block)
		require.NoError(t, err)
		parentHeader = block.Header
	}

	err = vm.VerifyBlock(block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_Ok(t *testing.T) {
	babeService := createTestService(t, &ServiceConfig{
		ThresholdNumerator:   1,
		ThresholdDenominator: 1,
	})
	cfg, err := babeService.rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(babeService.epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)

	err = vm.VerifyBlock(block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_MultipleEpochs(t *testing.T) {
	babeService := createTestService(t, &ServiceConfig{
		ThresholdNumerator:   1,
		ThresholdDenominator: 1,
	})
	cfg, err := babeService.rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(babeService.epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)

	futureEpoch := uint64(5)

	err = vm.epochState.(*state.EpochState).SetEpochData(futureEpoch, &types.EpochData{
		Authorities: babeService.epochData.authorities,
		Randomness:  babeService.epochData.randomness,
	})
	require.NoError(t, err)

	// create block in future epoch
	block1, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, cfg.EpochLength*futureEpoch+1, futureEpoch)
	block2, _ := createTestBlock(t, babeService, block1.Header, [][]byte{}, cfg.EpochLength*futureEpoch+2, futureEpoch)

	err = vm.VerifyBlock(block2.Header)
	require.NoError(t, err)

	// create block in epoch 1
	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, cfg.EpochLength-10, testEpochIndex)

	err = vm.VerifyBlock(block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_InvalidBlockOverThreshold(t *testing.T) {
	t.Skip() // TODO
	babeService := createTestService(t, &ServiceConfig{
		ThresholdNumerator:   1,
		ThresholdDenominator: 1,
	})
	cfg, err := babeService.rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(babeService.epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 100

	vm := newTestVerificationManager(t, cfg)

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)

	err = vm.VerifyBlock(block.Header)
	require.Equal(t, ErrVRFOutputOverThreshold, errors.Unwrap(err))
}

func TestVerificationManager_VerifyBlock_InvalidBlockAuthority(t *testing.T) {
	babeService := createTestService(t, &ServiceConfig{
		ThresholdNumerator:   1,
		ThresholdDenominator: 1,
	})
	cfg, err := babeService.rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.C1 = 1
	cfg.C2 = 1
	cfg.GenesisAuthorities = []*types.AuthorityRaw{}

	vm := newTestVerificationManager(t, cfg)

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)

	err = vm.VerifyBlock(block.Header)
	require.Equal(t, ErrInvalidBlockProducerIndex, errors.Unwrap(err))
}

func TestVerifyPimarySlotWinner(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)

	// create proof that we can authorize this block
	babeService.epochData.threshold = maxThreshold
	babeService.epochData.authorityIndex = 0
	var slotNumber uint64 = 1

	addAuthorshipProof(t, babeService, slotNumber, testEpochIndex)
	duration, err := time.ParseDuration("1s")
	require.NoError(t, err)

	slot := Slot{
		start:    time.Now(),
		duration: duration,
		number:   slotNumber,
	}

	// create babe header
	babeHeader, err := babeService.buildBlockBABEPrimaryPreDigest(slot)
	require.NoError(t, err)

	Authorities := make([]*types.Authority, 1)
	Authorities[0] = &types.Authority{
		Key: kp.Public().(*sr25519.PublicKey),
	}
	babeService.epochData.authorities = Authorities

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	})
	require.NoError(t, err)

	ok, err := verifier.verifyPrimarySlotWinner(babeHeader.AuthorityIndex(), slot.number, babeHeader.VrfOutput(), babeHeader.VrfProof())
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyAuthorshipRight(t *testing.T) {
	babeService := createTestService(t, nil)
	babeService.epochData.threshold = maxThreshold

	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	})
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(block.Header)
	require.NoError(t, err)
}

func TestVerifyAuthorshipRight_Equivocation(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	babeService.epochData.threshold = maxThreshold

	babeService.epochData.authorities = make([]*types.Authority, 1)
	babeService.epochData.authorities[0] = &types.Authority{
		Key: kp.Public().(*sr25519.PublicKey),
	}

	// create and add first block
	block, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	block.Header.Hash()

	err = babeService.blockState.AddBlock(block)
	require.NoError(t, err)

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: babeService.epochData.authorities,
		threshold:   babeService.epochData.threshold,
		randomness:  babeService.epochData.randomness,
	})
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(block.Header)
	require.NoError(t, err)

	// create new block
	block2, _ := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex)
	block2.Header.Hash()

	err = babeService.blockState.AddBlock(block2)
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(block2.Header)
	require.Equal(t, ErrProducerEquivocated, err)
}
