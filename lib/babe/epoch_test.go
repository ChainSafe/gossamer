package babe

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestBabeService_checkAndSetFirstSlot(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockEpochState0 := NewMockEpochState(ctrl)
	mockEpochState1 := NewMockEpochState(ctrl)

	mockEpochState0.EXPECT().GetStartSlotForEpoch(gomock.Eq(uint64(0))).Return(uint64(1), nil)
	mockEpochState1.EXPECT().GetStartSlotForEpoch(gomock.Eq(uint64(0))).Return(uint64(99), nil)
	mockEpochState1.EXPECT().SetFirstSlot(gomock.Eq(uint64(1))).Return(nil)

	testBabeSecondaryPlainPreDigest := types.BabeSecondaryPlainPreDigest{
		SlotNumber: 1,
	}

	encDigest := newEncodedBabeDigest(t, testBabeSecondaryPlainPreDigest)
	header := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encDigest))

	block := &types.Block{
		Header: *header,
	}

	mockBlockState.EXPECT().GetBlockByNumber(gomock.Eq(big.NewInt(1))).Return(block, nil)
	mockBlockState.EXPECT().GetBlockByNumber(gomock.Eq(big.NewInt(1))).Return(block, nil)

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
