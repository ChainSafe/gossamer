package sync

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultSlotDuration = 6 * time.Second

func newTestChainSyncWithReadyBlocks(ctrl *gomock.Controller) *chainSync {
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))

	cfg := chainSyncConfig{
		bs:            mockBlockState,
		pendingBlocks: newDisjointBlockSet(pendingBlocksLimit),
		minPeers:      1,
		maxPeers:      5,
		slotDuration:  defaultSlotDuration,
	}

	return newChainSync(cfg)
}

func newTestChainSync(ctrl *gomock.Controller) *chainSync {
	return newTestChainSyncWithReadyBlocks(ctrl)
}

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

func Test_chainSync_setBlockAnnounce(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")
	const somePeer = peer.ID("abc")

	block1AnnounceHeader := types.NewHeader(common.Hash{}, common.Hash{},
		common.Hash{}, 1, scale.VaryingDataTypeSlice{})
	block2AnnounceHeader := types.NewHeader(common.Hash{}, common.Hash{},
		common.Hash{}, 2, scale.VaryingDataTypeSlice{})

	testCases := map[string]struct {
		chainSyncBuilder            func(ctrl *gomock.Controller) *chainSync
		peerID                      peer.ID
		blockAnnounceHeader         *types.Header
		errWrapped                  error
		errMessage                  string
		expectedQueuedBlockAnnounce *announcedBlock
	}{
		"best_block_header_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				blockState.EXPECT().BestBlockHeader().Return(nil, errTest)
				return &chainSync{
					peerView:   map[peer.ID]*peerView{},
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "best block header: test error",
		},
		"number_smaller_than_best_block_number_get_hash_by_number_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{}, errTest)
				return &chainSync{
					peerView:   map[peer.ID]*peerView{},
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "get block hash by number: test error",
		},
		"number_smaller_than_best_block_number_and_same_hash": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).Return(block1AnnounceHeader.Hash(), nil)
				return &chainSync{
					peerView:   map[peer.ID]*peerView{},
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
		},
		"number_smaller_than_best_block_number_get_highest_finalised_header_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 2}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(1)).Return(common.Hash{2}, nil)
				blockState.EXPECT().GetHighestFinalisedHeader().Return(nil, errTest)
				return &chainSync{
					peerView:   map[peer.ID]*peerView{},
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "get highest finalised header: test error",
		},
		"number_smaller_than_best_block_announced_number_equaks_finalised_number": {
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
					peerView:   map[peer.ID]*peerView{},
					blockState: blockState,
					network:    network,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errPeerOnInvalidFork,
			errMessage:          "peer is on an invalid fork: for peer ZiCa and block number 1",
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
					peerView:   map[peer.ID]*peerView{},
					blockState: blockState,
					network:    network,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block1AnnounceHeader,
			errWrapped:          errPeerOnInvalidFork,
			errMessage:          "peer is on an invalid fork: for peer ZiCa and block number 1",
		},
		"number_smaller_than_best_block_number_and_" +
			"finalised_number_smaller_than_number_and_" +
			"has_header_error": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 3}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(2)).
					Return(common.Hash{5, 1, 2}, nil) // other hash than block2AnnounceHeader hash
				finalisedBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				blockState.EXPECT().HasHeader(block2AnnounceHeader.Hash()).Return(false, errTest)
				return &chainSync{
					peerView:   map[peer.ID]*peerView{},
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "has header: test error",
		},
		"number_smaller_than_best_block_number_and_" +
			"finalised_number_smaller_than_number_and_" +
			"has_the_hash": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 3}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				blockState.EXPECT().GetHashByNumber(uint(2)).
					Return(common.Hash{2}, nil) // other hash than someHash
				finalisedBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().GetHighestFinalisedHeader().Return(finalisedBlockHeader, nil)
				blockState.EXPECT().HasHeader(block2AnnounceHeader.Hash()).Return(true, nil)
				return &chainSync{
					peerView:   map[peer.ID]*peerView{},
					blockState: blockState,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
		},
		"number_bigger_than_best_block_number_already_exists_in_disjoint_set": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				pendingBlocks.EXPECT().hasBlock(block2AnnounceHeader.Hash()).Return(true)
				return &chainSync{
					peerView:      map[peer.ID]*peerView{},
					blockState:    blockState,
					pendingBlocks: pendingBlocks,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
			errWrapped:          errAlreadyInDisjointSet,
			errMessage: fmt.Sprintf("already in disjoint set: block %s (#%d)",
				block2AnnounceHeader.Hash(), block2AnnounceHeader.Number),
		},
		"number_bigger_than_best_block_number_added_in_disjoint_set_with_success": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				blockState := NewMockBlockState(ctrl)
				bestBlockHeader := &types.Header{Number: 1}
				blockState.EXPECT().BestBlockHeader().Return(bestBlockHeader, nil)
				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				pendingBlocks.EXPECT().hasBlock(block2AnnounceHeader.Hash()).Return(false)
				pendingBlocks.EXPECT().addHeader(block2AnnounceHeader).Return(nil)
				return &chainSync{
					peerView:      map[peer.ID]*peerView{},
					blockState:    blockState,
					pendingBlocks: pendingBlocks,
					// buffered of 1 so setBlockAnnounce can write to it
					// without a consumer of the channel on the other end.
					blockAnnounceCh: make(chan announcedBlock, 1),
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
			expectedQueuedBlockAnnounce: &announcedBlock{
				who:    somePeer,
				header: block2AnnounceHeader,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			chainSync := testCase.chainSyncBuilder(ctrl)

			err := chainSync.setBlockAnnounce(testCase.peerID, testCase.blockAnnounceHeader)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}

			if testCase.expectedQueuedBlockAnnounce != nil {
				queuedBlockAnnounce := <-chainSync.blockAnnounceCh
				assert.Equal(t, *testCase.expectedQueuedBlockAnnounce, queuedBlockAnnounce)
			}
		})
	}
}

func TestChainSync_setPeerHead(t *testing.T) {
	const randomHashString = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
	randomHash := common.MustHexToHash(randomHashString)

	testcases := map[string]struct {
		newChainSync      func(t *testing.T, ctrl *gomock.Controller) *chainSync
		peerID            peer.ID
		bestHash          common.Hash
		bestNumber        uint
		shouldBeAndWorker bool
		workerStatus      byte
	}{
		"set_peer_head_with_new_peer": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock)

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:            peer.ID("peer-test"),
			bestHash:          randomHash,
			bestNumber:        uint(20),
			shouldBeAndWorker: true,
			workerStatus:      available,
		},
		"set_peer_head_with_a_to_ignore_peer_should_not_be_included_in_the_workerpoll": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock)
				workerPool.ignorePeers = map[peer.ID]struct{}{
					peer.ID("peer-test"): {},
				}

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:            peer.ID("peer-test"),
			bestHash:          randomHash,
			bestNumber:        uint(20),
			shouldBeAndWorker: false,
		},
		"set_peer_head_that_stills_punished_in_the_worker_poll": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock)
				workerPool.workers = map[peer.ID]*peerSyncWorker{
					peer.ID("peer-test"): {
						status:       punished,
						punishedTime: time.Now().Add(3 * time.Hour),
					},
				}

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:            peer.ID("peer-test"),
			bestHash:          randomHash,
			bestNumber:        uint(20),
			shouldBeAndWorker: true,
			workerStatus:      punished,
		},
		"set_peer_head_that_punishment_isnot_valid_in_the_worker_poll": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock)
				workerPool.workers = map[peer.ID]*peerSyncWorker{
					peer.ID("peer-test"): {
						status:       punished,
						punishedTime: time.Now().Add(-3 * time.Hour),
					},
				}

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:            peer.ID("peer-test"),
			bestHash:          randomHash,
			bestNumber:        uint(20),
			shouldBeAndWorker: true,
			workerStatus:      available,
		},
	}

	for tname, tt := range testcases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			cs := tt.newChainSync(t, ctrl)
			cs.setPeerHead(tt.peerID, tt.bestHash, tt.bestNumber)

			view, exists := cs.peerView[tt.peerID]
			require.True(t, exists)
			require.Equal(t, tt.peerID, view.who)
			require.Equal(t, tt.bestHash, view.hash)
			require.Equal(t, tt.bestNumber, view.number)

			if tt.shouldBeAndWorker {
				syncWorker, exists := cs.workerPool.workers[tt.peerID]
				require.True(t, exists)
				require.Equal(t, tt.workerStatus, syncWorker.status)
			} else {
				_, exists := cs.workerPool.workers[tt.peerID]
				require.False(t, exists)
			}
		})
	}
}

func newChainSyncTest(t *testing.T, ctrl *gomock.Controller) *chainSync {
	t.Helper()

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))

	cfg := chainSyncConfig{
		bs:            mockBlockState,
		pendingBlocks: newDisjointBlockSet(pendingBlocksLimit),
		minPeers:      1,
		maxPeers:      5,
		slotDuration:  6 * time.Second,
	}

	return newChainSync(cfg)
}
