package sync

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

func addTestBlocksToState(t *testing.T, depth int, blockState BlockState) {
	previousHash := blockState.BestBlockHash()
	previousNum, err := blockState.BestBlockNumber()
	require.Nil(t, err)

	for i := 1; i <= depth; i++ {
		block := &types.Block{
			Header: types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)).Add(previousNum, big.NewInt(int64(i))),
				StateRoot:  trie.EmptyHash,
				Digest:     types.NewDigest(),
			},
			Body: types.Body{},
		}

		previousHash = block.Header.Hash()

		err := blockState.AddBlock(block)
		require.Nil(t, err)
	}
}

func TestService_CreateBlockResponse_MaxSize(t *testing.T) {
	s := newTestSyncer(t)
	addTestBlocksToState(t, int(maxResponseSize*2), s.blockState)

	// test ascending
	start, err := variadic.NewUint64OrHash(uint64(1))
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Ascending,
		Max:           nil,
	}

	resp, err := s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(128), resp.BlockData[127].Number())

	max := uint32(maxResponseSize + 100)
	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Ascending,
		Max:           &max,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(128), resp.BlockData[127].Number())

	max = uint32(16)
	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Ascending,
		Max:           &max,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(max), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(16), resp.BlockData[15].Number())

	// test descending
	start, err = variadic.NewUint64OrHash(uint64(128))
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(128), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(1), resp.BlockData[127].Number())

	max = uint32(maxResponseSize + 100)
	start, err = variadic.NewUint64OrHash(uint64(256))
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Descending,
		Max:           &max,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(256), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(129), resp.BlockData[127].Number())

	max = uint32(16)
	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Descending,
		Max:           &max,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(max), len(resp.BlockData))
	require.Equal(t, big.NewInt(256), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(241), resp.BlockData[15].Number())
}

func TestService_CreateBlockResponse_StartHash(t *testing.T) {
	s := newTestSyncer(t)
	addTestBlocksToState(t, int(maxResponseSize*2), s.blockState)

	// test ascending with nil endBlockHash
	startHash, err := s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)

	start, err := variadic.NewUint64OrHash(startHash)
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Ascending,
		Max:           nil,
	}

	resp, err := s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(128), resp.BlockData[127].Number())

	endHash, err := s.blockState.GetHashByNumber(big.NewInt(16))
	require.NoError(t, err)

	// test ascending with non-nil endBlockHash
	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  &endHash,
		Direction:     network.Ascending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(16), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(16), resp.BlockData[15].Number())

	// test descending with nil endBlockHash
	startHash, err = s.blockState.GetHashByNumber(big.NewInt(16))
	require.NoError(t, err)

	start, err = variadic.NewUint64OrHash(startHash)
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(16), len(resp.BlockData))
	require.Equal(t, big.NewInt(16), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(1), resp.BlockData[15].Number())

	// test descending with non-nil endBlockHash
	endHash, err = s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  &endHash,
		Direction:     network.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(16), len(resp.BlockData))
	require.Equal(t, big.NewInt(16), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(1), resp.BlockData[15].Number())

	// test descending with nil endBlockHash and start > maxResponseSize
	startHash, err = s.blockState.GetHashByNumber(big.NewInt(256))
	require.NoError(t, err)

	start, err = variadic.NewUint64OrHash(startHash)
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(256), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(129), resp.BlockData[127].Number())

	startHash, err = s.blockState.GetHashByNumber(big.NewInt(128))
	require.NoError(t, err)

	start, err = variadic.NewUint64OrHash(startHash)
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  nil,
		Direction:     network.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(128), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(1), resp.BlockData[127].Number())
}

func TestService_CreateBlockResponse_Ascending_EndHash(t *testing.T) {
	t.Parallel()
	s := newTestSyncer(t)
	addTestBlocksToState(t, int(maxResponseSize+1), s.blockState)

	// should error if end < start
	start, err := variadic.NewUint64OrHash(uint64(128))
	require.NoError(t, err)

	end, err := s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  &end,
		Direction:     network.Ascending,
		Max:           nil,
	}

	_, err = s.CreateBlockResponse(req)
	require.Error(t, err)

	// base case
	start, err = variadic.NewUint64OrHash(uint64(1))
	require.NoError(t, err)

	end, err = s.blockState.GetHashByNumber(big.NewInt(128))
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  &end,
		Direction:     network.Ascending,
		Max:           nil,
	}

	resp, err := s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(128), resp.BlockData[127].Number())
}

func TestService_CreateBlockResponse_Descending_EndHash(t *testing.T) {
	s := newTestSyncer(t)
	addTestBlocksToState(t, int(maxResponseSize+1), s.blockState)

	// should error if start < end
	start, err := variadic.NewUint64OrHash(uint64(1))
	require.NoError(t, err)

	end, err := s.blockState.GetHashByNumber(big.NewInt(128))
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  &end,
		Direction:     network.Descending,
		Max:           nil,
	}

	_, err = s.CreateBlockResponse(req)
	require.Error(t, err)

	// base case
	start, err = variadic.NewUint64OrHash(uint64(128))
	require.NoError(t, err)

	end, err = s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		EndBlockHash:  &end,
		Direction:     network.Descending,
		Max:           nil,
	}

	resp, err := s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(128), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(1), resp.BlockData[127].Number())
}

func TestService_checkOrGetDescendantHash(t *testing.T) {
	t.Parallel()
	s := newTestSyncer(t)
	branches := map[int]int{
		8: 1,
	}
	state.AddBlocksToStateWithFixedBranches(t, s.blockState.(*state.BlockState), 16, branches, 1)

	// base case
	ancestor, err := s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)
	descendant, err := s.blockState.GetHashByNumber(big.NewInt(16))
	require.NoError(t, err)
	descendantNumber := big.NewInt(16)

	res, err := s.checkOrGetDescendantHash(ancestor, &descendant, descendantNumber)
	require.NoError(t, err)
	require.Equal(t, descendant, res)

	// supply descendant that's not on canonical chain
	leaves := s.blockState.(*state.BlockState).Leaves()
	require.Equal(t, 2, len(leaves))

	ancestor, err = s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)
	descendant, err = s.blockState.GetHashByNumber(big.NewInt(16))
	require.NoError(t, err)

	for _, leaf := range leaves {
		if !leaf.Equal(descendant) {
			descendant = leaf
			break
		}
	}

	res, err = s.checkOrGetDescendantHash(ancestor, &descendant, descendantNumber)
	require.NoError(t, err)
	require.Equal(t, descendant, res)

	// supply descedant that's not on same chain as ancestor
	ancestor, err = s.blockState.GetHashByNumber(big.NewInt(9))
	require.NoError(t, err)
	res, err = s.checkOrGetDescendantHash(ancestor, &descendant, descendantNumber)
	require.Error(t, err)

	// don't supply descendant, should return block on canonical chain
	// as ancestor is on canonical chain
	expected, err := s.blockState.GetHashByNumber(big.NewInt(16))
	require.NoError(t, err)

	res, err = s.checkOrGetDescendantHash(ancestor, nil, descendantNumber)
	require.NoError(t, err)
	require.Equal(t, expected, res)

	// don't supply descendant and provide ancestor not on canonical chain
	// should return descendant block also not on canonical chain
	block9s, err := s.blockState.GetAllBlocksAtNumber(big.NewInt(9))
	require.NoError(t, err)
	canonical, err := s.blockState.GetHashByNumber(big.NewInt(9))
	require.NoError(t, err)

	// set ancestor to non-canonical block 9
	for _, block := range block9s {
		if !canonical.Equal(block) {
			ancestor = block
			break
		}
	}

	// expected is non-canonical block 16
	for _, leaf := range leaves {
		is, err := s.blockState.IsDescendantOf(ancestor, leaf) //nolint
		require.NoError(t, err)
		if is {
			expected = leaf
			break
		}
	}

	res, err = s.checkOrGetDescendantHash(ancestor, nil, descendantNumber)
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

func TestService_CreateBlockResponse_Fields(t *testing.T) {
	s := newTestSyncer(t)
	addTestBlocksToState(t, 2, s.blockState)

	bestHash := s.blockState.BestBlockHash()
	bestBlock, err := s.blockState.GetBlockByNumber(big.NewInt(1))
	require.NoError(t, err)

	// set some nils and check no error is thrown
	bds := &types.BlockData{
		Hash:          bestHash,
		Header:        nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}
	err = s.blockState.CompareAndSetBlockData(bds)
	require.NoError(t, err)

	// set receipt message and justification
	a := []byte("asdf")
	b := []byte("ghjkl")
	c := []byte("qwerty")
	bds = &types.BlockData{
		Hash:          bestHash,
		Receipt:       &a,
		MessageQueue:  &b,
		Justification: &c,
	}

	endHash := s.blockState.BestBlockHash()
	start, err := variadic.NewUint64OrHash(uint64(1))
	require.NoError(t, err)

	err = s.blockState.CompareAndSetBlockData(bds)
	require.NoError(t, err)

	testCases := []struct {
		description      string
		value            *network.BlockRequestMessage
		expectedMsgValue *network.BlockResponseMessage
	}{
		{
			description: "test get Header and Body",
			value: &network.BlockRequestMessage{
				RequestedData: 3,
				StartingBlock: *start,
				EndBlockHash:  &endHash,
				Direction:     network.Ascending,
				Max:           nil,
			},
			expectedMsgValue: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:   bestHash,
						Header: &bestBlock.Header,
						Body:   &bestBlock.Body,
					},
				},
			},
		},
		{
			description: "test get Header",
			value: &network.BlockRequestMessage{
				RequestedData: 1,
				StartingBlock: *start,
				EndBlockHash:  &endHash,
				Direction:     network.Ascending,
				Max:           nil,
			},
			expectedMsgValue: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:   bestHash,
						Header: &bestBlock.Header,
						Body:   nil,
					},
				},
			},
		},
		{
			description: "test get Receipt",
			value: &network.BlockRequestMessage{
				RequestedData: 4,
				StartingBlock: *start,
				EndBlockHash:  &endHash,
				Direction:     network.Ascending,
				Max:           nil,
			},
			expectedMsgValue: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:    bestHash,
						Header:  nil,
						Body:    nil,
						Receipt: bds.Receipt,
					},
				},
			},
		},
		{
			description: "test get MessageQueue",
			value: &network.BlockRequestMessage{
				RequestedData: 8,
				StartingBlock: *start,
				EndBlockHash:  &endHash,
				Direction:     network.Ascending,
				Max:           nil,
			},
			expectedMsgValue: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:         bestHash,
						Header:       nil,
						Body:         nil,
						MessageQueue: bds.MessageQueue,
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resp, err := s.CreateBlockResponse(test.value)
			require.NoError(t, err)
			require.Len(t, resp.BlockData, 2)
			require.Equal(t, test.expectedMsgValue.BlockData[0].Hash, bestHash)
			require.Equal(t, test.expectedMsgValue.BlockData[0].Header, resp.BlockData[0].Header)
			require.Equal(t, test.expectedMsgValue.BlockData[0].Body, resp.BlockData[0].Body)

			if test.expectedMsgValue.BlockData[0].Receipt != nil {
				require.Equal(t, test.expectedMsgValue.BlockData[0].Receipt, resp.BlockData[1].Receipt)
			}

			if test.expectedMsgValue.BlockData[0].MessageQueue != nil {
				require.Equal(t, test.expectedMsgValue.BlockData[0].MessageQueue, resp.BlockData[1].MessageQueue)
			}

			if test.expectedMsgValue.BlockData[0].Justification != nil {
				require.Equal(t, test.expectedMsgValue.BlockData[0].Justification, resp.BlockData[1].Justification)
			}
		})
	}
}
