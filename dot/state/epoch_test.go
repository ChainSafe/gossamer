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
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/require"
)

var genesisBABEConfig = &types.BabeConfiguration{
	SlotDuration:       1000,
	EpochLength:        200,
	C1:                 1,
	C2:                 4,
	GenesisAuthorities: []types.AuthorityRaw{},
	Randomness:         [32]byte{},
	SecondarySlots:     0,
}

func newEpochStateFromGenesis(t *testing.T) *EpochState {
	db := NewInMemoryDB(t)
	s, err := NewEpochStateFromGenesis(db, newTestBlockState(t, nil), genesisBABEConfig)
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
	require.Equal(t, uint64(0), epoch)

	err = s.SetCurrentEpoch(1)
	require.NoError(t, err)
	epoch, err = s.GetCurrentEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)
}

func TestEpochState_EpochData(t *testing.T) {
	s := newEpochStateFromGenesis(t)
	has, err := s.HasEpochData(0)
	require.NoError(t, err)
	require.True(t, has)

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	auth := types.Authority{
		Key:    keyring.Alice().Public().(*sr25519.PublicKey),
		Weight: 1,
	}

	info := &types.EpochData{
		Authorities: []types.Authority{auth},
		Randomness:  [32]byte{77},
	}

	err = s.SetEpochData(1, info)
	require.NoError(t, err)
	res, err := s.GetEpochData(1)
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
	require.Equal(t, uint64(1)+s.epochLength, start)

	start, err = s.GetStartSlotForEpoch(2)
	require.NoError(t, err)
	require.Equal(t, genesisBABEConfig.EpochLength*2+1, start)
}

func TestEpochState_ConfigData(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	data := &types.ConfigData{
		C1:             1,
		C2:             8,
		SecondarySlots: 1,
	}

	err := s.SetConfigData(1, data)
	require.NoError(t, err)

	ret, err := s.GetConfigData(1)
	require.NoError(t, err)
	require.Equal(t, data, ret)

	ret, err = s.GetLatestConfigData()
	require.NoError(t, err)
	require.Equal(t, data, ret)
}

func TestEpochState_GetEpochForBlock(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	babeHeader := types.NewBabePrimaryPreDigest(0, s.epochLength+2, [32]byte{}, [64]byte{})
	enc := babeHeader.Encode()
	d := types.NewBABEPreRuntimeDigest(enc)
	digest := types.NewDigest()
	digest.Add(*d)

	header := &types.Header{
		Digest: digest,
	}

	epoch, err := s.GetEpochForBlock(header)
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)

	babeHeader = types.NewBabePrimaryPreDigest(0, s.epochLength*2+3, [32]byte{}, [64]byte{})
	enc = babeHeader.Encode()
	d = types.NewBABEPreRuntimeDigest(enc)
	digest2 := types.NewDigest()
	digest2.Add(*d)

	header = &types.Header{
		Digest: digest2,
	}

	epoch, err = s.GetEpochForBlock(header)
	require.NoError(t, err)
	require.Equal(t, uint64(2), epoch)
}

func TestEpochState_SetAndGetSlotDuration(t *testing.T) {
	s := newEpochStateFromGenesis(t)
	expected := time.Millisecond * time.Duration(genesisBABEConfig.SlotDuration)

	ret, err := s.GetSlotDuration()
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}

func TestEpochState_GetEpochFromTime(t *testing.T) {
	s := newEpochStateFromGenesis(t)
	s.blockState = newTestBlockState(t, testGenesisHeader)

	epochDuration, err := time.ParseDuration(fmt.Sprintf("%dms", genesisBABEConfig.SlotDuration*genesisBABEConfig.EpochLength))
	require.NoError(t, err)

	slotDuration := time.Millisecond * time.Duration(genesisBABEConfig.SlotDuration)

	start := time.Unix(1, 0) // let's say first slot is 1 second after January 1, 1970 UTC
	slot := uint64(start.UnixNano()) / uint64(slotDuration.Nanoseconds())

	err = s.SetFirstSlot(slot)
	require.NoError(t, err)

	epoch, err := s.GetEpochFromTime(start)
	require.NoError(t, err)
	require.Equal(t, uint64(0), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration))
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration / 2))
	require.NoError(t, err)
	require.Equal(t, uint64(0), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration * 3 / 2))
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration*100 + 1))
	require.NoError(t, err)
	require.Equal(t, uint64(100), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration*100 - 1))
	require.NoError(t, err)
	require.Equal(t, uint64(99), epoch)
}
