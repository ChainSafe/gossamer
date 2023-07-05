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
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
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
			name: "case_bootstrap",
			s:    bootstrap,
			want: "bootstrap",
		},
		{
			name: "case_tip",
			s:    tip,
			want: "tip",
		},
		{
			name: "case_unknown",
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

func Test_chainSync_setPeerHead(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")
	const somePeer = peer.ID("abc")
	someHash := common.Hash{1, 2, 3, 4}

	testCases := map[string]struct {
		chainSyncBuilder          func(ctrl *gomock.Controller) *chainSync
		peerID                    peer.ID
		hash                      common.Hash
		number                    uint
		errWrapped                error
		errMessage                string
		expectedPeerIDToPeerState map[peer.ID]*peerState
		expectedQueuedPeerStates  []*peerState
	}{
		"best_block_header_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().Return(nil, errTest)
				return &chainSync{
					peerState:  map[peer.ID]*peerState{},
					blockState: blockState,
				}
			},
			peerID:     somePeer,
			hash:       someHash,
			number:     1,
			errWrapped: errTest,
			errMessage: "best block header: test error",
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 1,
				},
			},
		},
		"number_smaller_than_best_block_number_get_hash_by_number_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).
					Return(common.Hash{}, errTest)
				return &chainSync{
					peerState:  map[peer.ID]*peerState{},
					blockState: blockState,
				}
			},
			peerID:     somePeer,
			hash:       someHash,
			number:     1,
			errWrapped: errTest,
			errMessage: "get block hash by number: test error",
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 1,
				},
			},
		},
		"number_smaller_than_best_block_number_and_same_hash": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).Return(someHash, nil)
				return &chainSync{
					peerState:  map[peer.ID]*peerState{},
					blockState: blockState,
				}
			},
			peerID: somePeer,
			hash:   someHash,
			number: 1,
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 1,
				},
			},
		},
		"number_smaller_than_best_block_number_get_highest_finalised_header_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).
					Return(common.Hash{2}, nil) // other hash than someHash
				blockState.EXPECT().GetHighestFinalisedHeader().Return(nil, errTest)
				return &chainSync{
					peerState:  map[peer.ID]*peerState{},
					blockState: blockState,
				}
			},
			peerID:     somePeer,
			hash:       someHash,
			number:     1,
			errWrapped: errTest,
			errMessage: "get highest finalised header: test error",
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 1,
				},
			},
		},
		"number_smaller_than_best_block_number_and_finalised_number_equal_than_number": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).
					Return(common.Hash{2}, nil) // other hash than someHash
				finalisedBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				network := NewMockNetwork(ctrl)
				network.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadBlockAnnouncementValue,
					Reason: peerset.BadBlockAnnouncementReason,
				}, somePeer)
				return &chainSync{
					peerState:  map[peer.ID]*peerState{},
					blockState: blockState,
					network:    network,
				}
			},
			peerID:     somePeer,
			hash:       someHash,
			number:     1,
			errWrapped: errPeerOnInvalidFork,
			errMessage: "peer is on an invalid fork: for peer ZiCa and block number 1",
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 1,
				},
			},
		},
		"number_smaller_than_best_block_number_and_finalised_number_bigger_than_number": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).
					Return(common.Hash{2}, nil) // other hash than someHash
				finalisedBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				network := NewMockNetwork(ctrl)
				network.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadBlockAnnouncementValue,
					Reason: peerset.BadBlockAnnouncementReason,
				}, somePeer)
				return &chainSync{
					peerState:  map[peer.ID]*peerState{},
					blockState: blockState,
					network:    network,
				}
			},
			peerID:     somePeer,
			hash:       someHash,
			number:     1,
			errWrapped: errPeerOnInvalidFork,
			errMessage: "peer is on an invalid fork: for peer ZiCa and block number 1",
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 1,
				},
			},
		},
		"number smaller than best block number and " +
			"finalised number smaller than number and " +
			"has_header_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 3}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(2)).
					Return(common.Hash{2}, nil) // other hash than someHash
				finalisedBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				blockState.EXPECT().HasHeader(someHash).Return(false, errTest)
				return &chainSync{
					peerState:  map[peer.ID]*peerState{},
					blockState: blockState,
				}
			},
			peerID:     somePeer,
			hash:       someHash,
			number:     2,
			errWrapped: errTest,
			errMessage: "has header: test error",
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 2,
				},
			},
		},
		"number smaller than best block number and " +
			"finalised number smaller than number and " +
			"has_the_hash": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 3}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(2)).
					Return(common.Hash{2}, nil) // other hash than someHash
				finalisedBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				blockState.EXPECT().HasHeader(someHash).Return(true, nil)
				return &chainSync{
					peerState:  map[peer.ID]*peerState{},
					blockState: blockState,
				}
			},
			peerID: somePeer,
			hash:   someHash,
			number: 2,
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 2,
				},
			},
		},
		"number_bigger_than_the_head_number_add_hash_and_number_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				pendingBlocks.EXPECT().addHashAndNumber(someHash, uint(2)).
					Return(errTest)
				return &chainSync{
					peerState:     map[peer.ID]*peerState{},
					blockState:    blockState,
					pendingBlocks: pendingBlocks,
				}
			},
			peerID:     somePeer,
			hash:       someHash,
			number:     2,
			errWrapped: errTest,
			errMessage: "add hash and number: test error",
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 2,
				},
			},
		},
		"number_bigger_than_the_head_number_success": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				pendingBlocks.EXPECT().addHashAndNumber(someHash, uint(2)).
					Return(nil)
				return &chainSync{
					peerState:     map[peer.ID]*peerState{},
					blockState:    blockState,
					pendingBlocks: pendingBlocks,
					// buffered of 1 so setPeerHead can write to it
					// without a consumer of the channel on the other end.
					workQueue: make(chan *peerState, 1),
				}
			},
			peerID: somePeer,
			hash:   someHash,
			number: 2,
			expectedPeerIDToPeerState: map[peer.ID]*peerState{
				somePeer: {
					who:    somePeer,
					hash:   someHash,
					number: 2,
				},
			},
			expectedQueuedPeerStates: []*peerState{
				{
					who:    somePeer,
					hash:   someHash,
					number: 2,
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			chainSync := testCase.chainSyncBuilder(ctrl)

			err := chainSync.setPeerHead(testCase.peerID, testCase.hash, testCase.number)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.expectedPeerIDToPeerState, chainSync.peerState)

			require.Equal(t, len(testCase.expectedQueuedPeerStates), len(chainSync.workQueue))
			for _, expectedPeerState := range testCase.expectedQueuedPeerStates {
				peerState := <-chainSync.workQueue
				assert.Equal(t, expectedPeerState, peerState)
			}
		})
	}
}

func TestChainSync_sync_bootstrap_withWorkerError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	mockBlockState := NewMockBlockState(ctrl)
	mockHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, 0,
		types.NewDigest())
	mockBlockState.EXPECT().BestBlockHeader().Return(mockHeader, nil).Times(2)
	cs.blockState = mockBlockState
	cs.handler = newBootstrapSyncer(mockBlockState)

	mockNetwork := NewMockNetwork(ctrl)
	startingBlock := variadic.MustNewUint32OrHash(1)
	max := uint32(128)

	mockReqRes := NewMockRequestMaker(ctrl)
	mockReqRes.EXPECT().Do(peer.ID("noot"), &network.BlockRequestMessage{
		RequestedData: 19,
		StartingBlock: *startingBlock,
		Direction:     0,
		Max:           &max,
	}, &network.BlockResponseMessage{})
	cs.blockReqRes = mockReqRes
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
			err: errEmptyBlockData, // since MockNetwork returns a nil response
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
	header := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, 1000,
		types.NewDigest())

	bs := NewMockBlockState(ctrl)
	bs.EXPECT().BestBlockHeader().Return(header, nil)
	bs.EXPECT().GetHighestFinalisedHeader().DoAndReturn(func() (*types.Header, error) {
		close(done)
		return header, nil
	})
	cs.blockState = bs

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
		"test_0": {
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
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test_1": {
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
					Direction:     network.Ascending,
					Max:           &max128,
				},
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: *variadic.MustNewUint32OrHash(1 + maxResponseSize),
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test_2": {
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
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test_3": {
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
					Direction:     network.Descending,
					Max:           &max9,
				},
			},
		},
		"test_4": {
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
					Direction:     network.Ascending,
					Max:           &max128,
				},
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: *variadic.MustNewUint32OrHash(1 + maxResponseSize),
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test_5": {
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
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test_6": {
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
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		"test_7": {
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
		"test_8": {
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
					Direction:     network.Descending,
					Max:           &max64,
				},
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(1 + maxResponseSize + (maxResponseSize / 2)),
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
	badBlockHash := common.NewHash([]byte("badblockhash"))

	tests := map[string]struct {
		blockStateBuilder func(ctrl *gomock.Controller) BlockState
		networkBuilder    func(ctrl *gomock.Controller) Network
		req               *network.BlockRequestMessage
		resp              *network.BlockResponseMessage
		expectedError     error
	}{
		"nil_req,_nil_resp": {
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
		"handle_error_response_is_not_chain,_has_header": {
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
		"handle_justification-only_request,_unknown_block": {
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
		"handle_error_unknown_parent": {
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
		"handle_error_bad_block": {
			blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
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
						Hash: badBlockHash,
						Header: &types.Header{
							Number: 2,
						},
						Body: &types.Body{},
					},
				},
			},
			expectedError: errBadBlock,
		},
		"no_error": {
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

			cfg := chainSyncConfig{
				bs:            tt.blockStateBuilder(ctrl),
				pendingBlocks: newDisjointBlockSet(pendingBlocksLimit),
				readyBlocks:   newBlockQueue(maxResponseSize),
				net:           tt.networkBuilder(ctrl),
				badBlocks: []string{
					badBlockHash.String(),
				},
			}
			mockReqRes := NewMockRequestMaker(ctrl)

			cs := newChainSync(cfg, mockReqRes)

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

	mockReqRes := NewMockRequestMaker(ctrl)
	mockReqRes.EXPECT().Do(peer.ID("noot"), &network.BlockRequestMessage{
		RequestedData: 19,
		StartingBlock: *startingBlock,
		Direction:     0,
		Max:           &max1,
	}, &network.BlockResponseMessage{})
	cs.blockReqRes = mockReqRes

	cs.network = mockNetwork

	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.NotNil(t, workerErr)
	require.Equal(t, errEmptyBlockData, workerErr.err)

	expectedResp := &network.BlockResponseMessage{
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

	mockReqRes.EXPECT().Do(peer.ID("noot"), &network.BlockRequestMessage{
		RequestedData: 19,
		StartingBlock: *startingBlock,
		Direction:     0,
		Max:           &max1,
	}, &network.BlockResponseMessage{}).Do(
		func(_ peer.ID, _ *network.BlockRequestMessage, resp *network.BlockResponseMessage) {
			*resp = *expectedResp
		},
	)

	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.Nil(t, workerErr)
	bd, err := readyBlocks.pop(context.Background())
	require.NotNil(t, bd)
	require.NoError(t, err)
	require.Equal(t, expectedResp.BlockData[0], bd)

	parent := (&types.Header{
		Number: 2,
	}).Hash()
	expectedResp = &network.BlockResponseMessage{
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

	mockReqRes.EXPECT().Do(peer.ID("noot"), &network.BlockRequestMessage{
		RequestedData: 19,
		StartingBlock: *startingBlock,
		Direction:     1,
		Max:           &max1,
	}, &network.BlockResponseMessage{}).Do(
		func(_ peer.ID, _ *network.BlockRequestMessage, resp *network.BlockResponseMessage) {
			*resp = *expectedResp
		},
	)

	cs.network = mockNetwork
	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.Nil(t, workerErr)

	bd, err = readyBlocks.pop(context.Background())
	require.NotNil(t, bd)
	require.Equal(t, expectedResp.BlockData[0], bd)
	require.NoError(t, err)

	bd, err = readyBlocks.pop(context.Background())
	require.NotNil(t, bd)
	require.Equal(t, expectedResp.BlockData[1], bd)
	require.NoError(t, err)
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

	require.False(t, cs.pendingBlocks.(*disjointBlockSet).hasBlock(header1.Hash()))
	require.False(t, cs.pendingBlocks.(*disjointBlockSet).hasBlock(header2.Hash()))
	require.False(t, cs.pendingBlocks.(*disjointBlockSet).hasBlock(header3.Hash()))
	require.True(t, cs.pendingBlocks.(*disjointBlockSet).hasBlock(header2NotDescendant.Hash()))

	blockData1, err := readyBlocks.pop(context.Background())
	require.NoError(t, err)
	require.Equal(t, block1.ToBlockData(), blockData1)

	blockData2, err := readyBlocks.pop(context.Background())
	require.NoError(t, err)
	require.Equal(t, block2.ToBlockData(), blockData2)

	blockData3, err := readyBlocks.pop(context.Background())
	require.NoError(t, err)
	require.Equal(t, block3.ToBlockData(), blockData3)
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
		networkBuilder    func(ctrl *gomock.Controller) Network
		state             chainSyncState
		benchmarker       *syncBenchmarker
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "state_bootstrap",
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{}, nil).Times(3)
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{}, nil)
					return mockBlockState
				},
				networkBuilder: func(ctrl *gomock.Controller) Network {
					mockNetwork := NewMockNetwork(ctrl)
					mockNetwork.EXPECT().Peers().Return(nil)
					return mockNetwork
				},
				benchmarker: newSyncBenchmarker(10),
				state:       bootstrap,
			},
		},
		{
			name: "case_tip",
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{}, nil).Times(3)
					mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{}, nil)
					return mockBlockState
				},
				networkBuilder: func(ctrl *gomock.Controller) Network {
					mockNetwork := NewMockNetwork(ctrl)
					mockNetwork.EXPECT().Peers().Return(nil)
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
			tickerChannel := make(chan time.Time)
			cs := &chainSync{
				ctx:            ctx,
				cancel:         cancel,
				blockState:     tt.fields.blockStateBuilder(ctrl),
				network:        tt.fields.networkBuilder(ctrl),
				state:          tt.fields.state,
				benchmarker:    tt.fields.benchmarker,
				logSyncTickerC: tickerChannel,
				logSyncTicker:  time.NewTicker(time.Hour), // just here to be stopped
				logSyncDone:    make(chan struct{}),
			}

			go cs.logSyncSpeed()

			tickerChannel <- time.Time{}
			cs.cancel()
			<-cs.logSyncDone
		})
	}
}

func Test_chainSync_start(t *testing.T) {
	t.Parallel()

	type fields struct {
		blockStateBuilder       func(ctrl *gomock.Controller) BlockState
		disjointBlockSetBuilder func(ctrl *gomock.Controller, called chan<- struct{}) DisjointBlockSet
		benchmarker             *syncBenchmarker
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "base_case",
			fields: fields{
				blockStateBuilder: func(ctrl *gomock.Controller) BlockState {
					mockBlockState := NewMockBlockState(ctrl)
					mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{}, nil)
					return mockBlockState
				},
				disjointBlockSetBuilder: func(ctrl *gomock.Controller, called chan<- struct{}) DisjointBlockSet {
					mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
					mockDisjointBlockSet.EXPECT().run(gomock.AssignableToTypeOf(make(<-chan struct{}))).
						DoAndReturn(func(stop <-chan struct{}) {
							close(called) // test glue, ideally we would use a ready chan struct passed to run().
						})
					return mockDisjointBlockSet
				},
				benchmarker: newSyncBenchmarker(1),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ctx, cancel := context.WithCancel(context.Background())
			disjointBlockSetCalled := make(chan struct{})
			cs := &chainSync{
				ctx:           ctx,
				cancel:        cancel,
				blockState:    tt.fields.blockStateBuilder(ctrl),
				pendingBlocks: tt.fields.disjointBlockSetBuilder(ctrl, disjointBlockSetCalled),
				benchmarker:   tt.fields.benchmarker,
				slotDuration:  time.Hour,
				logSyncTicker: time.NewTicker(time.Hour), // just here to be closed
				logSyncDone:   make(chan struct{}),
			}
			cs.start()
			<-disjointBlockSetCalled
			cs.stop()
		})
	}
}

func Test_chainSync_setBlockAnnounce(t *testing.T) {
	t.Parallel()

	type args struct {
		from   peer.ID
		header *types.Header
	}
	tests := map[string]struct {
		chainSyncBuilder func(*types.Header, *gomock.Controller) chainSync
		args             args
		wantErr          error
	}{
		"base_case": {
			wantErr: blocktree.ErrBlockExists,
			args: args{
				header: &types.Header{Number: 2},
			},
			chainSyncBuilder: func(_ *types.Header, ctrl *gomock.Controller) chainSync {
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
		"err_when_calling_has_header": {
			wantErr: errors.New("checking header exists"),
			args: args{
				header: &types.Header{Number: 2},
			},
			chainSyncBuilder: func(_ *types.Header, ctrl *gomock.Controller) chainSync {
				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().
					HasHeader(common.MustHexToHash(
						"0x05bdcc454f60a08d427d05e7f19f240fdc391f570ab76fcb96ecca0b5823d3bf")).
					Return(false, errors.New("checking header exists"))
				mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
				return chainSync{
					blockState:    mockBlockState,
					pendingBlocks: mockDisjointBlockSet,
				}
			},
		},
		"adding_block_header_to_pending_blocks": {
			args: args{
				header: &types.Header{Number: 2},
			},
			chainSyncBuilder: func(expectedHeader *types.Header, ctrl *gomock.Controller) chainSync {
				argumentHeaderHash := common.MustHexToHash(
					"0x05bdcc454f60a08d427d05e7f19f240fdc391f570ab76fcb96ecca0b5823d3bf")

				mockBlockState := NewMockBlockState(ctrl)
				mockBlockState.EXPECT().
					HasHeader(argumentHeaderHash).
					Return(false, nil)

				mockBlockState.EXPECT().
					BestBlockHeader().
					Return(&types.Header{Number: 1}, nil)

				mockDisjointBlockSet := NewMockDisjointBlockSet(ctrl)
				mockDisjointBlockSet.EXPECT().
					addHeader(expectedHeader).
					Return(nil)

				mockDisjointBlockSet.EXPECT().
					addHashAndNumber(argumentHeaderHash, uint(2)).
					Return(nil)

				return chainSync{
					blockState:    mockBlockState,
					pendingBlocks: mockDisjointBlockSet,
					peerState:     make(map[peer.ID]*peerState),
					// creating an buffered channel for this specific test
					// since it will put a work on the queue and an unbufered channel
					// will hang until we read on this channel and the goal is to
					// put the work on the channel and don't block
					workQueue: make(chan *peerState, 1),
				}
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			sync := tt.chainSyncBuilder(tt.args.header, ctrl)
			err := sync.setBlockAnnounce(tt.args.from, tt.args.header)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if sync.workQueue != nil {
				assert.Equal(t, len(sync.workQueue), 1)
			}
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
		"res.err_==_nil": {
			chainSyncBuilder: func(ctrl *gomock.Controller, result *worker) chainSync {
				return chainSync{
					workerState: newWorkerState(),
				}
			},
			res: &worker{},
		},
		"res.err.err.Error()_==_context.Canceled": {
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
		"res.err.err.Error()_==_context.DeadlineExceeded": {
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
		"res.err.err.Error()_dial_backoff": {
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
		"res.err.err.Error()_==_errNoPeers": {
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
		"res.err.err.Error()_==_protocol_not_supported": {
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
		"no_error,_no_retries": {
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
		"handle_work_result_error,_no_retries": {
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
		"handle_work_result_nil,_no_retries": {
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
		"no_error,_maxWorkerRetries_2": {
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
		"no_error": {
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

	cfg := chainSyncConfig{
		bs:            mockBlockState,
		readyBlocks:   readyBlocks,
		pendingBlocks: newDisjointBlockSet(pendingBlocksLimit),
		minPeers:      1,
		maxPeers:      5,
		slotDuration:  defaultSlotDuration,
	}
	mockReqRes := NewMockRequestMaker(ctrl)

	return newChainSync(cfg, mockReqRes)
}

func newTestChainSync(ctrl *gomock.Controller) *chainSync {
	readyBlocks := newBlockQueue(maxResponseSize)
	return newTestChainSyncWithReadyBlocks(ctrl, readyBlocks)
}
