// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func newTestVerificationManager(t *testing.T, genCfg *types.BabeConfiguration) *VerificationManager {
	testDatadirPath := t.TempDir()

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Info,
		Telemetry: telemetryMock,
	}

	dbSrv := state.NewService(config)
	dbSrv.UseMemDB()

	gen, genTrie, genHeader := genesis.NewDevGenesisWithTrieAndHeader(t)
	err := dbSrv.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = dbSrv.Start()
	require.NoError(t, err)

	if genCfg == nil {
		genCfg = genesisBABEConfig
	}

	dbSrv.Epoch, err = state.NewEpochStateFromGenesis(dbSrv.DB(), dbSrv.Block, genCfg)
	require.NoError(t, err)

	logger.Patch(log.SetLevel(defaultTestLogLvl))

	vm, err := NewVerificationManager(dbSrv.Block, dbSrv.Epoch)
	require.NoError(t, err)
	return vm
}

func TestVerificationManager_OnDisabled_InvalidIndex(t *testing.T) {
	vm := newTestVerificationManager(t, nil)
	babeService := createTestService(t, nil)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	err = vm.SetOnDisabled(1, &block.Header)
	require.Equal(t, err, ErrInvalidBlockProducerIndex)
}

func TestVerificationManager_OnDisabled_NewDigest(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	vm := newTestVerificationManager(t, nil)
	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	}

	parent, _ := babeService.blockState.BestBlockHeader()

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	block = createTestBlock(t, babeService, parent, [][]byte{}, 2, testEpochIndex, epochData)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_OnDisabled_DuplicateDigest(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	vm := newTestVerificationManager(t, nil)
	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	}

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	err = vm.blockState.AddBlock(block)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	block2 := createTestBlock(t, babeService, &block.Header, [][]byte{}, 2, testEpochIndex, epochData)
	err = vm.blockState.AddBlock(block2)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block2.Header)
	require.Equal(t, ErrAuthorityAlreadyDisabled, err)
}

func TestVerificationManager_VerifyBlock_Ok(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	err = vm.VerifyBlock(&block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_Secondary(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1
	cfg.SecondarySlots = 0

	vm := newTestVerificationManager(t, cfg)

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	dig := createSecondaryVRFPreDigest(t, kp, 0, uint64(0), uint64(0), Randomness{})

	bd := types.NewBabeDigest()
	err = bd.Set(dig)
	require.NoError(t, err)

	bdEnc, err := scale.Marshal(bd)
	require.NoError(t, err)

	// create pre-digest
	preDigest := &types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              bdEnc,
	}

	// create new block header
	const number uint = 1
	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	require.NoError(t, err)

	// create seal and add to digest
	seal := &types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{0},
	}
	require.NoError(t, err)

	err = digest.Add(*seal)
	require.NoError(t, err)

	header, err := types.NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, number, digest)
	require.NoError(t, err)

	block := types.Block{
		Header: *header,
		Body:   nil,
	}
	err = vm.VerifyBlock(&block.Header)
	require.EqualError(t, err, "failed to verify pre-runtime digest: could not verify slot claim VRF proof")
}

func TestVerificationManager_VerifyBlock_MultipleEpochs(t *testing.T) {
	t.Skip() // TODO: no idea why it's complaining it can't find the epoch data. fix later
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 1

	vm := newTestVerificationManager(t, cfg)

	futureEpoch := uint64(5)

	err = vm.epochState.(*state.EpochState).SetEpochData(futureEpoch, &types.EpochData{
		Authorities: epochData.authorities,
		Randomness:  epochData.randomness,
	})
	require.NoError(t, err)

	futureEpochData, err := babeService.initiateEpoch(futureEpoch)
	require.NoError(t, err)

	// create block in future epoch
	block1 := createTestBlock(t, babeService, genesisHeader, [][]byte{},
		cfg.EpochLength*futureEpoch+1, futureEpoch, futureEpochData)
	block2 := createTestBlock(t, babeService, &block1.Header, [][]byte{},
		cfg.EpochLength*futureEpoch+2, futureEpoch, futureEpochData)

	err = vm.VerifyBlock(&block2.Header)
	require.NoError(t, err)

	// create block in epoch 1
	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, cfg.EpochLength-10, testEpochIndex, epochData)

	err = vm.VerifyBlock(&block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_InvalidBlockOverThreshold(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	cfg.GenesisAuthorities = types.AuthoritiesToRaw(epochData.authorities)
	cfg.C1 = 1
	cfg.C2 = 100

	vm := newTestVerificationManager(t, cfg)

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)

	err = vm.VerifyBlock(&block.Header)
	require.Equal(t, ErrVRFOutputOverThreshold, errors.Unwrap(err))
}

func TestVerificationManager_VerifyBlock_InvalidBlockAuthority(t *testing.T) {
	babeService := createTestService(t, nil)
	rt, err := babeService.blockState.GetRuntime(nil)
	require.NoError(t, err)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	cfg.C1 = 1
	cfg.C2 = 1
	cfg.GenesisAuthorities = []types.AuthorityRaw{}

	vm := newTestVerificationManager(t, cfg)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)

	err = vm.VerifyBlock(&block.Header)
	require.Equal(t, ErrInvalidBlockProducerIndex, errors.Unwrap(err))
}

func TestVerifyPimarySlotWinner(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	epochData, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	// create proof that we can authorize this block
	epochData.threshold = maxThreshold
	epochData.authorityIndex = 0

	const slotNumber uint64 = 1

	preRuntimeDigest, err := claimSlot(testEpochIndex, slotNumber, epochData, babeService.keypair)
	require.NoError(t, err)

	babePreDigest, err := types.DecodeBabePreDigest(preRuntimeDigest.Data)
	require.NoError(t, err)

	d, ok := babePreDigest.(types.BabePrimaryPreDigest)
	require.True(t, ok)

	Authorities := make([]types.Authority, 1)
	Authorities[0] = types.Authority{
		Key: kp.Public().(*sr25519.PublicKey),
	}
	epochData.authorities = Authorities

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	})
	require.NoError(t, err)

	ok, err = verifier.verifyPrimarySlotWinner(d.AuthorityIndex, slotNumber, d.VRFOutput, d.VRFProof)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyAuthorshipRight(t *testing.T) {
	babeService := createTestService(t, nil)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)
	epochData.threshold = maxThreshold

	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	})
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(&block.Header)
	require.NoError(t, err)
}

func TestVerifyAuthorshipRight_Equivocation(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)
	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	epochData.threshold = maxThreshold
	epochData.authorities = []types.Authority{
		{
			Key: kp.Public().(*sr25519.PublicKey),
		},
	}

	// create and add first block
	block := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	block.Header.Hash()

	err = babeService.blockState.AddBlock(block)
	require.NoError(t, err)

	verifier, err := newVerifier(babeService.blockState, testEpochIndex, &verifierInfo{
		authorities: epochData.authorities,
		threshold:   epochData.threshold,
		randomness:  epochData.randomness,
	})
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(&block.Header)
	require.NoError(t, err)

	// create new block
	block2 := createTestBlock(t, babeService, genesisHeader, [][]byte{}, 1, testEpochIndex, epochData)
	block2.Header.Hash()

	err = babeService.blockState.AddBlock(block2)
	require.NoError(t, err)

	err = verifier.verifyAuthorshipRight(&block2.Header)
	require.Equal(t, ErrProducerEquivocated, err)
}
