// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/stretchr/testify/require"
)

func TestInitiateEpoch_Epoch0(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)
	babeService.constants.epochLength = 20

	epochDescriptor, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	epochStartSlot, err := babeService.epochState.GetStartSlotForEpoch(0, genesisHeader.Hash())
	require.NoError(t, err)
	require.GreaterOrEqual(t, epochStartSlot, epochDescriptor.startSlot)
}

func TestInitiateEpoch_Epoch1Epoch2(t *testing.T) {
	cfg := ServiceConfig{
		Authority: true,
	}
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	epoch0DataRaw, err := babeService.epochState.GetEpochDataRaw(0, nil)
	require.NoError(t, err)

	// for epoch 1, change authorities and randomness
	epoch1DataRaw := *epoch0DataRaw
	epoch1DataRaw.Authorities = []types.AuthorityRaw{
		{
			Key:    [32]byte(keyring.KeyBob.Public().Encode()),
			Weight: 1,
		},
		// on epoch 1, babe service account will be at index 1
		{
			Key:    babeService.keypair.Public().(*sr25519.PublicKey).AsBytes(),
			Weight: 1,
		},
		{
			Key:    [32]byte(keyring.KeyCharlie.Public().Encode()),
			Weight: 1,
		},
	}
	epoch1DataRaw.Randomness = [32]byte{0x0f}
	err = babeService.epochState.(*state.EpochState).SetEpochDataRaw(1, &epoch1DataRaw)
	require.NoError(t, err)

	const authorityIndex = 0
	currentEpochNumber := uint64(0)
	ed, err := babeService.initiateEpoch(currentEpochNumber)
	require.NoError(t, err)

	const numOfAuthoritiesOnEpoch0 = 1
	expectedThreshold, err := CalculateThreshold(AuthorOnEverySlotBABEConfig.C1,
		AuthorOnEverySlotBABEConfig.C2, numOfAuthoritiesOnEpoch0)
	require.NoError(t, err)

	expected := &epochData{
		randomness: AuthorOnEverySlotBABEConfig.Randomness,
		authorities: []types.AuthorityRaw{
			{
				Key:    babeService.keypair.Public().(*sr25519.PublicKey).AsBytes(),
				Weight: 1,
			},
		},
		authorityIndex: authorityIndex,
		threshold:      expectedThreshold,
	}
	require.Equal(t, expected.randomness, ed.data.randomness)
	require.Equal(t, expected.authorityIndex, ed.data.authorityIndex)
	require.Equal(t, expected.threshold, ed.data.threshold)

	preRuntimeDigest, err := claimSlot(ed.epoch, ed.startSlot, ed.data, babeService.keypair)
	require.NoError(t, err)

	digest := types.NewDigest()
	err = digest.Add(*preRuntimeDigest)
	require.NoError(t, err)

	blockNumber1Header := state.AddBlockToState(t,
		babeService.blockState.(*state.BlockState), 1, digest, genesisHeader.Hash())

	for i, auth := range ed.data.authorities {
		require.Equal(t, expected.authorities[i], auth)
	}

	const numOfAuthoritiesOnEpoch1 = 3
	expectedThreshold, err = CalculateThreshold(AuthorOnEverySlotBABEConfig.C1,
		AuthorOnEverySlotBABEConfig.C2, numOfAuthoritiesOnEpoch1)
	require.NoError(t, err)

	expectedEpoch1Data := &epochData{
		randomness:     epoch1DataRaw.Randomness,
		authorities:    epoch1DataRaw.Authorities,
		authorityIndex: 1,
		threshold:      expectedThreshold,
	}

	// epoch 1 has only a different epoch data raw
	// the epoch config remains
	currentEpochNumber = uint64(1)
	ed, err = babeService.initiateEpoch(currentEpochNumber)
	require.NoError(t, err)
	require.Equal(t, expectedEpoch1Data.randomness, ed.data.randomness)
	require.Equal(t, expectedEpoch1Data.authorityIndex, ed.data.authorityIndex)
	require.Equal(t, expectedEpoch1Data.threshold, ed.data.threshold)

	for i, auth := range ed.data.authorities {
		require.Equal(t, expectedEpoch1Data.authorities[i], auth)
	}

	preRuntimeDigest, err = claimSlot(ed.epoch, ed.startSlot, ed.data, babeService.keypair)
	require.NoError(t, err)

	digest = types.NewDigest()
	err = digest.Add(*preRuntimeDigest)
	require.NoError(t, err)

	state.AddBlockToState(t,
		babeService.blockState.(*state.BlockState), 2, digest, blockNumber1Header.Hash())

	// for epoch 2, set EpochData and ConfigData
	epoch2DataRaw := &types.EpochDataRaw{
		Authorities: []types.AuthorityRaw{
			// on epoch 2, babe service account will be at index 0
			{
				Key:    babeService.keypair.Public().(*sr25519.PublicKey).AsBytes(),
				Weight: 1,
			},
			{
				Key:    [32]byte(keyring.KeyCharlie.Public().Encode()),
				Weight: 1,
			},
		},
		Randomness: [32]byte{9},
	}

	epoch2ConfigData := &types.ConfigData{
		C1: 1,
		C2: 99,
	}

	err = babeService.epochState.(*state.EpochState).SetEpochDataRaw(2, epoch2DataRaw)
	require.NoError(t, err)

	err = babeService.epochState.(*state.EpochState).StoreConfigData(2, epoch2ConfigData)
	require.NoError(t, err)

	const numOfAuthoritiesOnEpoch2 = 2
	expectedThreshold, err = CalculateThreshold(epoch2ConfigData.C1, epoch2ConfigData.C2, numOfAuthoritiesOnEpoch2)
	require.NoError(t, err)

	expectedEpochDescriptor := &epochDescriptor{
		data: &epochData{
			randomness:     epoch2DataRaw.Randomness,
			authorities:    epoch2DataRaw.Authorities,
			authorityIndex: 0,
			threshold:      expectedThreshold,
		},
		epoch:     2,
		startSlot: ed.endSlot,
		endSlot:   ed.endSlot + babeService.constants.epochLength,
	}

	currentEpochNumber = uint64(2)
	ed, err = babeService.initiateEpoch(currentEpochNumber)

	require.NoError(t, err)
	require.Equal(t, expectedEpochDescriptor, ed)
}

func TestIncrementEpoch(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	bs := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, AuthorOnEverySlotBABEConfig)

	next, err := bs.incrementEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), next)

	next, err = bs.incrementEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(2), next)

	epoch, err := bs.epochState.GetCurrentEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(2), epoch)
}

func TestService_getLatestEpochData_genesis(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	service, _, genesisCfg := newTestServiceSetupParameters(t, genesis, genesisTrie, genesisHeader)

	service.keypair = keyring.KeyAlice
	service.authority = true

	latestEpochData, err := service.getEpochData(0, &genesisHeader)
	require.NoError(t, err)

	threshold, err := CalculateThreshold(genesisCfg.C1, genesisCfg.C2, len(genesisCfg.GenesisAuthorities))
	require.NoError(t, err)

	require.Equal(t, genesisCfg.GenesisAuthorities, latestEpochData.authorities)
	require.Equal(t, threshold, latestEpochData.threshold)
	require.Equal(t, genesisCfg.Randomness, latestEpochData.randomness)
}

func TestService_getLatestEpochData_epochData(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	service, epochState, genesisCfg := newTestServiceSetupParameters(t, genesis, genesisTrie, genesisHeader)

	service.keypair = keyring.KeyAlice
	service.authority = true

	err := epochState.StoreCurrentEpoch(1)
	require.NoError(t, err)

	data := &types.EpochDataRaw{
		Authorities: genesisCfg.GenesisAuthorities,
		Randomness:  [types.RandomnessLength]byte{99, 88, 77},
	}
	err = epochState.SetEpochDataRaw(1, data)
	require.NoError(t, err)

	ed, err := service.getEpochData(1, &genesisHeader)
	require.NoError(t, err)

	threshold, err := CalculateThreshold(genesisCfg.C1, genesisCfg.C2, len(data.Authorities))
	require.NoError(t, err)
	require.Equal(t, data.Authorities, ed.authorities)
	require.Equal(t, data.Randomness, ed.randomness)
	require.Equal(t, threshold, ed.threshold)
}

func TestService_getLatestEpochData_configData(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	service, epochState, genesisCfg := newTestServiceSetupParameters(t, genesis, genesisTrie, genesisHeader)

	service.keypair = keyring.KeyAlice
	service.authority = true

	err := epochState.StoreCurrentEpoch(7)
	require.NoError(t, err)

	data := &types.EpochDataRaw{
		Authorities: genesisCfg.GenesisAuthorities,
		Randomness:  [types.RandomnessLength]byte{99, 88, 77},
	}
	err = epochState.SetEpochDataRaw(7, data)
	require.NoError(t, err)

	cfgData := &types.ConfigData{
		C1: 1,
		C2: 7,
	}
	// set config data for a previous epoch, ensure latest config data is used
	err = epochState.StoreConfigData(1, cfgData)
	require.NoError(t, err)

	ed, err := service.getEpochData(7, &genesisHeader)
	require.NoError(t, err)
	threshold, err := CalculateThreshold(cfgData.C1, cfgData.C2, len(data.Authorities))
	require.NoError(t, err)

	require.Equal(t, data.Authorities, ed.authorities)
	require.Equal(t, data.Randomness, ed.randomness)
	require.Equal(t, threshold, ed.threshold)
}
