// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var keyring, _ = keystore.NewSr25519Keyring()

func TestBabeService_checkAndSetFirstSlot(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockEpochState0 := NewMockEpochState(ctrl)
	mockEpochState1 := NewMockEpochState(ctrl)

	mockEpochState0.EXPECT().GetStartSlotForEpoch(uint64(0)).Return(uint64(1), nil)
	mockEpochState1.EXPECT().GetStartSlotForEpoch(uint64(0)).Return(uint64(99), nil)
	mockEpochState1.EXPECT().SetFirstSlot(uint64(1)).Return(nil)

	testBabeSecondaryPlainPreDigest := types.BabeSecondaryPlainPreDigest{
		SlotNumber: 1,
	}

	encDigest := newEncodedBabeDigest(t, testBabeSecondaryPlainPreDigest)
	header := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encDigest))

	block := &types.Block{
		Header: *header,
	}

	mockBlockState.EXPECT().GetBlockByNumber(uint(1)).Return(block, nil)
	mockBlockState.EXPECT().GetBlockByNumber(uint(1)).Return(block, nil)

	bs0 := &Service{
		epochState: mockEpochState0,
		blockState: mockBlockState,
	}

	bs1 := &Service{
		epochState: mockEpochState1,
		blockState: mockBlockState,
	}

	cases := []struct {
		name    string
		service *Service
	}{
		{
			name:    "should not set first slot, as it's already set correctly",
			service: bs0,
		},
		{
			name:    "should update first slot, as it's set incorrectly",
			service: bs1,
		},
	}

	for _, tc := range cases {
		err := tc.service.checkAndSetFirstSlot()
		require.NoError(t, err)
	}
}

func TestBabeService_getEpochDataAndStartSlot(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockEpochState0 := NewMockEpochState(ctrl)
	mockEpochState1 := NewMockEpochState(ctrl)
	mockEpochState2 := NewMockEpochState(ctrl)

	mockEpochState0.EXPECT().GetStartSlotForEpoch(uint64(0)).Return(uint64(1), nil)
	mockEpochState1.EXPECT().GetStartSlotForEpoch(uint64(1)).Return(uint64(201), nil)
	mockEpochState2.EXPECT().GetStartSlotForEpoch(uint64(1)).Return(uint64(201), nil)

	mockEpochState1.EXPECT().HasEpochData(uint64(1)).Return(true, nil)
	mockEpochState2.EXPECT().HasEpochData(uint64(1)).Return(true, nil)

	kp := keyring.Alice().(*sr25519.Keypair)
	authority := types.NewAuthority(kp.Public(), uint64(1))
	testEpochData := &types.EpochData{
		Randomness:  [32]byte{1},
		Authorities: []types.Authority{*authority},
	}

	mockEpochState1.EXPECT().GetEpochData(uint64(1), nil).Return(testEpochData, nil)
	mockEpochState2.EXPECT().GetEpochData(uint64(1), nil).Return(testEpochData, nil)

	mockEpochState1.EXPECT().HasConfigData(uint64(1)).Return(true, nil)
	mockEpochState2.EXPECT().HasConfigData(uint64(1)).Return(false, nil)

	testConfigData := &types.ConfigData{
		C1: 1,
		C2: 1,
	}

	mockEpochState1.EXPECT().GetConfigData(uint64(1), nil).Return(testConfigData, nil)

	testLatestConfigData := &types.ConfigData{
		C1: 1,
		C2: 2,
	}

	mockEpochState2.EXPECT().GetLatestConfigData().Return(testLatestConfigData, nil)

	testEpochDataEpoch0 := &types.EpochData{
		Randomness:  [32]byte{9},
		Authorities: []types.Authority{*authority},
	}

	mockEpochState0.EXPECT().GetLatestEpochData().Return(testEpochDataEpoch0, nil)
	mockEpochState0.EXPECT().GetLatestConfigData().Return(testConfigData, nil)

	bs0 := &Service{
		authority:  true,
		keypair:    kp,
		epochState: mockEpochState0,
		blockState: mockBlockState,
	}

	bs1 := &Service{
		authority:  true,
		keypair:    kp,
		epochState: mockEpochState1,
		blockState: mockBlockState,
	}

	bs2 := &Service{
		authority:  true,
		keypair:    kp,
		epochState: mockEpochState2,
		blockState: mockBlockState,
	}

	threshold0, err := CalculateThreshold(testConfigData.C1, testConfigData.C2, 1)
	require.NoError(t, err)

	threshold1, err := CalculateThreshold(testLatestConfigData.C1, testLatestConfigData.C2, 1)
	require.NoError(t, err)

	cases := []struct {
		name              string
		service           *Service
		epoch             uint64
		expected          *epochData
		expectedStartSlot uint64
	}{
		{
			name:    "should get epoch data for epoch 0",
			service: bs0,
			epoch:   0,
			expected: &epochData{
				randomness:     testEpochDataEpoch0.Randomness,
				authorities:    testEpochDataEpoch0.Authorities,
				authorityIndex: 0,
				threshold:      threshold0,
			},
			expectedStartSlot: 1,
		},
		{
			name:    "should get epoch data for epoch 1 with config data from epoch 1",
			service: bs1,
			epoch:   1,
			expected: &epochData{
				randomness:     testEpochData.Randomness,
				authorities:    testEpochData.Authorities,
				authorityIndex: 0,
				threshold:      threshold0,
			},
			expectedStartSlot: 201,
		},
		{
			name:    "should get epoch data for epoch 1 and config data for epoch 0",
			service: bs2,
			epoch:   1,
			expected: &epochData{
				randomness:     testEpochData.Randomness,
				authorities:    testEpochData.Authorities,
				authorityIndex: 0,
				threshold:      threshold1,
			},
			expectedStartSlot: 201,
		},
	}

	for _, tc := range cases {
		res, startSlot, err := tc.service.getEpochDataAndStartSlot(tc.epoch)
		require.NoError(t, err)
		require.Equal(t, tc.expected, res)
		require.Equal(t, tc.expectedStartSlot, startSlot)
	}
}
