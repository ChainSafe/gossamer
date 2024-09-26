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
	blockStateMock.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil).AnyTimes()

	provider := &WarpSyncProofProvider{
		blockState: blockStateMock,
	}

	// Check errMissingStartBlock returned by provider
	_, err := provider.Generate(notFinalizedBlockHeader.Hash())
	assert.Error(t, err)
	assert.ErrorIs(t, err, errStartBlockNotFinalized)
}

/*func TestGenerateWarpSyncProofOk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const blocksAmount = 10
	encodedJustification := []byte{0x1}
	blockHeaders := []*types.Header{}

	blockStateMock := NewMockBlockState(ctrl)
	grandpaStateMock := NewMockGrandpaState(ctrl)

	for blockNumber := uint(1); blockNumber <= blocksAmount; blockNumber++ {
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
			grandpaStateMock.EXPECT().GetAuthoritesChangesFromBlock(blockNumber).Return([]uint{5}, nil).AnyTimes()
		} else if blockNumber == 5 {
			blockStateMock.EXPECT().GetJustification(header.Hash()).Return(encodedJustification, nil).AnyTimes()
		} else {
			grandpaStateMock.EXPECT().GetAuthoritesChangesFromBlock(blockNumber).Return([]uint{}, nil).AnyTimes()
		}
	}

	blockStateMock.EXPECT().BestBlockHeader().Return(blockHeaders[len(blockHeaders)-1], nil).AnyTimes()

	provider := &WarpSyncProofProvider{
		blockState:   blockStateMock,
		grandpaState: grandpaStateMock,
	}

	proof, err := provider.Generate(blockHeaders[0].Hash())
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x01}, proof)
}*/
