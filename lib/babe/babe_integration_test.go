// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/stretchr/testify/require"
)

var genesisBABEConfig = &types.BabeConfiguration{
	// slots are 6 seconds on westend and using time.Now() allows us to create a block at any point in the slot.
	// So we need to manually set time to produce consistent results. See here:
	// https://github.com/paritytech/substrate/blob/09de7b41599add51cf27eca8f1bc4c50ed8e9453/frame/timestamp/src/lib.rs#L229
	// https://github.com/paritytech/substrate/blob/09de7b41599add51cf27eca8f1bc4c50ed8e9453/frame/timestamp/src/lib.rs#L206
	SlotDuration: 6000,
	EpochLength:  200,
	C1:           1,
	C2:           1,
	GenesisAuthorities: []types.AuthorityRaw{
		{
			Key:    keyring.Alice().Public().(*sr25519.PublicKey).AsBytes(),
			Weight: 1,
		},
	},
	Randomness:     [32]byte{},
	SecondarySlots: 0,
}

func TestService_SlotDuration(t *testing.T) {
	duration, err := time.ParseDuration("1000ms")
	require.NoError(t, err)

	bs := &Service{
		constants: constants{
			slotDuration: duration,
		},
	}

	dur := bs.constants.slotDuration
	require.Equal(t, dur.Milliseconds(), int64(1000))
}

func TestService_ProducesBlocks(t *testing.T) {
	cfg := ServiceConfig{
		Authority: true,
	}

	var authorOnEverySlot = &types.BabeConfiguration{
		SlotDuration: 6000,
		EpochLength:  200,
		C1:           1,
		C2:           1,
		GenesisAuthorities: []types.AuthorityRaw{
			{
				Key:    keyring.Alice().Public().(*sr25519.PublicKey).AsBytes(),
				Weight: 1,
			},
		},
		Randomness:     [32]byte{},
		SecondarySlots: 0,
	}

	gen, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, gen, genTrie, genHeader, authorOnEverySlot)

	err := babeService.Start()
	require.NoError(t, err)
	time.Sleep(babeService.constants.slotDuration * 2)
	err = babeService.Stop()
	require.NoError(t, err)

	bestHeader, err := babeService.blockState.BestBlockHeader()
	require.NoError(t, err)
	require.GreaterOrEqual(t, bestHeader.Number, uint(2))
}

func TestService_GetAuthorityIndex(t *testing.T) {
	kpA, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	kpB, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	pubA := kpA.Public().(*sr25519.PublicKey)
	pubB := kpB.Public().(*sr25519.PublicKey)

	authData := []types.AuthorityRaw{
		{Key: pubA.AsBytes(), Weight: 1},
		{Key: pubB.AsBytes(), Weight: 1},
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
	cfg := ServiceConfig{}
	gen, genTrie, genHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	bs := createTestService(t, cfg, gen, genTrie, genHeader, genesisBABEConfig)
	err := bs.Start()
	require.NoError(t, err)
	err = bs.Stop()
	require.NoError(t, err)
}

func TestService_PauseAndResume(t *testing.T) {
	// TODO: https://github.com/ChainSafe/gossamer/issues/3443
	t.Skip()

	cfg := ServiceConfig{}
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, genesis, genesisTrie, genesisHeader, genesisBABEConfig)
	err := babeService.Start()
	require.NoError(t, err)
	time.Sleep(time.Second)

	go func() {
		_ = babeService.Pause()
	}()

	go func() {
		_ = babeService.Pause()
	}()

	go func() {
		err := babeService.Resume()
		require.NoError(t, err)
	}()

	go func() {
		err := babeService.Resume()
		require.NoError(t, err)
	}()

	err = babeService.Stop()
	require.NoError(t, err)
}

func TestService_HandleSlotWithLaggingSlot(t *testing.T) {
	cfg := ServiceConfig{
		Authority: true,
	}

	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)

	babeService := createTestService(t, cfg, genesis, genesisTrie, genesisHeader, genesisBABEConfig)

	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	epochDescriptor, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	startTime := getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration)
	fmt.Println(startTime)

	slot := Slot{
		start:    startTime,
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}

	preRuntimeDigest, err := claimSlot(
		testEpochIndex, slot.number, epochDescriptor.data, babeService.keypair)
	require.NoError(t, err)

	const authorityIndex = 0 // alice
	builder := NewBlockBuilder(
		babeService.keypair,
		babeService.transactionState,
		babeService.blockState,
		authorityIndex,
		preRuntimeDigest,
	)

	block, err := builder.buildBlock(&genesisHeader, slot, rt)
	require.NoError(t, err)

	fmt.Println(epochDescriptor.startSlot)

	err = babeService.blockState.AddBlock(block)
	require.NoError(t, err)

	time.Sleep(babeService.constants.slotDuration)

	header, err := babeService.blockState.BestBlockHeader()
	require.NoError(t, err)

	bestBlockSlotNum, err := babeService.blockState.GetSlotForBlock(header.Hash())
	require.NoError(t, err)

	slotnum := uint64(1)
	slot = Slot{
		start:    time.Now(),
		duration: babeService.constants.slotDuration * time.Millisecond,
		number:   slotnum,
	}
	preRuntimeDigest, err = types.NewBabePrimaryPreDigest(
		0, slot.number,
		[sr25519.VRFOutputLength]byte{},
		[sr25519.VRFProofLength]byte{},
	).ToPreRuntimeDigest()
	require.NoError(t, err)

	slot = Slot{
		start:    time.Now(),
		duration: babeService.constants.slotDuration * time.Millisecond,
		number:   bestBlockSlotNum - 1,
	}
	err = babeService.handleSlot(
		epochDescriptor.epoch,
		slot,
		epochDescriptor.data.authorityIndex,
		preRuntimeDigest)
	require.ErrorIs(t, err, errLaggingSlot)
}

func TestService_HandleSlotWithSameSlot(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, genesisBABEConfig)
	const authorityIndex = 0

	bestBlockHash := babeService.blockState.BestBlockHash()
	runtime, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	epochDescriptor, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	startTimestamp := getSlotStartTime(epochDescriptor.startSlot, babeService.constants.slotDuration)
	slot := Slot{
		start:    startTimestamp,
		duration: babeService.constants.slotDuration,
		number:   epochDescriptor.startSlot,
	}

	preRuntimeDigest, err := claimSlot(
		epochDescriptor.epoch, slot.number, epochDescriptor.data, babeService.keypair)
	require.NoError(t, err)

	builder := NewBlockBuilder(
		babeService.keypair,
		babeService.transactionState,
		babeService.blockState,
		authorityIndex,
		preRuntimeDigest,
	)

	block, err := builder.buildBlock(&genesisHeader, slot, runtime)
	require.NoError(t, err)

	// Create new non authority service
	cfgBob := ServiceConfig{
		Keypair: keyring.Bob().(*sr25519.Keypair),
	}
	genBob, genTrieBob, genHeaderBob := newWestendDevGenesisWithTrieAndHeader(t)
	babeServiceBob := createTestService(t, cfgBob, genBob, genTrieBob, genHeaderBob, genesisBABEConfig)

	// Add block created by alice to bob
	err = babeServiceBob.blockState.AddBlock(block)
	require.NoError(t, err)

	// If the slot we are claiming is the same as the slot of the best block header, test that we can
	// still claim the slot without error.

	preRuntimeDigest, err = types.NewBabePrimaryPreDigest(
		0, slot.number,
		[sr25519.VRFOutputLength]byte{},
		[sr25519.VRFProofLength]byte{},
	).ToPreRuntimeDigest()
	require.NoError(t, err)

	// slot gets occupied even if it has been occupied by a block
	// authored by someone else
	err = babeServiceBob.handleSlot(
		testEpochIndex,
		slot,
		0,
		preRuntimeDigest)
	require.NoError(t, err)
}
