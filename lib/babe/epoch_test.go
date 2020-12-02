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

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/stretchr/testify/require"
)

func TestInitiateEpoch(t *testing.T) {
	bs := createTestService(t, nil)
	bs.epochLength = testEpochLength

	// epoch 1
	err := bs.initiateEpoch(1, testEpochLength+1)
	require.NoError(t, err)

	// add blocks w/ babe header to state
	parent := genesisHeader
	for i := 1; i < int(testEpochLength*2+1); i++ {
		block, _ := createTestBlock(t, bs, parent, nil, uint64(i))
		err = bs.blockState.AddBlock(block)
		require.NoError(t, err)
		parent = block.Header
	}

	// add epoch 2 info
	epochData := &types.EpochData{
		Authorities: bs.epochData.authorityData,
		Randomness:  [32]byte{9},
	}

	err = bs.epochState.(*state.EpochState).SetEpochData(2, epochData)
	require.NoError(t, err)

	// epoch 2
	state.AddBlocksToState(t, bs.blockState.(*state.BlockState), int(testEpochLength*2))
	err = bs.initiateEpoch(2, testEpochLength*2+1)
	require.NoError(t, err)

	// assert epoch info was stored
	has, err := bs.epochState.HasEpochData(1)
	require.NoError(t, err)
	require.True(t, has)

	has, err = bs.epochState.HasEpochData(2)
	require.NoError(t, err)
	require.True(t, has)

	// assert slot lottery was run for epochs 0, 1 and 2
	require.Equal(t, int(testEpochLength*3), len(bs.slotToProof))
}

func TestGetVRFOutput(t *testing.T) {
	bs := createTestService(t, nil)
	block, _ := createTestBlock(t, bs, genesisHeader, nil, 1)
	out, err := getVRFOutput(block.Header)
	require.NoError(t, err)
	require.Equal(t, bs.slotToProof[1].output, out)
}

func TestIncrementEpoch(t *testing.T) {
	bs := createTestService(t, nil)
	next, err := bs.incrementEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(2), next)

	next, err = bs.incrementEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(3), next)

	epoch, err := bs.epochState.GetCurrentEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(3), epoch)
}
