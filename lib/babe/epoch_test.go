// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
	kp := keyring.Alice().(*sr25519.Keypair)
	authority := types.NewAuthority(kp.Public(), uint64(1))
	testEpochData := &types.EpochData{
		Randomness:  [32]byte{1},
		Authorities: []types.Authority{*authority},
	}

	testConfigData := &types.ConfigData{
		C1: 1,
		C2: 1,
	}

	testLatestConfigData := &types.ConfigData{
		C1: 1,
		C2: 2,
	}

	testEpochDataEpoch0 := &types.EpochData{
		Randomness:  [32]byte{9},
		Authorities: []types.Authority{*authority},
	}

	threshold0, err := CalculateThreshold(testConfigData.C1, testConfigData.C2, 1)
	require.NoError(t, err)

	threshold1, err := CalculateThreshold(testLatestConfigData.C1, testLatestConfigData.C2, 1)
	require.NoError(t, err)

	cases := []struct {
		service           func(*gomock.Controller) *Service
		name              string
		epoch             uint64
		expected          *epochData
		expectedStartSlot uint64
	}{
		{
			name: "should_get_epoch_data_for_epoch_0",
			service: func(ctrl *gomock.Controller) *Service {
				mockEpochState := NewMockEpochState(ctrl)

				mockEpochState.EXPECT().GetLatestEpochData().Return(testEpochDataEpoch0, nil)
				mockEpochState.EXPECT().GetLatestConfigData().Return(testConfigData, nil)

				return &Service{
					authority:  true,
					keypair:    kp,
					epochState: mockEpochState,
				}
			},
			epoch: 0,
			expected: &epochData{
				randomness:     testEpochDataEpoch0.Randomness,
				authorities:    testEpochDataEpoch0.Authorities,
				authorityIndex: 0,
				threshold:      threshold0,
			},
			expectedStartSlot: 1,
		},
		{
			name: "should_get_epoch_data_for_epoch_1_with_config_data_from_epoch_1",
			service: func(ctrl *gomock.Controller) *Service {
				mockEpochState := NewMockEpochState(ctrl)

				mockEpochState.EXPECT().GetEpochData(uint64(1), nil).Return(testEpochData, nil)
				mockEpochState.EXPECT().GetConfigData(uint64(1), nil).Return(testConfigData, nil)

				return &Service{
					authority:  true,
					keypair:    kp,
					epochState: mockEpochState,
				}
			},
			epoch: 1,
			expected: &epochData{
				randomness:     testEpochData.Randomness,
				authorities:    testEpochData.Authorities,
				authorityIndex: 0,
				threshold:      threshold0,
			},
			expectedStartSlot: 201,
		},
		{
			name: "should_get_epoch_data_for_epoch_1_and_config_data_for_epoch_0",
			service: func(ctrl *gomock.Controller) *Service {
				mockEpochState := NewMockEpochState(ctrl)

				mockEpochState.EXPECT().GetEpochData(uint64(1), nil).Return(testEpochData, nil)
				mockEpochState.EXPECT().GetConfigData(uint64(1), nil).Return(testLatestConfigData, nil)

				return &Service{
					authority:  true,
					keypair:    kp,
					epochState: mockEpochState,
				}
			},
			epoch: 1,
			expected: &epochData{
				randomness:     testEpochData.Randomness,
				authorities:    testEpochData.Authorities,
				authorityIndex: 0,
				threshold:      threshold1,
			},
			expectedStartSlot: 201,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			service := tt.service(ctrl)

			res, err := service.getEpochData(tt.epoch, nil)
			require.NoError(t, err)
			require.Equal(t, tt.expected, res)
		})
	}
}
