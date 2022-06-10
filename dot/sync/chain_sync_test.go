// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const defaultSlotDuration = 6 * time.Second

func Test_chainSyncState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    chainSyncState
		want string
	}{
		{
			name: "case bootstrap",
			s:    bootstrap,
			want: "bootstrap",
		},
		{
			name: "case tip",
			s:    tip,
			want: "tip",
		},
		{
			name: "case unknown",
			s:    3,
			want: "unknown",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.s.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChainSync_SetPeerHead(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)

	testPeer := peer.ID("noot")
	hash := common.Hash{0xa, 0xb}
	const number = 1000
	mockBlockState := NewMockBlockState(ctrl)
	mockHeader, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, 0,
		types.NewDigest())
	require.NoError(t, err)
	mockBlockState.EXPECT().BestBlockHeader().Return(mockHeader, nil)
	cs.blockState = mockBlockState

	err = cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)

	expected := &peerState{
		who:    testPeer,
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	require.Equal(t, expected, <-cs.workQueue)
	require.True(t, cs.pendingBlocks.hasBlock(hash))

	// test case where peer has a lower head than us, but they are on the same chain as us
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, number,
		types.NewDigest())
	require.NoError(t, err)
	mockBlockState = NewMockBlockState(ctrl)
	mockBlockState.EXPECT().BestBlockHeader().Return(header, nil)
	mockBlockState.EXPECT().GetHashByNumber(uint(999)).Return(hash, nil)
	cs.blockState = mockBlockState

	fin, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, number-2,
		types.NewDigest())
	require.NoError(t, err)

	err = cs.setPeerHead(testPeer, hash, number-1)
	require.NoError(t, err)
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number - 1,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	require.Len(t, cs.workQueue, 0)

	// test case where peer has a lower head than us, and they are on an invalid fork
	mockBlockState = NewMockBlockState(ctrl)
	mockBlockState.EXPECT().BestBlockHeader().Return(header, nil)
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, number,
		types.NewDigest())
	require.NoError(t, err)
	mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(fin, nil)
	mockBlockState.EXPECT().GetHashByNumber(uint(999)).Return(common.Hash{}, nil)
	cs.blockState = mockBlockState

	mockNetwork := NewMockNetwork(ctrl)
	mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  -4096,
		Reason: "Bad block announcement",
	}, peer.ID("noot"))
	cs.network = mockNetwork
	err = cs.setPeerHead(testPeer, hash, number-1)
	require.True(t, errors.Is(err, errPeerOnInvalidFork))
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number - 1,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	require.Len(t, cs.workQueue, 0)

	// test case where peer has a lower head than us, but they are on a valid fork (that is not our chain)
	mockBlockState = NewMockBlockState(ctrl)
	mockBlockState.EXPECT().BestBlockHeader().Return(header, nil)
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, number-2,
		types.NewDigest())
	require.NoError(t, err)
	mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(fin, nil)
	mockBlockState.EXPECT().GetHashByNumber(uint(999)).Return(common.Hash{}, nil)
	mockBlockState.EXPECT().HasHeader(common.Hash{10, 11}).Return(true, nil)
	cs.blockState = mockBlockState
	err = cs.setPeerHead(testPeer, hash, number-1)
	require.NoError(t, err)
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number - 1,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	require.Len(t, cs.workQueue, 0)
}

func TestChainSync_sync_bootstrap_withWorkerError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	mockBlockState := NewMockBlockState(ctrl)
	mockHeader, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, 0,
		types.NewDigest())
	require.NoError(t, err)
	mockBlockState.EXPECT().BestBlockHeader().Return(mockHeader, nil).Times(2)
	cs.blockState = mockBlockState
	cs.handler = newBootstrapSyncer(mockBlockState)

	mockNetwork := NewMockNetwork(ctrl)
	startingBlock := variadic.MustNewUint32OrHash(1)
	max := uint32(128)
	mockNetwork.EXPECT().DoBlockRequest(peer.ID("noot"), &network.BlockRequestMessage{
		RequestedData: 19,
		StartingBlock: *startingBlock,
		EndBlockHash:  nil,
		Direction:     0,
		Max:           &max,
	})
	cs.network = mockNetwork

	go cs.sync()
	defer cs.cancel()

	testPeer := peer.ID("noot")
	cs.peerState[testPeer] = &peerState{
		number: 1000,
	}

	cs.workQueue <- cs.peerState[testPeer]

	select {
	case res := <-cs.resultQueue:
		expected := &workerError{
			err: errNilResponse, // since MockNetwork returns a nil response
			who: testPeer,
		}
		require.Equal(t, expected, res.err)
	case <-time.After(5 * time.Second):
		t.Fatal("did not get worker response")
	}

	require.Equal(t, bootstrap, cs.state)
}

func TestChainSync_sync_tip(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	cs.blockState = new(mocks.BlockState)
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, 1000,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*mocks.BlockState).On("BestBlockHeader").Return(header, nil)
	cs.blockState.(*mocks.BlockState).On("GetHighestFinalisedHeader").Run(func(args mock.Arguments) {
		close(done)
	}).Return(header, nil)

	go cs.sync()
	defer cs.cancel()

	testPeer := peer.ID("noot")
	cs.peerState[testPeer] = &peerState{
		number: 999,
	}

	cs.workQueue <- cs.peerState[testPeer]
	<-done
	require.Equal(t, tip, cs.state)
}

func TestChainSync_getTarget(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	require.Equal(t, uint(1<<32-1), cs.getTarget())
	cs.peerState = map[peer.ID]*peerState{
		"a": {
			number: 0, // outlier
		},
		"b": {
			number: 110,
		},
		"c": {
			number: 120,
		},
		"d": {
			number: 130,
		},
		"e": {
			number: 140,
		},
		"f": {
			number: 150,
		},
		"g": {
			number: 1000, // outlier
		},
	}

	require.Equal(t, uint(130), cs.getTarget()) // sum:650/count:5= avg:130

	cs.peerState = map[peer.ID]*peerState{
		"testA": {
			number: 1000,
		},
		"testB": {
			number: 2000,
		},
	}

	require.Equal(t, uint(1500), cs.getTarget())
}

func TestWorkerToRequests(t *testing.T) {
	t.Parallel()

	w := &worker{
		startNumber:  uintPtr(10),
		targetNumber: uintPtr(1),
		direction:    network.Ascending,
	}
	_, err := workerToRequests(w)
	require.Equal(t, errInvalidDirection, err)

	type testCase struct {
		w        *worker
		expected []*network.BlockRequestMessage
	}

	var (
		max128 = uint32(128)
		max9   = uint32(9)
		max64  = uint32(64)
	)

	testCases := map[string]testCase{
		"test 0": {
			w: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(1 + maxResponseSize),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(1),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test 1": {
			w: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(1 + (maxResponseSize * 2)),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(1),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: *variadic.MustNewUint32OrHash(1 + maxResponseSize),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test 2": {
			w: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(10),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(1),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test 3": {
			w: &worker{
				startNumber:  uintPtr(10),
				targetNumber: uintPtr(1),
				direction:    network.Descending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(10),
					EndBlockHash:  nil,
					Direction:     network.Descending,
					Max:           &max9,
				},
			},
		},
		"test 4": {
			w: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(1 + maxResponseSize + (maxResponseSize / 2)),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(1),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: *variadic.MustNewUint32OrHash(1 + maxResponseSize),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test 5": {
			w: &worker{
				startNumber:  uintPtr(1),
				targetNumber: uintPtr(10),
				targetHash:   common.Hash{0xa},
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(1),
					EndBlockHash:  &(common.Hash{0xa}),
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test 6": {
			w: &worker{
				startNumber:  uintPtr(1),
				startHash:    common.Hash{0xb},
				targetNumber: uintPtr(10),
				targetHash:   common.Hash{0xc},
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(common.Hash{0xb}),
					EndBlockHash:  &(common.Hash{0xc}),
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test 7": {
			w: &worker{
				startNumber:  uintPtr(10),
				targetNumber: uintPtr(10),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(10),
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test 8": {
			w: &worker{
				startNumber:  uintPtr(1 + maxResponseSize + (maxResponseSize / 2)),
				targetNumber: uintPtr(1),
				direction:    network.Descending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: *variadic.MustNewUint32OrHash(1 + (maxResponseSize / 2)),
					EndBlockHash:  nil,
					Direction:     network.Descending,
					Max:           &max64,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(1 + maxResponseSize + (maxResponseSize / 2)),
					EndBlockHash:  nil,
					Direction:     network.Descending,
					Max:           &max128,
				},
			},
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			reqs, err := workerToRequests(tc.w)
			require.NoError(t, err)
			require.Equal(t, tc.expected, reqs)
		})
	}
}

func TestChainSync_validateResponse(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		networkBuilder    func(ctrl *gomock.Controller) Network
		req               *network.BlockRequestMessage
		resp              *network.BlockResponseMessage
		expectedError     error
	}{
		"nil req, nil resp": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
				return mockBlockState
			},
			networkBuilder: func(ctrl *gomock.Controller) Network {
				return NewMockNetwork(ctrl)
			},
			expectedError: errEmptyBlockData,
		},
		"handle error response is not chain, has header": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
				return mockBlockState
			},
			networkBuilder: func(ctrl *gomock.Controller) Network {
				return NewMockNetwork(ctrl)
			},
			req: &network.BlockRequestMessage{
				RequestedData: network.RequestedDataHeader,
			},
			resp: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Header: &types.Header{
							Number: 1,
						},
						Body: &types.Body{},
					},
					{
						Header: &types.Header{
							Number: 2,
						},
						Body: &types.Body{},
					},
				},
			},
			expectedError: errResponseIsNotChain,
		},
		"handle justification-only request, unknown block": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				return mockBlockState
			},
			networkBuilder: func(ctrl *gomock.Controller) Network {
				mockNetwork := NewMockNetwork(ctrl)
				mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadJustificationValue,
					Reason: peerset.BadJustificationReason,
				}, peer.ID(""))
				return mockNetwork
			},
			req: &network.BlockRequestMessage{
				RequestedData: network.RequestedDataJustification,
			},
			resp: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Justification: &[]byte{0},
					},
				},
			},
			expectedError: errUnknownBlockForJustification,
		},
		"handle error unknown parent": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(false, nil)
				return mockBlockState
			},
			networkBuilder: func(ctrl *gomock.Controller) Network {
				return NewMockNetwork(ctrl)
			},
			req: &network.BlockRequestMessage{
				RequestedData: network.RequestedDataHeader,
			},
			resp: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Header: &types.Header{
							Number: 1,
						},
						Body: &types.Body{},
					},
					{
						Header: &types.Header{
							Number: 2,
						},
						Body: &types.Body{},
					},
				},
			},
			expectedError: errUnknownParent,
		},
		"no error": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
				mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil)
				return mockBlockState
			},
			networkBuilder: func(ctrl *gomock.Controller) Network {
				return NewMockNetwork(ctrl)
			},
			req: &network.BlockRequestMessage{
				RequestedData: network.RequestedDataHeader,
			},
			resp: &network.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Header: &types.Header{
							Number: 2,
						},
						Body: &types.Body{},
					},
					{
						Header: &types.Header{
							ParentHash: (&types.Header{
								Number: 2,
							}).Hash(),
							Number: 3,
						},
						Body: &types.Body{},
					},
				},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			cfg := &chainSyncConfig{
				bs:            tt.blockStateBuilder(ctrl),
				pendingBlocks: newDisjointBlockSet(pendingBlocksLimit),
				readyBlocks:   newBlockQueue(maxResponseSize),
				net:           tt.networkBuilder(ctrl),
			}
			cs := newChainSync(cfg)

			err := cs.validateResponse(tt.req, tt.resp, "")
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChainSync_doSync(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	readyBlocks := newBlockQueue(maxResponseSize)
	cs := newTestChainSyncWithReadyBlocks(ctrl, readyBlocks)

	max := uint32(1)
	req := &network.BlockRequestMessage{
		RequestedData: bootstrapRequestData,
		StartingBlock: *variadic.MustNewUint32OrHash(1),
		EndBlockHash:  nil,
		Direction:     network.Ascending,
		Max:           &max,
	}

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil).Times(2)
	cs.blockState = mockBlockState

	workerErr := cs.doSync(req, make(map[peer.ID]struct{}))
	require.NotNil(t, workerErr)
	require.Equal(t, errNoPeers, workerErr.err)

	cs.peerState["noot"] = &peerState{
		number: 100,
	}

	mockNetwork := NewMockNetwork(ctrl)
	startingBlock := variadic.MustNewUint32OrHash(1)
	max1 := uint32(1)
	mockNetwork.EXPECT().DoBlockRequest(peer.ID("noot"), &network.BlockRequestMessage{
		RequestedData: 19,
		StartingBlock: *startingBlock,
		EndBlockHash:  nil,
		Direction:     0,
		Max:           &max1,
	})
	cs.network = mockNetwork

	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.NotNil(t, workerErr)
	require.Equal(t, errNilResponse, workerErr.err)

	resp := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Hash: common.Hash{0x1},
				Header: &types.Header{
					Number: 1,
				},
				Body: &types.Body{},
			},
		},
	}

	mockNetwork = NewMockNetwork(ctrl)
	mockNetwork.EXPECT().DoBlockRequest(peer.ID("noot"), &network.BlockRequestMessage{
		RequestedData: 19,
		StartingBlock: *startingBlock,
		EndBlockHash:  nil,
		Direction:     0,
		Max:           &max1,
	}).Return(resp, nil)
	cs.network = mockNetwork

	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.Nil(t, workerErr)
	bd := readyBlocks.pop(context.Background())
	require.NotNil(t, bd)
	require.Equal(t, resp.BlockData[0], bd)

	parent := (&types.Header{
		Number: 2,
	}).Hash()
	resp = &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Hash: common.Hash{0x3},
				Header: &types.Header{
					ParentHash: parent,
					Number:     3,
				},
				Body: &types.Body{},
			},
			{
				Hash: common.Hash{0x2},
				Header: &types.Header{
					Number: 2,
				},
				Body: &types.Body{},
			},
		},
	}

	// test to see if descending blocks get reversed
	req.Direction = network.Descending
	mockNetwork = NewMockNetwork(ctrl)
	mockNetwork.EXPECT().DoBlockRequest(peer.ID("noot"), &network.BlockRequestMessage{
		RequestedData: 19,
		StartingBlock: *startingBlock,
		EndBlockHash:  nil,
		Direction:     1,
		Max:           &max1,
	}).Return(resp, nil)
	cs.network = mockNetwork
	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.Nil(t, workerErr)

	bd = readyBlocks.pop(context.Background())
	require.NotNil(t, bd)
	require.Equal(t, resp.BlockData[0], bd)

	bd = readyBlocks.pop(context.Background())
	require.NotNil(t, bd)
	require.Equal(t, resp.BlockData[1], bd)
}

func TestHandleReadyBlock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	readyBlocks := newBlockQueue(maxResponseSize)
	cs := newTestChainSyncWithReadyBlocks(ctrl, readyBlocks)

	// test that descendant chain gets returned by getReadyDescendants on block 1 being ready
	header1 := &types.Header{
		Number: 1,
	}
	block1 := &types.Block{
		Header: *header1,
		Body:   types.Body{},
	}

	header2 := &types.Header{
		ParentHash: header1.Hash(),
		Number:     2,
	}
	block2 := &types.Block{
		Header: *header2,
		Body:   types.Body{},
	}
	cs.pendingBlocks.addBlock(block2)

	header3 := &types.Header{
		ParentHash: header2.Hash(),
		Number:     3,
	}
	block3 := &types.Block{
		Header: *header3,
		Body:   types.Body{},
	}
	cs.pendingBlocks.addBlock(block3)

	header2NotDescendant := &types.Header{
		ParentHash: common.Hash{0xff},
		Number:     2,
	}
	block2NotDescendant := &types.Block{
		Header: *header2NotDescendant,
		Body:   types.Body{},
	}
	cs.pendingBlocks.addBlock(block2NotDescendant)

	cs.handleReadyBlock(block1.ToBlockData())

	require.False(t, cs.pendingBlocks.hasBlock(header1.Hash()))
	require.False(t, cs.pendingBlocks.hasBlock(header2.Hash()))
	require.False(t, cs.pendingBlocks.hasBlock(header3.Hash()))
	require.True(t, cs.pendingBlocks.hasBlock(header2NotDescendant.Hash()))

	require.Equal(t, block1.ToBlockData(), readyBlocks.pop(context.Background()))
	require.Equal(t, block2.ToBlockData(), readyBlocks.pop(context.Background()))
	require.Equal(t, block3.ToBlockData(), readyBlocks.pop(context.Background()))
}

func TestChainSync_determineSyncPeers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)

	req := &network.BlockRequestMessage{}
	testPeerA := peer.ID("a")
	testPeerB := peer.ID("b")
	peersTried := make(map[peer.ID]struct{})

	// test base case
	cs.peerState[testPeerA] = &peerState{
		number: 129,
	}
	cs.peerState[testPeerB] = &peerState{
		number: 257,
	}

	peers := cs.determineSyncPeers(req, peersTried)
	require.Equal(t, 2, len(peers))
	require.Contains(t, peers, testPeerA)
	require.Contains(t, peers, testPeerB)

	// test peer ignored case
	cs.ignorePeers[testPeerA] = struct{}{}
	peers = cs.determineSyncPeers(req, peersTried)
	require.Equal(t, 1, len(peers))
	require.Equal(t, []peer.ID{testPeerB}, peers)

	// test all peers ignored case
	cs.ignorePeers[testPeerB] = struct{}{}
	peers = cs.determineSyncPeers(req, peersTried)
	require.Equal(t, 2, len(peers))
	require.Contains(t, peers, testPeerA)
	require.Contains(t, peers, testPeerB)
	require.Equal(t, 0, len(cs.ignorePeers))

	// test peer's best block below number case, shouldn't include that peer
	start, err := variadic.NewUint32OrHash(130)
	require.NoError(t, err)
	req.StartingBlock = *start
	peers = cs.determineSyncPeers(req, peersTried)
	require.Equal(t, 1, len(peers))
	require.Equal(t, []peer.ID{testPeerB}, peers)

	// test peer tried case, should ignore peer already tried
	peersTried[testPeerA] = struct{}{}
	req.StartingBlock = variadic.Uint32OrHash{}
	peers = cs.determineSyncPeers(req, peersTried)
	require.Equal(t, 1, len(peers))
	require.Equal(t, []peer.ID{testPeerB}, peers)
}

func Test_chainSync_logSyncSpeed(t *testing.T) {
	t.Parallel()

	type fields struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		networkBuilder    func(ctrl *gomock.Controller, done chan struct{}) Network
		state             chainSyncState
		benchmarker       *syncBenchmarker
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "state bootstrap",
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{}, nil).AnyTimes()
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{}, nil)
					return mockBlockState
				},
				networkBuilder: func(ctrl *gomock.Controller, done chan struct{}) Network {
					mockNetwork := NewMockNetwork(ctrl)
					mockNetwork.EXPECT().Peers().DoAndReturn(func() error {
						close(done)
						return nil
					})
					return mockNetwork
				},
				benchmarker: newSyncBenchmarker(10),
				state:       bootstrap,
			},
		},
		{
			name: "case tip",
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{}, nil).AnyTimes()
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{}, nil)
					return mockBlockState
				},
				networkBuilder: func(ctrl *gomock.Controller, done chan struct{}) Network {
					mockNetwork := NewMockNetwork(ctrl)
					mockNetwork.EXPECT().Peers().DoAndReturn(func() error {
						close(done)
						return nil
					})
					return mockNetwork
				},
				benchmarker: newSyncBenchmarker(10),
				state:       tip,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			cs := &chainSync{
				ctx:           ctx,
				cancel:        cancel,
				blockState:    tt.fields.blockStateBuilder(ctrl),
				network:       tt.fields.networkBuilder(ctrl, done),
				state:         tt.fields.state,
				benchmarker:   tt.fields.benchmarker,
				logSyncPeriod: time.Millisecond,
			}
			go cs.logSyncSpeed()
			<-done
			cancel()
		})
	}
}

func Test_chainSync_start(t *testing.T) {
	t.Parallel()

	type fields struct {
		blockStateBuilder       func(ctrl *gomock.Controller) BlockState
		disjointBlockSetBuilder func(ctrl *gomock.Controller) DisjointBlockSet
		networkBuilder          func(ctrl *gomock.Controller, done chan struct{}) Network
		benchmarker             *syncBenchmarker
		slotDuration            time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "base case",
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{}, nil).AnyTimes()
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{}, nil)
					mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{}, nil).AnyTimes()
					return mockBlockState
				},
				disjointBlockSetBuilder: func(ctrl *gomock.Controller) DisjointBlockSet {
					mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
					mockDisjointBlockSet.EXPECT().run(gomock.Any())
					return mockDisjointBlockSet
				},
				networkBuilder: func(ctrl *gomock.Controller, done chan struct{}) Network {
					mockNetwork := NewMockNetwork(ctrl)
					mockNetwork.EXPECT().Peers().DoAndReturn(func() []common.PeerInfo {
						close(done)
						return nil
					})
					return mockNetwork
				},
				slotDuration: defaultSlotDuration,
				benchmarker:  newSyncBenchmarker(1),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			cs := &chainSync{
				ctx:           ctx,
				cancel:        cancel,
				blockState:    tt.fields.blockStateBuilder(ctrl),
				pendingBlocks: tt.fields.disjointBlockSetBuilder(ctrl),
				network:       tt.fields.networkBuilder(ctrl, done),
				benchmarker:   tt.fields.benchmarker,
				slotDuration:  tt.fields.slotDuration,
				logSyncPeriod: time.Second,
			}
			cs.start()
			<-done
			cs.stop()
		})
	}
}

func Test_chainSync_setBlockAnnounce(t *testing.T) {
	type args struct {
		from   peer.ID
		header *types.Header
	}
	tests := map[string]struct {
		chainSyncBuilder func(ctrl *gomock.Controller) chainSync
		args             args
		wantErr          error
	}{
		"base case": {
			args: args{
				header: &types.Header{Number: 2},
			},
			chainSyncBuilder: func(ctrl *gomock.Controller) chainSync {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().HasHeader(common.MustHexToHash(
					"0x05bdcc454f60a08d427d05e7f19f240fdc391f570ab76fcb96ecca0b5823d3bf")).Return(true, nil)
				mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
				return chainSync{
					blockState:    mockBlockState,
					pendingBlocks: mockDisjointBlockSet,
				}
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			sync := tt.chainSyncBuilder(ctrl)
			err := sync.setBlockAnnounce(tt.args.from, tt.args.header)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func Test_chainSync_getHighestBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		peerState        map[peer.ID]*peerState
		wantHighestBlock uint
		expectedError    error
	}{
		{
			name:          "error no peers",
			expectedError: errors.New("no peers to sync with"),
		},
		{
			name:             "base case",
			peerState:        map[peer.ID]*peerState{"1": {number: 2}},
			wantHighestBlock: 2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs := &chainSync{
				peerState: tt.peerState,
			}
			gotHighestBlock, err := cs.getHighestBlock()
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantHighestBlock, gotHighestBlock)
		})
	}
}

func Test_chainSync_handleResult(t *testing.T) {
	t.Parallel()
	mockError := errors.New("test mock error")
	tests := map[string]struct {
		chainSyncBuilder func(ctrl *gomock.Controller, result *worker) chainSync
		maxWorkerRetries uint16
		res              *worker
		err              error
	}{
		"res.err == nil": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				return chainSync{
					workerState: newWorkerState(),
				}
			},
			res: &worker{},
		},
		"res.err.err.Error() == context.Canceled": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				return chainSync{
					workerState: newWorkerState(),
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: context.Canceled,
				},
			},
		},
		"res.err.err.Error() == context.DeadlineExceeded": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				mockNetwork := NewMockNetwork(ctrl)
				mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{Value: -1024, Reason: "Request timeout"},
					peer.ID(""))
				mockWorkHandler := NewMockworkHandler(ctrl)
				mockWorkHandler.EXPECT().handleWorkerResult(result).Return(result, nil)
				return chainSync{
					workerState: newWorkerState(),
					network:     mockNetwork,
					handler:     mockWorkHandler,
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: context.DeadlineExceeded,
				},
			},
		},
		"res.err.err.Error() dial backoff": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				return chainSync{
					workerState: newWorkerState(),
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New("dial backoff"),
				},
			},
		},
		"res.err.err.Error() == errNoPeers": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				return chainSync{
					workerState: newWorkerState(),
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errNoPeers,
				},
			},
		},
		"res.err.err.Error() == protocol not supported": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				mockNetwork := NewMockNetwork(ctrl)
				mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{Value: -2147483648,
					Reason: "Unsupported protocol"},
					peer.ID(""))
				return chainSync{
					workerState: newWorkerState(),
					network:     mockNetwork,
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New("protocol not supported"),
				},
			},
		},
		"no error, no retries": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				mockWorkHandler := NewMockworkHandler(ctrl)
				mockWorkHandler.EXPECT().handleWorkerResult(result).Return(result, nil)
				return chainSync{
					workerState: newWorkerState(),
					handler:     mockWorkHandler,
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New(""),
				},
			},
		},
		"handle work result error, no retries": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				mockWorkHandler := NewMockworkHandler(ctrl)
				mockWorkHandler.EXPECT().handleWorkerResult(result).Return(nil, mockError)
				return chainSync{
					workerState: newWorkerState(),
					handler:     mockWorkHandler,
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New(""),
				},
			},
			err: mockError,
		},
		"handle work result nil, no retries": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				mockWorkHandler := NewMockworkHandler(ctrl)
				mockWorkHandler.EXPECT().handleWorkerResult(result).Return(nil, nil)
				return chainSync{
					workerState: newWorkerState(),
					handler:     mockWorkHandler,
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New(""),
				},
			},
		},
		"no error, maxWorkerRetries 2": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				mockWorkHandler := NewMockworkHandler(ctrl)
				mockWorkHandler.EXPECT().handleWorkerResult(result).Return(result, nil)
				mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
				mockDisjointBlockSet.EXPECT().removeBlock(common.Hash{})
				return chainSync{
					workerState:   newWorkerState(),
					handler:       mockWorkHandler,
					pendingBlocks: mockDisjointBlockSet,
				}
			},
			maxWorkerRetries: 2,
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New(""),
				},
				pendingBlock: newPendingBlock(common.Hash{}, 1, nil, nil, time.Now()),
			},
		},
		"no error": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				mockWorkHandler := NewMockworkHandler(ctrl)
				mockWorkHandler.EXPECT().handleWorkerResult(result).Return(result, nil)
				mockWorkHandler.EXPECT().hasCurrentWorker(&worker{
					ctx: context.Background(),
					err: &workerError{
						err: mockError,
					},
					retryCount: 1,
					peersTried: map[peer.ID]struct{}{
						"": {},
					},
				}, newWorkerState().workers).Return(true)
				return chainSync{
					workerState:      newWorkerState(),
					handler:          mockWorkHandler,
					maxWorkerRetries: 2,
				}
			},
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: mockError,
				},
			},
		},
	}
	for testName, tt := range tests {
		tt := tt
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sync := tt.chainSyncBuilder(ctrl, tt.res)
			err := sync.handleResult(tt.res)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func newTestChainSyncWithReadyBlocks(ctrl *gomock.Controller, readyBlocks *blockQueue) *chainSync {
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))

	cfg := &chainSyncConfig{
		bs:            mockBlockState,
		readyBlocks:   readyBlocks,
		pendingBlocks: newDisjointBlockSet(pendingBlocksLimit),
		minPeers:      1,
		maxPeers:      5,
		slotDuration:  defaultSlotDuration,
	}

	return newChainSync(cfg)
}

func newTestChainSync(ctrl *gomock.Controller) *chainSync {
	readyBlocks := newBlockQueue(maxResponseSize)
	return newTestChainSyncWithReadyBlocks(ctrl, readyBlocks)
}
