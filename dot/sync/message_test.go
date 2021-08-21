package sync

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

func addTestBlocksToState(t *testing.T, depth int, blockState BlockState) {
	previousHash := blockState.BestBlockHash()
	previousNum, err := blockState.BestBlockNumber()
	require.Nil(t, err)

	for i := 1; i <= depth; i++ {
		//block := &types.Block{
		//	Header: &types.Header{
		//		ParentHash: previousHash,
		//		Number:     big.NewInt(int64(i)).Add(previousNum, big.NewInt(int64(i))),
		//		StateRoot:  trie.EmptyHash,
		//		Digest:     types.Digest{},
		//	},
		//	Body: &types.Body{},
		//}
		block := &types.BlockVdt{
			Header: types.HeaderVdt{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)).Add(previousNum, big.NewInt(int64(i))),
				StateRoot:  trie.EmptyHash,
				Digest:     types.NewDigestVdt(),
			},
			Body: types.Body{},
		}

		previousHash = block.Header.Hash()

		err := blockState.AddBlockVdt(block)
		require.Nil(t, err)
	}
}

func TestService_CreateBlockResponse_MaxSize(t *testing.T) {
	s := NewTestSyncer(t, false)
	addTestBlocksToState(t, int(maxResponseSize), s.blockState)

	start, err := variadic.NewUint64OrHash(uint64(1))
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     0,
		Max:           optional.NewUint32(false, 0),
	}

	resp, err := s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(128), resp.BlockData[127].Number())

	req = &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     0,
		Max:           optional.NewUint32(true, maxResponseSize+100),
	}

	resp, err = s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(128), resp.BlockData[127].Number())
}

func TestService_CreateBlockResponse_StartHash(t *testing.T) {
	s := NewTestSyncer(t, false)
	addTestBlocksToState(t, int(maxResponseSize), s.blockState)

	startHash, err := s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)

	start, err := variadic.NewUint64OrHash(startHash)
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     0,
		Max:           optional.NewUint32(false, 0),
	}

	resp, err := s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(1), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(128), resp.BlockData[127].Number())
}

func TestService_CreateBlockResponse_Descending(t *testing.T) {
	s := NewTestSyncer(t, false)
	addTestBlocksToState(t, int(maxResponseSize), s.blockState)

	startHash, err := s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)

	start, err := variadic.NewUint64OrHash(startHash)
	require.NoError(t, err)

	req := &network.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     1,
		Max:           optional.NewUint32(false, 0),
	}

	resp, err := s.CreateBlockResponse(req)
	require.NoError(t, err)
	require.Equal(t, int(maxResponseSize), len(resp.BlockData))
	require.Equal(t, big.NewInt(128), resp.BlockData[0].Number())
	require.Equal(t, big.NewInt(1), resp.BlockData[127].Number())
}

// tests the ProcessBlockRequestMessage method
func TestService_CreateBlockResponse(t *testing.T) {
	s := NewTestSyncer(t, false)
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
	bds = &types.BlockData{
		Hash:          bestHash,
		Receipt:       optional.NewBytes(true, []byte("asdf")),
		MessageQueue:  optional.NewBytes(true, []byte("ghjkl")),
		Justification: optional.NewBytes(true, []byte("qwerty")),
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
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     0,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgValue: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:   optional.NewHash(true, bestHash).Value(),
						Header: bestBlock.Header.AsOptional(),
						Body:   bestBlock.Body.AsOptional(),
					},
				},
			},
		},
		{
			description: "test get Header",
			value: &network.BlockRequestMessage{
				RequestedData: 1,
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     0,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgValue: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:   optional.NewHash(true, bestHash).Value(),
						Header: bestBlock.Header.AsOptional(),
						Body:   optional.NewBody(false, nil),
					},
				},
			},
		},
		{
			description: "test get Receipt",
			value: &network.BlockRequestMessage{
				RequestedData: 4,
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     0,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgValue: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:    optional.NewHash(true, bestHash).Value(),
						Header:  optional.NewHeader(false, nil),
						Body:    optional.NewBody(false, nil),
						Receipt: bds.Receipt,
					},
				},
			},
		},
		{
			description: "test get MessageQueue",
			value: &network.BlockRequestMessage{
				RequestedData: 8,
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     0,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgValue: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:         optional.NewHash(true, bestHash).Value(),
						Header:       optional.NewHeader(false, nil),
						Body:         optional.NewBody(false, nil),
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
