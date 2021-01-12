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
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/require"
)

func newEpochStateFromGenesis(t *testing.T) *EpochState {
	db := chaindb.NewMemDatabase()
	s, err := NewEpochStateFromGenesis(db, genesisBABEConfig)
	require.NoError(t, err)
	return s
}

func TestLoadStoreEpochLength(t *testing.T) {
	db := chaindb.NewMemDatabase()
	length := uint64(2222)
	err := storeEpochLength(db, length)
	require.NoError(t, err)

	ret, err := loadEpochLength(db)
	require.NoError(t, err)
	require.Equal(t, length, ret)
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

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	auth := &types.Authority{
		Key:    keyring.Alice().Public().(*sr25519.PublicKey),
		Weight: 1,
	}

	info := &types.EpochData{
		Authorities: []*types.Authority{auth},
		Randomness:  [32]byte{77},
	}

	err = s.SetEpochData(2, info)
	require.NoError(t, err)
	res, err := s.GetEpochData(2)
	require.NoError(t, err)
	require.Equal(t, info.Randomness, res.Randomness)

	for i, auth := range res.Authorities {
		expected, err := info.Authorities[i].Encode()
		require.NoError(t, err)
		res, err := auth.Encode()
		require.NoError(t, err)
		require.Equal(t, expected, res)
	}
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

func TestEpochState_ConfigData(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	data := &types.ConfigData{
		C1:             1,
		C2:             8,
		SecondarySlots: true,
	}

	err := s.SetConfigData(1, data)
	require.NoError(t, err)

	ret, err := s.GetConfigData(1)
	require.NoError(t, err)
	require.Equal(t, data, ret)
}

func TestEpochState_GetEpochForBlock(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	babeHeader := &types.BabeHeader{
		SlotNumber: 10,
	}

	enc := babeHeader.Encode()
	digest := types.NewBABEPreRuntimeDigest(enc)

	header := &types.Header{
		Digest: types.Digest{digest},
	}

	epoch, err := s.GetEpochForBlock(header)
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)

	babeHeader = &types.BabeHeader{
		SlotNumber: 210,
	}

	enc = babeHeader.Encode()
	digest = types.NewBABEPreRuntimeDigest(enc)

	header = &types.Header{
		Digest: types.Digest{digest},
	}

	epoch, err = s.GetEpochForBlock(header)
	require.NoError(t, err)
	require.Equal(t, uint64(2), epoch)
}
