// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGenerateWarpSyncProofBlockNotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock block state to return not found block
	blockStateMock := NewMockBlockState(ctrl)
	blockStateMock.EXPECT().GetHeader(common.EmptyHash).Return(nil, errors.New("not found")).AnyTimes()

	provider := &WarpSyncProofProvider{
		blockState: blockStateMock,
	}

	// Check errMissingStartBlock returned by provider
	_, err := provider.Generate(common.EmptyHash)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errMissingStartBlock)
}

func TestGenerateWarpSyncProofBlockNotFinalized(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock block state to return not found block
	bestBlockHeader := &types.Header{
		Number:     2,
		ParentHash: common.MustBlake2bHash([]byte("1")),
	}

	notFinalizedBlockHeader := &types.Header{
		Number:     3,
		ParentHash: common.MustBlake2bHash([]byte("2")),
	}

	blockStateMock := NewMockBlockState(ctrl)
	blockStateMock.EXPECT().GetHeader(notFinalizedBlockHeader.Hash()).Return(notFinalizedBlockHeader, nil).AnyTimes()
	blockStateMock.EXPECT().GetHighestFinalisedHeader().Return(bestBlockHeader, nil).AnyTimes()

	provider := &WarpSyncProofProvider{
		blockState: blockStateMock,
	}

	// Check errMissingStartBlock returned by provider
	_, err := provider.Generate(notFinalizedBlockHeader.Hash())
	assert.Error(t, err)
	assert.ErrorIs(t, err, errStartBlockNotFinalized)
}

//nolint:lll
func TestGenerateWarpSyncProofOk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	encodedJustification1 := []byte{42, 0, 0, 0, 0, 0, 0, 0, 236, 148, 26, 228, 225, 134, 98, 140, 150, 154, 23, 185, 43, 231, 172, 194, 69, 150, 27, 191, 202, 50, 108, 91, 220, 57, 214, 47, 202, 62, 70, 238, 10, 0, 0, 0, 0, 0, 0, 0, 4, 236, 148, 26, 228, 225, 134, 98, 140, 150, 154, 23, 185, 43, 231, 172, 194, 69, 150, 27, 191, 202, 50, 108, 91, 220, 57, 214, 47, 202, 62, 70, 238, 10, 0, 0, 0, 0, 0, 0, 0, 201, 232, 26, 136, 31, 77, 15, 194, 34, 200, 248, 43, 219, 148, 207, 56, 240, 171, 208, 221, 162, 202, 153, 209, 150, 27, 71, 207, 227, 102, 133, 32, 206, 74, 78, 26, 148, 166, 18, 67, 188, 76, 163, 200, 68, 249, 134, 28, 122, 74, 182, 69, 135, 90, 199, 52, 72, 109, 41, 12, 37, 18, 161, 4, 136, 220, 52, 23, 213, 5, 142, 196, 180, 80, 62, 12, 18, 234, 26, 10, 137, 190, 32, 15, 233, 137, 34, 66, 61, 67, 52, 1, 79, 166, 176, 238, 0}
	encodedJustification2 := []byte{50, 0, 0, 0, 0, 0, 0, 0, 236, 148, 26, 228, 225, 134, 98, 140, 150, 154, 23, 185, 43, 231, 172, 194, 69, 150, 27, 191, 202, 50, 108, 91, 220, 57, 214, 47, 202, 62, 70, 238, 10, 0, 0, 0, 0, 0, 0, 0, 4, 236, 148, 26, 228, 225, 134, 98, 140, 150, 154, 23, 185, 43, 231, 172, 194, 69, 150, 27, 191, 202, 50, 108, 91, 220, 57, 214, 47, 202, 62, 70, 238, 10, 0, 0, 0, 0, 0, 0, 0, 201, 232, 26, 136, 31, 77, 15, 194, 34, 200, 248, 43, 219, 148, 207, 56, 240, 171, 208, 221, 162, 202, 153, 209, 150, 27, 71, 207, 227, 102, 133, 32, 206, 74, 78, 26, 148, 166, 18, 67, 188, 76, 163, 200, 68, 249, 134, 28, 122, 74, 182, 69, 135, 90, 199, 52, 72, 109, 41, 12, 37, 18, 161, 4, 136, 220, 52, 23, 213, 5, 142, 196, 180, 80, 62, 12, 18, 234, 26, 10, 137, 190, 32, 15, 233, 137, 34, 66, 61, 67, 52, 1, 79, 166, 176, 238, 0}
	var blockHeaders []*types.Header

	blockStateMock := NewMockBlockState(ctrl)
	grandpaStateMock := NewMockGrandpaState(ctrl)

	for blockNumber := uint(1); blockNumber <= 10; blockNumber++ {
		// Create block header
		var header *types.Header
		parentHash := common.Hash{0x00}
		if blockNumber > 1 {
			parentHash = blockHeaders[blockNumber-2].Hash()
		}

		header = types.NewHeader(
			parentHash,
			common.Hash{byte(blockNumber)},
			common.Hash{byte(blockNumber)},
			blockNumber,
			types.Digest{},
		)

		blockHeaders = append(blockHeaders, header)

		// Mock block state responses
		blockStateMock.EXPECT().GetHeader(header.Hash()).Return(header, nil).AnyTimes()
		blockStateMock.EXPECT().GetHeaderByNumber(blockNumber).Return(header, nil).AnyTimes()

		// authorities set changes happens only in block 5
		if blockNumber < 5 {
			grandpaStateMock.EXPECT().GetAuthoritiesChangesFromBlock(blockNumber).Return([]uint{5}, nil).AnyTimes()
		} else if blockNumber == 5 {
			blockStateMock.EXPECT().GetJustification(header.Hash()).Return(encodedJustification1, nil).AnyTimes()
		} else {
			grandpaStateMock.EXPECT().GetAuthoritiesChangesFromBlock(blockNumber).Return([]uint{}, nil).AnyTimes()
		}
	}

	blockStateMock.EXPECT().GetHighestFinalisedHeader().Return(blockHeaders[len(blockHeaders)-1], nil).AnyTimes()
	blockStateMock.EXPECT().GetJustification(blockHeaders[len(blockHeaders)-1].Hash()).Return(encodedJustification2, nil).AnyTimes()

	provider := &WarpSyncProofProvider{
		blockState:   blockStateMock,
		grandpaState: grandpaStateMock,
	}

	proof, err := provider.Generate(blockHeaders[0].Hash())
	assert.NoError(t, err)

	expectedProof := []byte{
		0x4, 0x1c, 0xa4, 0x2, 0x25, 0x71, 0x86, 0xee, 0x43, 0x46, 0xfd, 0x2c, 0x9, 0xfe, 0xeb, 0x91, 0x17, 0x10, 0xe5, 0x88, 0x41, 0x89, 0xc3, 0xc7, 0x5f, 0xb5, 0x1, 0x1a, 0x75, 0x21, 0x37, 0x2f, 0xf9, 0x14, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2a, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xec, 0x94, 0x1a, 0xe4, 0xe1, 0x86, 0x62, 0x8c, 0x96, 0x9a, 0x17, 0xb9, 0x2b, 0xe7, 0xac, 0xc2, 0x45, 0x96, 0x1b, 0xbf, 0xca, 0x32, 0x6c, 0x5b, 0xdc, 0x39, 0xd6, 0x2f, 0xca, 0x3e, 0x46, 0xee, 0xa, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0xec, 0x94, 0x1a, 0xe4, 0xe1, 0x86, 0x62, 0x8c, 0x96, 0x9a, 0x17, 0xb9, 0x2b, 0xe7, 0xac, 0xc2, 0x45, 0x96, 0x1b, 0xbf, 0xca, 0x32, 0x6c, 0x5b, 0xdc, 0x39, 0xd6, 0x2f, 0xca, 0x3e, 0x46, 0xee, 0xa, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc9, 0xe8, 0x1a, 0x88, 0x1f, 0x4d, 0xf, 0xc2, 0x22, 0xc8, 0xf8, 0x2b, 0xdb, 0x94, 0xcf, 0x38, 0xf0, 0xab, 0xd0, 0xdd, 0xa2, 0xca, 0x99, 0xd1, 0x96, 0x1b, 0x47, 0xcf, 0xe3, 0x66, 0x85, 0x20, 0xce, 0x4a, 0x4e, 0x1a, 0x94, 0xa6, 0x12, 0x43, 0xbc, 0x4c, 0xa3, 0xc8, 0x44, 0xf9, 0x86, 0x1c, 0x7a, 0x4a, 0xb6, 0x45, 0x87, 0x5a, 0xc7, 0x34, 0x48, 0x6d, 0x29, 0xc, 0x25, 0x12, 0xa1, 0x4, 0x88, 0xdc, 0x34, 0x17, 0xd5, 0x5, 0x8e, 0xc4, 0xb4, 0x50, 0x3e, 0xc, 0x12, 0xea, 0x1a, 0xa, 0x89, 0xbe, 0x20, 0xf, 0xe9, 0x89, 0x22, 0x42, 0x3d, 0x43, 0x34, 0x1, 0x4f, 0xa6, 0xb0, 0xee, 0x0, 0x1,
	}
	assert.Equal(t, expectedProof, proof)
}
