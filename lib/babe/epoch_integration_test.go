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
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, nil)
	babeService.constants.epochLength = 20
	startSlot := uint64(1000)

	err := babeService.epochState.SetFirstSlot(startSlot)
	require.NoError(t, err)
	_, err = babeService.initiateEpoch(0)
	require.NoError(t, err)

	startSlot, err = babeService.epochState.GetStartSlotForEpoch(0)
	require.NoError(t, err)
	require.Greater(t, startSlot, uint64(1))
}

func TestInitiateEpoch_Epoch1(t *testing.T) {
	cfg := ServiceConfig{
		Authority: true,
	}
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, genesis, genesisTrie, genesisHeader, nil)
	babeService.constants.epochLength = 10

	state.AddBlocksToState(t, babeService.blockState.(*state.BlockState), 1, false)

	// epoch 1, check that genesis EpochData and ConfigData was properly set
	auth := types.Authority{
		Key:    babeService.keypair.Public().(*sr25519.PublicKey),
		Weight: 1,
	}

	data, err := babeService.epochState.GetEpochData(0, nil)
	require.NoError(t, err)
	data.Authorities = []types.Authority{auth}
	err = babeService.epochState.(*state.EpochState).SetEpochData(1, data)
	require.NoError(t, err)

	ed, err := babeService.initiateEpoch(1)
	require.NoError(t, err)

	expected := &epochData{
		randomness:     [32]byte{},
		authorities:    []types.Authority{auth},
		authorityIndex: 0,
		threshold:      ed.threshold,
	}
	require.Equal(t, expected.randomness, ed.randomness)
	require.Equal(t, expected.authorityIndex, ed.authorityIndex)
	require.Equal(t, expected.threshold, ed.threshold)

	for i, auth := range ed.authorities {
		expAuth, err := expected.authorities[i].Encode()
		require.NoError(t, err)
		res, err := auth.Encode()
		require.NoError(t, err)
		require.Equal(t, expAuth, res)
	}

	// for epoch 2, set EpochData but not ConfigData
	edata := &types.EpochData{
		Authorities: ed.authorities,
		Randomness:  [32]byte{9},
	}

	err = babeService.epochState.(*state.EpochState).SetEpochData(2, edata)
	require.NoError(t, err)

	expected = &epochData{
		randomness:     edata.Randomness,
		authorities:    edata.Authorities,
		authorityIndex: 0,
		threshold:      ed.threshold,
	}

	ed, err = babeService.initiateEpoch(2)
	require.NoError(t, err)
	require.Equal(t, expected.randomness, ed.randomness)
	require.Equal(t, expected.authorityIndex, ed.authorityIndex)
	require.Equal(t, expected.threshold, ed.threshold)

	for i, auth := range ed.authorities {
		expAuth, err := expected.authorities[i].Encode()
		require.NoError(t, err)
		res, err := auth.Encode()
		require.NoError(t, err)
		require.Equal(t, expAuth, res)
	}

	// for epoch 3, set EpochData and ConfigData
	edata = &types.EpochData{
		Authorities: ed.authorities,
		Randomness:  [32]byte{9},
	}

	err = babeService.epochState.(*state.EpochState).SetEpochData(3, edata)
	require.NoError(t, err)

	cdata := &types.ConfigData{
		C1: 1,
		C2: 99,
	}

	err = babeService.epochState.(*state.EpochState).SetConfigData(3, cdata)
	require.NoError(t, err)

	threshold, err := CalculateThreshold(cdata.C1, cdata.C2, 1)
	require.NoError(t, err)

	expected = &epochData{
		randomness:     edata.Randomness,
		authorities:    edata.Authorities,
		authorityIndex: 0,
		threshold:      threshold,
	}
	ed, err = babeService.initiateEpoch(3)
	require.NoError(t, err)
	require.Equal(t, expected, ed)
}

func TestIncrementEpoch(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	bs := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, nil)

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

	latestEpochData, err := service.getLatestEpochData()
	require.NoError(t, err)
	auths, err := types.BABEAuthorityRawToAuthority(genesisCfg.GenesisAuthorities)
	require.NoError(t, err)
	threshold, err := CalculateThreshold(genesisCfg.C1, genesisCfg.C2, len(auths))
	require.NoError(t, err)

	require.Equal(t, auths, latestEpochData.authorities)
	require.Equal(t, threshold, latestEpochData.threshold)
	require.Equal(t, genesisCfg.Randomness, latestEpochData.randomness)
}

func TestService_getLatestEpochData_epochData(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	service, epochState, genesisCfg := newTestServiceSetupParameters(t, genesis, genesisTrie, genesisHeader)

	err := epochState.SetCurrentEpoch(1)
	require.NoError(t, err)

	auths, err := types.BABEAuthorityRawToAuthority(genesisCfg.GenesisAuthorities)
	require.NoError(t, err)

	data := &types.EpochData{
		Authorities: auths,
		Randomness:  [types.RandomnessLength]byte{99, 88, 77},
	}
	err = epochState.SetEpochData(1, data)
	require.NoError(t, err)

	ed, err := service.getLatestEpochData()
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

	err := epochState.SetCurrentEpoch(7)
	require.NoError(t, err)

	auths, err := types.BABEAuthorityRawToAuthority(genesisCfg.GenesisAuthorities)
	require.NoError(t, err)

	data := &types.EpochData{
		Authorities: auths,
		Randomness:  [types.RandomnessLength]byte{99, 88, 77},
	}
	err = epochState.SetEpochData(7, data)
	require.NoError(t, err)

	cfgData := &types.ConfigData{
		C1: 1,
		C2: 7,
	}
	err = epochState.SetConfigData(1, cfgData) // set config data for a previous epoch, ensure latest config data is used
	require.NoError(t, err)

	ed, err := service.getLatestEpochData()
	require.NoError(t, err)
	threshold, err := CalculateThreshold(cfgData.C1, cfgData.C2, len(data.Authorities))
	require.NoError(t, err)

	require.Equal(t, data.Authorities, ed.authorities)
	require.Equal(t, data.Randomness, ed.randomness)
	require.Equal(t, threshold, ed.threshold)
}
