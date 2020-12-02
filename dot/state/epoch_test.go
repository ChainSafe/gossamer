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

package state

import (
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/stretchr/testify/require"
)

func newEpochStateFromGenesis(t *testing.T) *EpochState {
	db := chaindb.NewMemDatabase()
	s, err := NewEpochStateFromGenesis(db, genesisBABEConfig)
	require.NoError(t, err)
	return s
}

func TestNewEpochStateFromGenesis(t *testing.T) {
	_ = newEpochStateFromGenesis(t)
}

func TestEpochState_CurrentEpoch(t *testing.T) {
	s := newEpochStateFromGenesis(t)
	epoch, err := s.GetCurrentEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)

	err = s.SetCurrentEpoch(2)
	require.NoError(t, err)
	epoch, err = s.GetCurrentEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(2), epoch)
}

func TestEpochState_EpochData(t *testing.T) {
	s := newEpochStateFromGenesis(t)
	has, err := s.HasEpochData(1)
	require.NoError(t, err)
	require.True(t, has)

	info := &types.EpochData{
		Authorities: []*types.Authority{},
		Randomness:  [32]byte{77},
	}

	err = s.SetEpochData(2, info)
	require.NoError(t, err)
	res, err := s.GetEpochData(2)
	require.NoError(t, err)
	require.Equal(t, info, res)
}

func TestEpochState_GetStartSlotForEpoch(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	info := &types.EpochData{
		Randomness: [32]byte{77},
	}

	err := s.SetEpochData(2, info)
	require.NoError(t, err)

	info = &types.EpochData{
		Randomness: [32]byte{77},
	}

	err = s.SetEpochData(3, info)
	require.NoError(t, err)

	start, err := s.GetStartSlotForEpoch(0)
	require.NoError(t, err)
	require.Equal(t, uint64(1), start)

	start, err = s.GetStartSlotForEpoch(1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), start)

	start, err = s.GetStartSlotForEpoch(2)
	require.NoError(t, err)
	require.Equal(t, genesisBABEConfig.EpochLength+1, start)
}
