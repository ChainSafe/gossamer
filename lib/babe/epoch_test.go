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
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/stretchr/testify/require"
)

func TestInitiateEpoch_Epoch0(t *testing.T) {
	bs := createTestService(t, nil)
	bs.epochLength = 5
	startSlot := uint64(1000)

	err := bs.epochState.SetFirstSlot(startSlot)
	require.NoError(t, err)
	err = bs.initiateEpoch(0)
	require.NoError(t, err)

	count := 0
	for i := startSlot; i < startSlot+bs.epochLength; i++ {
		_, has := bs.slotToProof[i]
		if has {
			count++
		}
	}
	require.GreaterOrEqual(t, count, 1)
}

func TestInitiateEpoch_Epoch1(t *testing.T) {
	bs := createTestService(t, nil)
	bs.epochLength = 5

	err := bs.initiateEpoch(0)
	require.NoError(t, err)

	state.AddBlocksToState(t, bs.blockState.(*state.BlockState), 1)

	// epoch 1, check that genesis EpochData and ConfigData was properly set
	threshold := bs.epochData.threshold

	auth := &types.Authority{
		Key:    bs.keypair.Public().(*sr25519.PublicKey),
		Weight: 1,
	}
	err = bs.initiateEpoch(1)
	require.NoError(t, err)

	expected := &epochData{
		randomness:     genesisBABEConfig.Randomness,
		authorities:    []*types.Authority{auth},
		authorityIndex: 0,
		threshold:      threshold,
	}
	require.Equal(t, expected, bs.epochData)
	require.GreaterOrEqual(t, len(bs.slotToProof), 1)

	// for epoch 2, set EpochData but not ConfigData
	edata := &types.EpochData{
		Authorities: bs.epochData.authorities,
		Randomness:  [32]byte{9},
	}

	err = bs.epochState.(*state.EpochState).SetEpochData(2, edata)
	require.NoError(t, err)

	expected = &epochData{
		randomness:     edata.Randomness,
		authorities:    edata.Authorities,
		authorityIndex: 0,
		threshold:      bs.epochData.threshold,
	}
	err = bs.initiateEpoch(2)
	require.NoError(t, err)
	require.Equal(t, expected.randomness, bs.epochData.randomness)
	require.Equal(t, expected.authorityIndex, bs.epochData.authorityIndex)
	require.Equal(t, expected.threshold, bs.epochData.threshold)
	require.GreaterOrEqual(t, len(bs.slotToProof), 1)

	for i, auth := range bs.epochData.authorities {
		expAuth, err := expected.authorities[i].Encode() //nolint
		require.NoError(t, err)
		res, err := auth.Encode()
		require.NoError(t, err)
		require.Equal(t, expAuth, res)
	}

	// for epoch 3, set EpochData and ConfigData
	edata = &types.EpochData{
		Authorities: bs.epochData.authorities,
		Randomness:  [32]byte{9},
	}

	err = bs.epochState.(*state.EpochState).SetEpochData(3, edata)
	require.NoError(t, err)

	cdata := &types.ConfigData{
		C1: 1,
		C2: 99,
	}

	err = bs.epochState.(*state.EpochState).SetConfigData(3, cdata)
	require.NoError(t, err)

	threshold, err = CalculateThreshold(cdata.C1, cdata.C2, 1)
	require.NoError(t, err)

	expected = &epochData{
		randomness:     edata.Randomness,
		authorities:    edata.Authorities,
		authorityIndex: 0,
		threshold:      threshold,
	}
	err = bs.initiateEpoch(3)
	require.NoError(t, err)
	require.Equal(t, expected, bs.epochData)

	time.Sleep(time.Second)
	require.GreaterOrEqual(t, len(bs.slotToProof), 1)
}

func TestIncrementEpoch(t *testing.T) {
	bs := createTestService(t, nil)
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
