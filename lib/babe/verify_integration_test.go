// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/tests/utils/config"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestVerificationManager_OnDisabled_InvalidIndex(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)

	slotState := state.NewSlotState(db)
	vm := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	epochDescriptor, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochDescriptor, slot)

	err = vm.SetOnDisabled(1, &block.Header)
	require.Equal(t, ErrInvalidBlockProducerIndex, err)
}

func TestVerificationManager_OnDisabled_NewDigest(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	vm := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	epochDescriptor, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: epochDescriptor.data.authorities,
		threshold:   epochDescriptor.data.threshold,
		randomness:  epochDescriptor.data.randomness,
	}

	slot := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochDescriptor, slot)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	slot2 := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block = createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochDescriptor, slot2)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_OnDisabled_DuplicateDigest(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	epochDescriptor, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)
	vm := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	vm.epochInfo[testEpochIndex] = &verifierInfo{
		authorities: epochDescriptor.data.authorities,
		threshold:   epochDescriptor.data.threshold,
		randomness:  epochDescriptor.data.randomness,
	}

	slot := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochDescriptor, slot)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block.Header)
	require.NoError(t, err)

	// create an OnDisabled change on a different branch
	slot2 := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block2 := createTestBlockWithSlot(t, babeService, &block.Header, [][]byte{}, epochDescriptor, slot2)
	require.NoError(t, err)

	err = vm.SetOnDisabled(0, &block2.Header)
	require.Equal(t, ErrAuthorityAlreadyDisabled, err)
}

func TestVerificationManager_VerifyBlock_Secondary(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)

	babeCfgWithSecondarySlots := config.BABEConfigurationTestDefault
	// these parameters will decrease the probability
	// of a primary author claiming which will makes us
	// propose a secondary block
	babeCfgWithSecondarySlots.C1 = 1
	babeCfgWithSecondarySlots.C2 = 9000
	babeCfgWithSecondarySlots.SecondarySlots = 1
	babeCfgWithSecondarySlots.GenesisAuthorities = []types.AuthorityRaw{
		{
			Key:    keyring.Alice().Public().(*sr25519.PublicKey).AsBytes(),
			Weight: 1,
		},
	}

	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie,
		genesisHeader, babeCfgWithSecondarySlots)
	babeService.authority = true

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	vm := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	const epoch = 0
	epochDescriptor, err := babeService.initiateEpoch(epoch)
	require.NoError(t, err)

	const authIndex = 0
	secondaryDigest := createSecondaryVRFPreDigest(t, keyring.Alice().(*sr25519.Keypair),
		authIndex, epochDescriptor.startSlot, epochDescriptor.epoch, epochDescriptor.data.randomness)
	babeDigest := types.NewBabeDigest()

	// NOTE: I think this was get encoded incorrectly before the VDT interface change.
	// *types.BabeSecondaryVRFPreDigest was being passed in and encoded later
	err = babeDigest.SetValue(*secondaryDigest)
	require.NoError(t, err)

	encodedBabeDigest, err := scale.Marshal(babeDigest)
	require.NoError(t, err)

	// create pre-digest
	preDigest := &types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              encodedBabeDigest,
	}

	// create new block header
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

	block := types.Block{
		Header: types.Header{
			Number:     1,
			ParentHash: genesisHeader.Hash(),
			Digest:     digest,
		},
		Body: nil,
	}
	err = vm.VerifyBlock(&block.Header)
	require.EqualError(t, err, "invalid signature length")
}

func TestVerificationManager_VerifyBlock_CurrentEpoch(t *testing.T) {
	t.Parallel()
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	vm := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	epochDescriptor, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	slot := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochDescriptor, slot)

	err = vm.VerifyBlock(&block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_FutureEpoch(t *testing.T) {
	t.Skip("TODO: move this test under TestVerificationManager_VerifyBlock_MultipleEpochs")

	t.Parallel()
	auth := types.Authority{
		Key:    keyring.Alice().(*sr25519.Keypair).Public(),
		Weight: 1,
	}
	defaultBabeConfiguration := config.BABEConfigurationTestDefault
	defaultBabeConfiguration.GenesisAuthorities = []types.AuthorityRaw{*auth.ToRaw()}
	defaultBabeConfiguration.SecondarySlots = 1

	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie,
		genesisHeader, defaultBabeConfiguration)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	verificationManager := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	const futureEpoch = uint64(2)
	err = babeService.epochState.(*state.EpochState).SetEpochDataRaw(futureEpoch, &types.EpochDataRaw{
		Authorities: []types.AuthorityRaw{{
			Key: [32]byte(keyring.Alice().(*sr25519.Keypair).Public().Encode()),
		}},
	})
	require.NoError(t, err)

	futureEpochDescriptor, err := babeService.initiateEpoch(futureEpoch)
	require.NoError(t, err)

	slot := Slot{
		start:    getSlotStartTime(futureEpochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   futureEpochDescriptor.startSlot,
	}
	slot.number = babeService.EpochLength()*futureEpoch + 1
	block := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, futureEpochDescriptor, slot)

	err = verificationManager.VerifyBlock(&block.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_MultipleEpochs(t *testing.T) {
	t.Parallel()
	auth := types.Authority{
		Key:    keyring.Alice().(*sr25519.Keypair).Public(),
		Weight: 1,
	}

	babeConfig := &types.BabeConfiguration{
		SlotDuration:       6000,
		EpochLength:        600,
		C1:                 1,
		C2:                 1,
		GenesisAuthorities: []types.AuthorityRaw{*auth.ToRaw()},
		Randomness:         [32]byte{},
		SecondarySlots:     1,
	}

	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie,
		genesisHeader, babeConfig)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)
	verificationManager := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	const epoch = uint64(0)
	epochDescriptor, err := babeService.initiateEpoch(epoch)
	require.NoError(t, err)

	slot := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}

	blockNumber01 := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochDescriptor, slot)
	err = verificationManager.VerifyBlock(&blockNumber01.Header)
	require.NoError(t, err)

	futureEpoch := uint64(1)
	err = babeService.epochState.(*state.EpochState).SetEpochDataRaw(futureEpoch, &types.EpochDataRaw{
		Randomness: [32]byte{9},
		Authorities: []types.AuthorityRaw{
			{
				Key:    [32]byte(keyring.Bob().(*sr25519.Keypair).Public().Encode()),
				Weight: 1,
			},
			{
				Key:    [32]byte(keyring.Alice().(*sr25519.Keypair).Public().Encode()),
				Weight: 1,
			},
		},
	})
	require.NoError(t, err)

	futureEpochDescriptor, err := babeService.initiateEpoch(futureEpoch)
	require.NoError(t, err)

	futureSlot := Slot{
		start:    getSlotStartTime(futureEpochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   futureEpochDescriptor.startSlot,
	}
	blockNumber02 := createTestBlockWithSlot(t, babeService,
		&blockNumber01.Header, [][]byte{}, futureEpochDescriptor, futureSlot)

	err = verificationManager.VerifyBlock(&blockNumber02.Header)
	require.NoError(t, err)

	// skip the epoch 2 and initiate epoch 3, we should use epoch data that were
	// meant to be used by epoch 2
	skippedEpoch := uint64(2)
	err = babeService.epochState.(*state.EpochState).SetEpochDataRaw(skippedEpoch, &types.EpochDataRaw{
		Randomness: [32]byte{9},
		Authorities: []types.AuthorityRaw{
			{
				Key:    [32]byte(keyring.Bob().(*sr25519.Keypair).Public().Encode()),
				Weight: 1,
			},
			{
				Key:    [32]byte(keyring.Alice().(*sr25519.Keypair).Public().Encode()),
				Weight: 1,
			},
		},
	})
	require.NoError(t, err)

	futureEpoch = uint64(3)
	futureEpochDescriptor, err = babeService.initiateEpoch(futureEpoch)
	require.NoError(t, err)

	futureSlot = Slot{
		start:    getSlotStartTime(futureEpochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   futureEpochDescriptor.startSlot,
	}
	blockNumber03 := createTestBlockWithSlot(t, babeService,
		&blockNumber01.Header, [][]byte{}, futureEpochDescriptor, futureSlot)
	err = verificationManager.VerifyBlock(&blockNumber03.Header)
	require.NoError(t, err)
}

func TestVerificationManager_VerifyBlock_InvalidBlockOverThreshold(t *testing.T) {
	t.Parallel()
	auth := types.Authority{
		Key:    keyring.Alice().(*sr25519.Keypair).Public(),
		Weight: 1,
	}

	babeConfig := &types.BabeConfiguration{
		SlotDuration: 6000,
		EpochLength:  600,
		// have decreased the primary probability to be 1 in 9000
		// slots, then when claiming a slot we can increase the likely
		// to test the ErrVRFOutputOverThreshold error
		C1:                 1,
		C2:                 9000,
		GenesisAuthorities: []types.AuthorityRaw{*auth.ToRaw()},
		Randomness:         [32]byte{},
		SecondarySlots:     0,
	}

	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, babeConfig)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	vm := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	const epoch = 0
	epochDescriptor, err := babeService.initiateEpoch(epoch)
	require.NoError(t, err)

	epochDescriptor.data.threshold = maxThreshold

	slot := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochDescriptor, slot)
	block.Header.Hash()

	err = vm.VerifyBlock(&block.Header)
	require.Equal(t, ErrVRFOutputOverThreshold, errors.Unwrap(err))
}

func TestVerificationManager_VerifyBlock_InvalidBlockAuthority(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	babeConfig := &types.BabeConfiguration{
		SlotDuration:       6000,
		EpochLength:        600,
		C1:                 1,
		C2:                 1,
		GenesisAuthorities: nil,
		Randomness:         [32]byte{},
		SecondarySlots:     0,
	}

	genesisBob, genesisTrieBob, genesisHeaderBob := newWestendDevGenesisWithTrieAndHeader(t)
	babeServiceBob := createTestService(t, ServiceConfig{}, genesisBob, genesisTrieBob,
		genesisHeaderBob, babeConfig)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	verificationManager := NewVerificationManager(babeServiceBob.blockState, slotState, babeServiceBob.epochState)

	epochDescriptor, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block := createTestBlockWithSlot(t, babeServiceBob, &genesisHeader, [][]byte{}, epochDescriptor, slot)

	err = verificationManager.VerifyBlock(&block.Header)
	require.Equal(t, ErrInvalidBlockProducerIndex, errors.Unwrap(err))
}

func TestVerifyPrimarySlotWinner(t *testing.T) {
	auth := types.Authority{
		Key:    keyring.Alice().(*sr25519.Keypair).Public(),
		Weight: 1,
	}
	babeConfig := config.BABEConfigurationTestDefault
	babeConfig.SecondarySlots = 1
	babeConfig.GenesisAuthorities = []types.AuthorityRaw{*auth.ToRaw()}

	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, babeConfig)
	epochDescriptor, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	// create proof that we can authorize this block
	epochDescriptor.data.threshold = maxThreshold
	epochDescriptor.data.authorityIndex = 0

	const slotNumber uint64 = 1

	preRuntimeDigest, err := claimSlot(testEpochIndex, slotNumber, epochDescriptor.data, babeService.keypair)
	require.NoError(t, err)

	babePreDigest, err := types.DecodeBabePreDigest(preRuntimeDigest.Data)
	require.NoError(t, err)

	digest, ok := babePreDigest.(types.BabePrimaryPreDigest)
	require.True(t, ok)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	verifier := newVerifier(babeService.blockState, slotState, testEpochIndex, &verifierInfo{
		authorities: epochDescriptor.data.authorities,
		threshold:   epochDescriptor.data.threshold,
		randomness:  epochDescriptor.data.randomness,
	}, time.Second)

	ok, err = verifier.verifyPrimarySlotWinner(digest.AuthorityIndex, slotNumber, digest.VRFOutput, digest.VRFProof)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyAuthorshipRight(t *testing.T) {
	serviceConfig := ServiceConfig{
		Authority: true,
	}
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, serviceConfig, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	epochDescriptor, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)
	epochDescriptor.data.threshold = maxThreshold

	slot := Slot{
		start:    getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration),
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}
	block := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochDescriptor, slot)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	verifier := newVerifier(babeService.blockState, slotState, testEpochIndex, &verifierInfo{
		authorities: epochDescriptor.data.authorities,
		threshold:   epochDescriptor.data.threshold,
		randomness:  epochDescriptor.data.randomness,
	}, time.Second)

	err = verifier.verifyAuthorshipRight(&block.Header)
	require.NoError(t, err)
}

func TestVerifyAuthorshipRight_Equivocation(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	verificationManager := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	// bestBlockHash := babeService.blockState.BestBlockHash()
	// runtime, err := babeService.blockState.GetRuntime(bestBlockHash)
	// require.NoError(t, err)

	// slots are 6 seconds on westend and using time.Now() allows us to create a block at any point in the slot.
	// So we need to manually set time to produce consistent results. See here:
	// https://github.com/paritytech/substrate/blob/09de7b41599add51cf27eca8f1bc4c50ed8e9453/frame/timestamp/src/lib.rs#L229
	// https://github.com/paritytech/substrate/blob/09de7b41599add51cf27eca8f1bc4c50ed8e9453/frame/timestamp/src/lib.rs#L206

	const slotDuration = 6 * time.Second
	slotNumber := getCurrentSlot(slotDuration)
	startTime := getSlotStartTime(slotNumber, slotDuration)
	slot := NewSlot(startTime, slotDuration, slotNumber)

	if time.Now().After(startTime) {
		slot = NewSlot(startTime.Add(6*time.Second), slotDuration, slotNumber+1)
	}

	block := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochData, *slot)
	block.Header.Hash()

	// create new block for same slot
	block2 := createTestBlockWithSlot(t, babeService, &genesisHeader, [][]byte{}, epochData, *slot)
	block2.Header.Hash()

	err = verificationManager.VerifyBlock(&block.Header)
	require.NoError(t, err)

	err = verificationManager.VerifyBlock(&block2.Header)
	require.ErrorIs(t, err, ErrProducerEquivocated)
	require.EqualError(t, err, fmt.Sprintf("%s for block header %s", ErrProducerEquivocated, block2.Header.Hash()))
}

func TestVerifyForkBlocksWithRespectiveEpochData(t *testing.T) {
	/*
	* Setup the services: StateService, DigestHandler, EpochState
	* and VerificationManager
	 */
	keyPairs := []*sr25519.Keypair{
		keyring.KeyAlice, keyring.KeyBob, keyring.KeyCharlie,
		keyring.KeyDave, keyring.KeyEve, keyring.KeyFerdie,
		keyring.KeyGeorge, keyring.KeyHeather, keyring.KeyIan,
	}

	authorities := make([]types.AuthorityRaw, len(keyPairs))
	for i, keyPair := range keyPairs {
		authorities[i] = types.AuthorityRaw{
			Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
		}
	}

	// starts with only 3 authorities in the authority set
	epochBABEConfig := &types.BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        10,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: authorities[:3],
		Randomness:         [32]byte{},
		SecondarySlots:     0,
	}

	genesis, trie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(
		telemetry.NewNotifyFinalized(
			genesisHeader.Hash(),
			fmt.Sprint(genesisHeader.Number),
		),
	)

	stateService := state.NewService(state.Config{
		Path:              t.TempDir(),
		Telemetry:         telemetryMock,
		GenesisBABEConfig: config.BABEConfigurationTestDefault,
	})

	stateService.UseMemDB()

	err := stateService.Initialise(&genesis, &genesisHeader, trie)
	require.NoError(t, err)

	inMemoryDB, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)

	epochState, err := state.NewEpochStateFromGenesis(inMemoryDB, stateService.Block, epochBABEConfig)
	require.NoError(t, err)

	onBlockImportDigestHandler := digest.NewBlockImportHandler(epochState, stateService.Grandpa)

	digestHandler, err := digest.NewHandler(stateService.Block, epochState, stateService.Grandpa)
	require.NoError(t, err)

	digestHandler.Start()

	verificationManager := NewVerificationManager(stateService.Block, stateService.Slot, epochState)

	/*
	* lets issue different blocks starting from genesis (a fork)
	 */
	aliceBlockNextEpoch := types.NextEpochData{
		Authorities: authorities[3:],
	}
	aliceBlockNextConfigData := types.NextConfigDataV1{
		C1:             9,
		C2:             10,
		SecondarySlots: 1,
	}
	aliceBlockHeader := issueConsensusDigestsBlockFromGenesis(t, &genesisHeader, keyring.KeyAlice,
		stateService, aliceBlockNextEpoch, aliceBlockNextConfigData, onBlockImportDigestHandler)

	bobBlockNextEpoch := types.NextEpochData{
		Authorities: authorities[6:],
	}
	bobBlockNextConfigData := types.NextConfigDataV1{
		C1:             3,
		C2:             8,
		SecondarySlots: 1,
	}
	bobBlockHeader := issueConsensusDigestsBlockFromGenesis(t, &genesisHeader, keyring.KeyBob,
		stateService, bobBlockNextEpoch, bobBlockNextConfigData, onBlockImportDigestHandler)

	// wait for digest handleBlockImport goroutine gets the imported
	// block, process its digest and store the info at epoch state
	time.Sleep(time.Second * 2)

	/*
	* Simulate a fork from the genesis file, the fork alice and the fork bob
	* contains different digest handlers.
	 */
	const chainLen = 5
	forkAliceChain := make([]types.Header, chainLen)
	forkBobChain := make([]types.Header, chainLen)

	forkAliceLastHeader := aliceBlockHeader
	forkBobLastHeader := bobBlockHeader

	for idx := 0; idx < chainLen; idx++ {
		forkAliceLastHeader = issueNewBlockFrom(t, forkAliceLastHeader,
			keyring.KeyAlice, stateService)

		forkBobLastHeader = issueNewBlockFrom(t, forkBobLastHeader,
			keyring.KeyBob, stateService)

		forkAliceChain[idx] = *forkAliceLastHeader
		forkBobChain[idx] = *forkBobLastHeader
	}

	// verify if each block from the fork alice get the right digest
	const epochToTest = 1

	expectedThreshold, err := CalculateThreshold(aliceBlockNextConfigData.C1,
		aliceBlockNextConfigData.C2, len(authorities[3:]))
	require.NoError(t, err)

	for _, headerToVerify := range forkAliceChain {
		verifierInfo, err := verificationManager.getVerifierInfo(epochToTest, &headerToVerify)
		require.NoError(t, err)

		require.Equal(t, len(authorities[3:]), len(verifierInfo.authorities))
		require.ElementsMatch(t, authorities[3:], verifierInfo.authorities)

		require.True(t, verifierInfo.secondarySlots)
		require.Equal(t, expectedThreshold, verifierInfo.threshold)
	}

	// each block from the fork bob should use the right digest

	expectedThreshold, err = CalculateThreshold(bobBlockNextConfigData.C1,
		bobBlockNextConfigData.C2, len(authorities[6:]))
	require.NoError(t, err)

	for _, headerToVerify := range forkBobChain {
		verifierInfo, err := verificationManager.getVerifierInfo(epochToTest, &headerToVerify)
		require.NoError(t, err)

		require.Equal(t, len(authorities[6:]), len(verifierInfo.authorities))
		// should keep the original authorities
		require.ElementsMatch(t, authorities[6:], verifierInfo.authorities)

		require.True(t, verifierInfo.secondarySlots)
		require.Equal(t, expectedThreshold, verifierInfo.threshold)
	}

	telemetryMock.EXPECT().SendMessage(
		telemetry.NewNotifyFinalized(
			forkBobLastHeader.Hash(),
			fmt.Sprint(forkBobLastHeader.Number),
		),
	)
	err = stateService.Block.SetFinalisedHash(forkBobLastHeader.Hash(), 1, 1)
	require.NoError(t, err)

	// wait for digest handleBlockFinalize goroutine gets the finalized
	// block, clean up the in memory data and store the finalized digest in db
	time.Sleep(time.Second * 2)

	// as a chain was finalized any block built upon it should use the database stored data
	blockUponFinalizedHeader := issueNewBlockFrom(t, forkBobLastHeader,
		keyring.KeyBob, stateService)

	verifierInfo, err := verificationManager.getVerifierInfo(epochToTest, blockUponFinalizedHeader)
	require.NoError(t, err)

	require.Equal(t, len(authorities[6:]), len(verifierInfo.authorities))
	// should keep the original authorities
	require.ElementsMatch(t, authorities[6:], verifierInfo.authorities)

	require.True(t, verifierInfo.secondarySlots)
	require.Equal(t, expectedThreshold, verifierInfo.threshold)
}

// issueConsensusDigestsBlocksFromGenesis will create different
// blocks that contains different consensus messages digests
func issueConsensusDigestsBlockFromGenesis(t *testing.T, genesisHeader *types.Header,
	kp *sr25519.Keypair, stateService *state.Service,
	nextEpoch types.NextEpochData, nextConfig types.NextConfigDataV1,
	onImportBlockDigestHandler *digest.BlockImportHandler) *types.Header {
	t.Helper()

	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(0), 0))
	require.NoError(t, err)

	babePrimaryPreDigest := types.BabePrimaryPreDigest{
		SlotNumber: 1,
		VRFOutput:  output,
		VRFProof:   proof,
	}

	preRuntimeDigest, err := babePrimaryPreDigest.ToPreRuntimeDigest()
	require.NoError(t, err)

	babeConsensusDigestNextEpoch := types.NewBabeConsensusDigest()
	require.NoError(t, babeConsensusDigestNextEpoch.SetValue(nextEpoch))

	babeConsensusDigestNextConfigData := types.NewBabeConsensusDigest()

	versionedNextConfigData := types.NewVersionedNextConfigData()
	versionedNextConfigData.SetValue(nextConfig)

	require.NoError(t, babeConsensusDigestNextConfigData.SetValue(versionedNextConfigData))

	nextEpochData, err := scale.Marshal(babeConsensusDigestNextEpoch)
	require.NoError(t, err)

	nextEpochConsensusDigest := types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              nextEpochData,
	}

	nextConfigData, err := scale.Marshal(babeConsensusDigestNextConfigData)
	require.NoError(t, err)

	nextConfigConsensusDigest := types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              nextConfigData,
	}

	digest := types.NewDigest()
	require.NoError(t, digest.Add(*preRuntimeDigest, nextEpochConsensusDigest, nextConfigConsensusDigest))

	headerWhoOwnsNextEpochDigest := &types.Header{
		ParentHash: genesisHeader.Hash(),
		Number:     1,
		Digest:     digest,
	}

	err = stateService.Block.AddBlock(&types.Block{
		Header: *headerWhoOwnsNextEpochDigest,
		Body:   *types.NewBody([]types.Extrinsic{}),
	})
	require.NoError(t, err)

	err = onImportBlockDigestHandler.HandleDigests(headerWhoOwnsNextEpochDigest)
	require.NoError(t, err)

	return headerWhoOwnsNextEpochDigest
}

// issueNewBlockFrom will create and store a new block following a chain
func issueNewBlockFrom(t *testing.T, parentHeader *types.Header,
	kp *sr25519.Keypair, stateService *state.Service) *types.Header {
	t.Helper()

	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	require.NoError(t, err)

	babePrimaryPreDigest := types.BabePrimaryPreDigest{
		SlotNumber: 1,
		VRFOutput:  output,
		VRFProof:   proof,
	}

	preRuntimeDigest, err := babePrimaryPreDigest.ToPreRuntimeDigest()
	require.NoError(t, err)

	digest := types.NewDigest()

	require.NoError(t, digest.Add(*preRuntimeDigest))

	header := &types.Header{
		ParentHash: parentHeader.Hash(),
		Number:     parentHeader.Number + 1,
		Digest:     digest,
	}

	err = stateService.Block.AddBlock(&types.Block{
		Header: *header,
		Body:   *types.NewBody([]types.Extrinsic{}),
	})
	require.NoError(t, err)

	return header
}

func TestVerifyBlockAfterSkippedEpochs(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	currentBABEConfig := AuthorOnEverySlotBABEConfig
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, currentBABEConfig)

	epoch0DataRaw := &types.EpochDataRaw{
		Authorities: currentBABEConfig.GenesisAuthorities,
		Randomness:  currentBABEConfig.Randomness,
	}
	epoch0ConfigData := &types.ConfigData{
		C1:             currentBABEConfig.C1,
		C2:             currentBABEConfig.C2,
		SecondarySlots: currentBABEConfig.SecondarySlots,
	}

	epochData := testBuildEpochData(t, epoch0DataRaw, epoch0ConfigData)
	epochData.authorityIndex = 0
	ed := &epochDescriptor{
		data:  epochData,
		epoch: 0,
	}

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)
	slotState := state.NewSlotState(db)

	verificationManager := NewVerificationManager(babeService.blockState, slotState, babeService.epochState)

	firstBlockSlot := Slot{
		start:    time.Unix(0, 0),
		duration: 6 * time.Second,
		number:   0,
	}

	firstBlock := createTestBlockToImport(t, babeService, &genesisHeader, ed, firstBlockSlot)

	err = verificationManager.VerifyBlock(&firstBlock.Header)
	require.NoError(t, err)

	// add to the state then we can use the chain first slot number
	// to calculate correctly the epochs from the next blocks
	err = babeService.blockState.AddBlock(firstBlock)
	require.NoError(t, err)

	epoch1DataRaw := &types.EpochDataRaw{
		Authorities: []types.AuthorityRaw{
			{Key: [32]byte(keyring.KeyAlice.Public().Encode()), Weight: 1},
			{Key: [32]byte(keyring.KeyBob.Public().Encode()), Weight: 1},
			{Key: [32]byte(keyring.KeyCharlie.Public().Encode()), Weight: 1},
		},
	}

	// setting epoch data raw for epoch 1 but the config remains the same
	// one that was set to epoch 0, the point is:
	// we are going to skip epoch 1 and produce a block on epoch 2
	// given that we're going to produce using the data that was stored on epoch 1
	// the validation should be able to use the correct epoch descriptor to validate the block
	err = babeService.epochState.(*state.EpochState).SetEpochDataRaw(1, epoch1DataRaw)
	require.NoError(t, err)

	epochDataToBuildSecondBlock := testBuildEpochData(t, epoch1DataRaw, epoch0ConfigData)
	epochData.authorityIndex = 3
	ed = &epochDescriptor{
		data:  epochDataToBuildSecondBlock,
		epoch: 2,
	}

	// now lets create another block but on epoch 2, skipping epoch 1
	// here is a littel scheme on how to find epoch 2 start slot number:
	// | 0 ---- 199 | 200 ---- 399 | 400 --- 599 |
	// |-- epoch 0 -|-- epoch 1 ---|-- epoch 2 --|
	epoch2StartTime := int64(((6 * time.Second) * 200 * 2).Seconds())
	secondBlockSlot := Slot{
		start:    time.Unix(epoch2StartTime, 0),
		duration: 6 * time.Second,
		number:   400,
	}

	secondBlock := createTestBlockToImport(t, babeService, &firstBlock.Header, ed, secondBlockSlot)
	err = verificationManager.VerifyBlock(&secondBlock.Header)
	require.NoError(t, err)
}

func createTestBlockToImport(t *testing.T, babeService *Service, parent *types.Header,
	epochDescriptor *epochDescriptor, slot Slot) *types.Block {
	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	preRuntimeDigest, err := claimSlot(epochDescriptor.epoch, slot.number, epochDescriptor.data, babeService.keypair)
	require.NoError(t, err)

	block, err := babeService.buildBlock(parent, slot, rt, epochDescriptor.data.authorityIndex, preRuntimeDigest)
	require.NoError(t, err)

	return block
}

func testBuildEpochData(t *testing.T, epochDataRaw *types.EpochDataRaw, configData *types.ConfigData) *epochData {
	t.Helper()

	threshold, err := CalculateThreshold(configData.C1, configData.C2, len(epochDataRaw.Authorities))
	require.NoError(t, err)

	return &epochData{
		randomness:   epochDataRaw.Randomness,
		authorities:  epochDataRaw.Authorities,
		threshold:    threshold,
		allowedSlots: types.AllowedSlots(configData.SecondarySlots),
	}
}
