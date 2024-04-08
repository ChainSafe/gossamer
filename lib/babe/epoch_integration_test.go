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
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, genesisBABEConfig)
	babeService.constants.epochLength = 20

	epochDescriptor, err := babeService.initiateEpoch(0)
	require.NoError(t, err)

	epochStartSlot, err := babeService.epochState.GetStartSlotForEpoch(0, genesisHeader.Hash())
	require.NoError(t, err)
	require.GreaterOrEqual(t, epochStartSlot, epochDescriptor.startSlot)
}

func TestInitiateEpoch_Epoch1And2(t *testing.T) {
	cfg := ServiceConfig{
		Authority: true,
	}
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, genesis, genesisTrie, genesisHeader, genesisBABEConfig)

	// epoch 1, check that genesis EpochData and ConfigData was properly set
	auth := types.AuthorityRaw{
		Key:    babeService.keypair.Public().(*sr25519.PublicKey).AsBytes(),
		Weight: 1,
	}

	data, err := babeService.epochState.GetEpochDataRaw(0, nil)
	require.NoError(t, err)

	data.Authorities = []types.AuthorityRaw{auth}
	err = babeService.epochState.(*state.EpochState).StoreEpochDataRaw(1, data)
	require.NoError(t, err)

	const authorityIndex = 0
	currentEpochNumber := uint64(0)
	ed, err := babeService.initiateEpoch(currentEpochNumber)
	require.NoError(t, err)

	expected := &epochData{
		randomness:     [32]byte{},
		authorities:    []types.AuthorityRaw{auth},
		authorityIndex: authorityIndex,
		threshold:      ed.data.threshold,
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

	// for epoch 2, set EpochData but not ConfigData
	edata := &types.EpochDataRaw{
		Authorities: ed.data.authorities,
		Randomness:  [32]byte{9},
	}

	err = babeService.epochState.(*state.EpochState).StoreEpochDataRaw(1, edata)
	require.NoError(t, err)

	expectedEpoch1Data := &epochData{
		randomness:     edata.Randomness,
		authorities:    edata.Authorities,
		authorityIndex: 0,
		threshold:      ed.data.threshold,
	}

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

	// for epoch 3, set EpochData and ConfigData
	edata = &types.EpochDataRaw{
		Authorities: ed.data.authorities,
		Randomness:  [32]byte{9},
	}

	err = babeService.epochState.(*state.EpochState).StoreEpochDataRaw(2, edata)
	require.NoError(t, err)

	cdata := &types.ConfigData{
		C1: 1,
		C2: 99,
	}

	err = babeService.epochState.(*state.EpochState).StoreConfigData(2, cdata)
	require.NoError(t, err)

	threshold, err := CalculateThreshold(cdata.C1, cdata.C2, 1)
	require.NoError(t, err)

	expectedEpochDescriptor := &EpochDescriptor{
		data: &epochData{
			randomness:     edata.Randomness,
			authorities:    edata.Authorities,
			authorityIndex: 0,
			threshold:      threshold,
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
	bs := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, genesisBABEConfig)

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
	err = epochState.StoreEpochDataRaw(1, data)
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
	err = epochState.StoreEpochDataRaw(7, data)
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
