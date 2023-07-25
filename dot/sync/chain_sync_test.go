// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func Test_chainSync_onImportBlock(t *testing.T) {
	t.Parallel()
	const somePeer = peer.ID("abc")

	errTest := errors.New("test error")
	emptyTrieState := storage.NewTrieState(nil)
	block1AnnounceHeader := types.NewHeader(common.Hash{}, emptyTrieState.MustRoot(),
		common.Hash{}, 1, scale.VaryingDataTypeSlice{})
	block2AnnounceHeader := types.NewHeader(block1AnnounceHeader.Hash(), emptyTrieState.MustRoot(),
		common.Hash{}, 2, scale.VaryingDataTypeSlice{})

	testCases := map[string]struct {
		listenForRequests   bool
		chainSyncBuilder    func(ctrl *gomock.Controller) *chainSync
		peerID              peer.ID
		blockAnnounceHeader *types.Header
		errWrapped          error
		errMessage          string
	}{
		"announced_block_already_exists_in_disjoint_set": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				pendingBlocks.EXPECT().hasBlock(block2AnnounceHeader.Hash()).Return(true)
				return &chainSync{
					pendingBlocks: pendingBlocks,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
			errWrapped:          errAlreadyInDisjointSet,
			errMessage: fmt.Sprintf("already in disjoint set: block %s (#%d)",
				block2AnnounceHeader.Hash(), block2AnnounceHeader.Number),
		},
		"failed_to_add_announced_block_in_disjoint_set": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				pendingBlocks.EXPECT().hasBlock(block2AnnounceHeader.Hash()).Return(false)
				pendingBlocks.EXPECT().addHeader(block2AnnounceHeader).Return(errTest)

				return &chainSync{
					pendingBlocks: pendingBlocks,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
			errWrapped:          errTest,
			errMessage:          "while adding pending block header: test error",
		},
		"announced_block_while_in_bootstrap_mode": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				pendingBlocks := NewMockDisjointBlockSet(ctrl)
				pendingBlocks.EXPECT().hasBlock(block2AnnounceHeader.Hash()).Return(false)
				pendingBlocks.EXPECT().addHeader(block2AnnounceHeader).Return(nil)

				state := atomic.Value{}
				state.Store(bootstrap)

				return &chainSync{
					pendingBlocks: pendingBlocks,
					syncMode:      state,
				}
			},
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
		},
		"announced_block_while_in_tip_mode": {
			chainSyncBuilder: func(ctrl *gomock.Controller) *chainSync {
				pendingBlocksMock := NewMockDisjointBlockSet(ctrl)
				pendingBlocksMock.EXPECT().hasBlock(block2AnnounceHeader.Hash()).Return(false)
				pendingBlocksMock.EXPECT().addHeader(block2AnnounceHeader).Return(nil)
				pendingBlocksMock.EXPECT().removeBlock(block2AnnounceHeader.Hash())
				pendingBlocksMock.EXPECT().size().Return(int(0))

				blockStateMock := NewMockBlockState(ctrl)
				blockStateMock.EXPECT().
					HasHeader(block2AnnounceHeader.Hash()).
					Return(false, nil)

				blockStateMock.EXPECT().
					BestBlockHeader().
					Return(block1AnnounceHeader, nil)

				blockStateMock.EXPECT().
					GetHighestFinalisedHeader().
					Return(block2AnnounceHeader, nil)

				expectedRequest := network.NewBlockRequest(*variadic.MustNewUint32OrHash(block2AnnounceHeader.Hash()),
					1, network.BootstrapRequestData, network.Descending)

				fakeBlockBody := types.Body([]types.Extrinsic{})
				mockedBlockResponse := &network.BlockResponseMessage{
					BlockData: []*types.BlockData{
						{
							Hash:   block2AnnounceHeader.Hash(),
							Header: block2AnnounceHeader,
							Body:   &fakeBlockBody,
						},
					},
				}

				networkMock := NewMockNetwork(ctrl)
				requestMaker := NewMockRequestMaker(ctrl)
				requestMaker.EXPECT().
					Do(somePeer, expectedRequest, &network.BlockResponseMessage{}).
					DoAndReturn(func(_, _, response any) any {
						responsePtr := response.(*network.BlockResponseMessage)
						*responsePtr = *mockedBlockResponse
						return nil
					})

				babeVerifierMock := NewMockBabeVerifier(ctrl)
				storageStateMock := NewMockStorageState(ctrl)
				importHandlerMock := NewMockBlockImportHandler(ctrl)
				telemetryMock := NewMockTelemetry(ctrl)

				const announceBlock = true
				ensureSuccessfulBlockImportFlow(t, block1AnnounceHeader, mockedBlockResponse.BlockData,
					blockStateMock, babeVerifierMock, storageStateMock, importHandlerMock, telemetryMock,
					announceBlock)

				workerPool := newSyncWorkerPool(networkMock, requestMaker)
				// include the peer who announced the block in the pool
				workerPool.newPeer(somePeer)

				state := atomic.Value{}
				state.Store(tip)

				return &chainSync{
					pendingBlocks:      pendingBlocksMock,
					syncMode:           state,
					workerPool:         workerPool,
					network:            networkMock,
					blockState:         blockStateMock,
					babeVerifier:       babeVerifierMock,
					telemetry:          telemetryMock,
					storageState:       storageStateMock,
					blockImportHandler: importHandlerMock,
				}
			},
			listenForRequests:   true,
			peerID:              somePeer,
			blockAnnounceHeader: block2AnnounceHeader,
		},
	}

	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			chainSync := tt.chainSyncBuilder(ctrl)
			stopCh := make(chan struct{})
			wg := sync.WaitGroup{}
			if tt.listenForRequests {
				wg.Add(1)
				go chainSync.workerPool.listenForRequests(stopCh, &wg)
			}

			err := chainSync.onBlockAnnounce(announcedBlock{
				who:    tt.peerID,
				header: tt.blockAnnounceHeader,
			})

			assert.ErrorIs(t, err, tt.errWrapped)
			if tt.errWrapped != nil {
				assert.EqualError(t, err, tt.errMessage)
			}

			if tt.listenForRequests {
				close(stopCh)
				wg.Wait()
			}
		})
	}
}

func TestChainSync_setPeerHead(t *testing.T) {
	const randomHashString = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
	randomHash := common.MustHexToHash(randomHashString)

	testcases := map[string]struct {
		newChainSync    func(t *testing.T, ctrl *gomock.Controller) *chainSync
		peerID          peer.ID
		bestHash        common.Hash
		bestNumber      uint
		shouldBeAWorker bool
		workerStatus    byte
	}{
		"set_peer_head_with_new_peer": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock, NewMockRequestMaker(nil))

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:          peer.ID("peer-test"),
			bestHash:        randomHash,
			bestNumber:      uint(20),
			shouldBeAWorker: true,
			workerStatus:    available,
		},
		"set_peer_head_with_a_to_ignore_peer_should_not_be_included_in_the_workerpoll": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock, NewMockRequestMaker(nil))
				workerPool.ignorePeers = map[peer.ID]struct{}{
					peer.ID("peer-test"): {},
				}

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:          peer.ID("peer-test"),
			bestHash:        randomHash,
			bestNumber:      uint(20),
			shouldBeAWorker: false,
		},
		"set_peer_head_that_stills_punished_in_the_worker_poll": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock, NewMockRequestMaker(nil))
				workerPool.workers = map[peer.ID]*peerSyncWorker{
					peer.ID("peer-test"): {
						status:         punished,
						punishmentTime: time.Now().Add(3 * time.Hour),
					},
				}

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:          peer.ID("peer-test"),
			bestHash:        randomHash,
			bestNumber:      uint(20),
			shouldBeAWorker: true,
			workerStatus:    punished,
		},
		"set_peer_head_that_punishment_isnot_valid_in_the_worker_poll": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock, NewMockRequestMaker(nil))
				workerPool.workers = map[peer.ID]*peerSyncWorker{
					peer.ID("peer-test"): {
						status:         punished,
						punishmentTime: time.Now().Add(-3 * time.Hour),
					},
				}

				cs := newChainSyncTest(t, ctrl)
				cs.workerPool = workerPool
				return cs
			},
			peerID:          peer.ID("peer-test"),
			bestHash:        randomHash,
			bestNumber:      uint(20),
			shouldBeAWorker: true,
			workerStatus:    available,
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

			if tt.shouldBeAWorker {
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

func setupChainSyncToBootstrapMode(t *testing.T, blocksAhead uint,
	bs BlockState, net Network, reqMaker network.RequestMaker, babeVerifier BabeVerifier,
	storageState StorageState, blockImportHandler BlockImportHandler, telemetry Telemetry) *chainSync {
	t.Helper()
	mockedPeerID := []peer.ID{
		peer.ID("some_peer_1"),
		peer.ID("some_peer_2"),
		peer.ID("some_peer_3"),
	}

	peerViewMap := map[peer.ID]peerView{}
	for _, p := range mockedPeerID {
		peerViewMap[p] = peerView{
			who:    p,
			hash:   common.Hash{1, 2, 3},
			number: blocksAhead,
		}
	}

	cfg := chainSyncConfig{
		pendingBlocks:      newDisjointBlockSet(pendingBlocksLimit),
		minPeers:           1,
		maxPeers:           5,
		slotDuration:       6 * time.Second,
		bs:                 bs,
		net:                net,
		requestMaker:       reqMaker,
		babeVerifier:       babeVerifier,
		storageState:       storageState,
		blockImportHandler: blockImportHandler,
		telemetry:          telemetry,
	}

	chainSync := newChainSync(cfg)
	chainSync.peerView = peerViewMap
	chainSync.syncMode.Store(bootstrap)

	return chainSync
}

func TestChainSync_BootstrapSync_SuccessfulSync_WithOneWorker(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())

	const blocksAhead = 129
	totalBlockResponse := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 1, int(blocksAhead)-1)
	mockedNetwork := NewMockNetwork(ctrl)

	workerPeerID := peer.ID("noot")
	startingBlock := variadic.MustNewUint32OrHash(1)
	max := uint32(128)

	mockedRequestMaker := NewMockRequestMaker(ctrl)

	expectedBlockRequestMessage := &network.BlockRequestMessage{
		RequestedData: network.BootstrapRequestData,
		StartingBlock: *startingBlock,
		Direction:     network.Ascending,
		Max:           &max,
	}

	mockedRequestMaker.EXPECT().
		Do(workerPeerID, expectedBlockRequestMessage, &network.BlockResponseMessage{}).
		DoAndReturn(func(_, _, response any) any {
			responsePtr := response.(*network.BlockResponseMessage)
			*responsePtr = *totalBlockResponse
			return nil
		})

	mockedBlockState := NewMockBlockState(ctrl)
	mockedBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)
	const announceBlock = false
	// setup mocks for new synced blocks that doesn't exists in our local database
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, totalBlockResponse.BlockData, mockedBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block X as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by X blocks, we should execute a bootstrap
	// sync request those blocks
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockedBlockState, mockedNetwork, mockedRequestMaker, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(129), target)

	// include a new worker in the worker pool set, this worker
	// should be an available peer that will receive a block request
	// the worker pool executes the workers management
	cs.workerPool.fromBlockAnnounce(peer.ID("noot"))

	stopCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go cs.workerPool.listenForRequests(stopCh, &wg)

	err = cs.requestMaxBlocksFrom(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	wg.Wait()
}

func TestChainSync_BootstrapSync_SuccessfulSync_WithTwoWorkers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())

	mockNetwork := NewMockNetwork(ctrl)
	mockRequestMaker := NewMockRequestMaker(ctrl)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)

	// this test expects two workers responding each request with 128 blocks which means
	// we should import 256 blocks in total
	blockResponse := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 1, 256)

	// here we split the whole set in two parts each one will be the "response" for each peer
	worker1Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[:128],
	}
	const announceBlock = false
	// the first peer will respond the from the block 1 to 128 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 128
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	worker2Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[128:],
	}
	// the worker 2 will respond from block 129 to 256 so the ensureBlockImportFlow
	// will setup the expectations starting from block 128, from previous worker, until block 256
	parent := worker1Response.BlockData[127]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker2Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	// we use gomock.Any since I cannot guarantee which peer picks which request
	// but the first call to DoBlockRequest will return the first set and the second
	// call will return the second set
	mockRequestMaker.EXPECT().
		Do(gomock.Any(), gomock.Any(), &network.BlockResponseMessage{}).
		DoAndReturn(func(_, _, response any) any {
			responsePtr := response.(*network.BlockResponseMessage)
			*responsePtr = *worker1Response
			return nil
		})

	mockRequestMaker.EXPECT().
		Do(gomock.Any(), gomock.Any(), &network.BlockResponseMessage{}).
		DoAndReturn(func(_, _, response any) any {
			responsePtr := response.(*network.BlockResponseMessage)
			*responsePtr = *worker2Response
			return nil
		})

	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block 129 as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by 128 blocks, we should execute a bootstrap
	// sync request those blocks
	const blocksAhead = 257
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockRequestMaker, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(blocksAhead), target)

	// include a new worker in the worker pool set, this worker
	// should be an available peer that will receive a block request
	// the worker pool executes the workers management
	cs.workerPool.fromBlockAnnounce(peer.ID("noot"))
	cs.workerPool.fromBlockAnnounce(peer.ID("noot2"))

	stopCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go cs.workerPool.listenForRequests(stopCh, &wg)

	err = cs.requestMaxBlocksFrom(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	wg.Wait()
}

func TestChainSync_BootstrapSync_SuccessfulSync_WithOneWorkerFailing(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())

	mockNetwork := NewMockNetwork(ctrl)
	mockRequestMaker := NewMockRequestMaker(ctrl)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)

	// this test expects two workers responding each request with 128 blocks which means
	// we should import 256 blocks in total
	blockResponse := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 1, 256)
	const announceBlock = false

	// here we split the whole set in two parts each one will be the "response" for each peer
	worker1Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[:128],
	}

	// the first peer will respond the from the block 1 to 128 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 128
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	worker2Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[128:],
	}
	// the worker 2 will respond from block 129 to 256 so the ensureBlockImportFlow
	// will setup the expectations starting from block 128, from previous worker, until block 256
	parent := worker1Response.BlockData[127]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker2Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	// we use gomock.Any since I cannot guarantee which peer picks which request
	// but the first call to DoBlockRequest will return the first set and the second
	// call will return the second set
	doBlockRequestCount := atomic.Int32{}
	mockRequestMaker.EXPECT().
		Do(gomock.Any(), gomock.Any(), &network.BlockResponseMessage{}).
		DoAndReturn(func(peerID, _, response any) any {
			// lets ensure that the DoBlockRequest is called by
			// peer.ID(alice) and peer.ID(bob). When bob calls, this method will fail
			// then alice should pick the failed request and re-execute it which will
			// be the third call
			responsePtr := response.(*network.BlockResponseMessage)
			defer func() { doBlockRequestCount.Add(1) }()

			pID := peerID.(peer.ID) // cast to peer ID
			switch doBlockRequestCount.Load() {
			case 0, 1:
				if pID == peer.ID("alice") {
					*responsePtr = *worker1Response
					return nil
				}

				if pID == peer.ID("bob") {
					return errors.New("a bad error while getting a response")
				}

				require.FailNow(t, "expected calls by %s and %s, got: %s",
					peer.ID("alice"), peer.ID("bob"), pID)
			default:
				// ensure the the third call will be made by peer.ID("alice")
				require.Equalf(t, pID, peer.ID("alice"),
					"expect third call be made by %s, got: %s", peer.ID("alice"), pID)
			}

			*responsePtr = *worker2Response
			return nil
		}).Times(3)

	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block 129 as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by 128 blocks, we should execute a bootstrap
	// sync request those blocks
	const blocksAhead = 257
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockRequestMaker, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(blocksAhead), target)

	// include a new worker in the worker pool set, this worker
	// should be an available peer that will receive a block request
	// the worker pool executes the workers management
	cs.workerPool.fromBlockAnnounce(peer.ID("alice"))
	cs.workerPool.fromBlockAnnounce(peer.ID("bob"))

	stopCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go cs.workerPool.listenForRequests(stopCh, &wg)

	err = cs.requestMaxBlocksFrom(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	wg.Wait()

	// peer should be punished
	syncWorker, ok := cs.workerPool.workers[peer.ID("bob")]
	require.True(t, ok)
	require.Equal(t, punished, syncWorker.status)
}

func TestChainSync_BootstrapSync_SuccessfulSync_WithProtocolNotSupported(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())

	mockNetwork := NewMockNetwork(ctrl)
	mockRequestMaker := NewMockRequestMaker(ctrl)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)

	// this test expects two workers responding each request with 128 blocks which means
	// we should import 256 blocks in total
	blockResponse := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 1, 256)
	const announceBlock = false

	// here we split the whole set in two parts each one will be the "response" for each peer
	worker1Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[:128],
	}

	// the first peer will respond the from the block 1 to 128 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 128
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	worker2Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[128:],
	}
	// the worker 2 will respond from block 129 to 256 so the ensureBlockImportFlow
	// will setup the expectations starting from block 128, from previous worker, until block 256
	parent := worker1Response.BlockData[127]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker2Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	// we use gomock.Any since I cannot guarantee which peer picks which request
	// but the first call to DoBlockRequest will return the first set and the second
	// call will return the second set
	doBlockRequestCount := atomic.Int32{}
	mockRequestMaker.EXPECT().
		Do(gomock.Any(), gomock.Any(), &network.BlockResponseMessage{}).
		DoAndReturn(func(peerID, _, response any) any {
			// lets ensure that the DoBlockRequest is called by
			// peer.ID(alice) and peer.ID(bob). When bob calls, this method will fail
			// then alice should pick the failed request and re-execute it which will
			// be the third call
			responsePtr := response.(*network.BlockResponseMessage)
			defer func() { doBlockRequestCount.Add(1) }()

			pID := peerID.(peer.ID) // cast to peer ID
			switch doBlockRequestCount.Load() {
			case 0, 1:
				if pID == peer.ID("alice") {
					*responsePtr = *worker1Response
					return nil
				}

				if pID == peer.ID("bob") {
					return errors.New("protocols not supported")
				}

				require.FailNow(t, "expected calls by %s and %s, got: %s",
					peer.ID("alice"), peer.ID("bob"), pID)
			default:
				// ensure the the third call will be made by peer.ID("alice")
				require.Equalf(t, pID, peer.ID("alice"),
					"expect third call be made by %s, got: %s", peer.ID("alice"), pID)
			}

			*responsePtr = *worker2Response
			return nil
		}).Times(3)

	// since peer.ID("bob") will fail with protocols not supported his
	// reputation will be affected and
	mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  peerset.BadProtocolValue,
		Reason: peerset.BadProtocolReason,
	}, peer.ID("bob"))
	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block 129 as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by 128 blocks, we should execute a bootstrap
	// sync request those blocks
	const blocksAhead = 257
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockRequestMaker, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(blocksAhead), target)

	// include a new worker in the worker pool set, this worker
	// should be an available peer that will receive a block request
	// the worker pool executes the workers management
	cs.workerPool.fromBlockAnnounce(peer.ID("alice"))
	cs.workerPool.fromBlockAnnounce(peer.ID("bob"))

	stopCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go cs.workerPool.listenForRequests(stopCh, &wg)

	err = cs.requestMaxBlocksFrom(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	wg.Wait()

	// peer should be punished
	syncWorker, ok := cs.workerPool.workers[peer.ID("bob")]
	require.True(t, ok)
	require.Equal(t, punished, syncWorker.status)
}

func TestChainSync_BootstrapSync_SuccessfulSync_WithNilHeaderInResponse(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())

	mockNetwork := NewMockNetwork(ctrl)
	mockRequestMaker := NewMockRequestMaker(ctrl)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)

	// this test expects two workers responding each request with 128 blocks which means
	// we should import 256 blocks in total
	blockResponse := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 1, 256)
	const announceBlock = false

	// here we split the whole set in two parts each one will be the "response" for each peer
	worker1Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[:128],
	}

	// the first peer will respond the from the block 1 to 128 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 128
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	worker2Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[128:],
	}
	// the worker 2 will respond from block 129 to 256 so the ensureBlockImportFlow
	// will setup the expectations starting from block 128, from previous worker, until block 256
	parent := worker1Response.BlockData[127]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker2Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	// we use gomock.Any since I cannot guarantee which peer picks which request
	// but the first call to DoBlockRequest will return the first set and the second
	// call will return the second set
	doBlockRequestCount := atomic.Int32{}
	mockRequestMaker.EXPECT().
		Do(gomock.Any(), gomock.Any(), &network.BlockResponseMessage{}).
		DoAndReturn(func(peerID, _, response any) any {
			// lets ensure that the DoBlockRequest is called by
			// peer.ID(alice) and peer.ID(bob). When bob calls, this method return an
			// response item but without header as was requested
			responsePtr := response.(*network.BlockResponseMessage)
			defer func() { doBlockRequestCount.Add(1) }()

			pID := peerID.(peer.ID) // cast to peer ID
			switch doBlockRequestCount.Load() {
			case 0, 1:
				if pID == peer.ID("alice") {
					*responsePtr = *worker1Response
					return nil
				}

				if pID == peer.ID("bob") {
					incompleteBlockData := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 128, 256)
					incompleteBlockData.BlockData[0].Header = nil

					*responsePtr = *incompleteBlockData
					return nil
				}

				require.FailNow(t, "expected calls by %s and %s, got: %s",
					peer.ID("alice"), peer.ID("bob"), pID)
			default:
				// ensure the the third call will be made by peer.ID("alice")
				require.Equalf(t, pID, peer.ID("alice"),
					"expect third call be made by %s, got: %s", peer.ID("alice"), pID)
			}

			*responsePtr = *worker2Response
			return nil
		}).Times(3)

	// since peer.ID("bob") will fail with protocols not supported his
	// reputation will be affected and
	mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  peerset.IncompleteHeaderValue,
		Reason: peerset.IncompleteHeaderReason,
	}, peer.ID("bob"))
	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block 129 as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by 128 blocks, we should execute a bootstrap
	// sync request those blocks
	const blocksAhead = 257
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockRequestMaker, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(blocksAhead), target)

	// include a new worker in the worker pool set, this worker
	// should be an available peer that will receive a block request
	// the worker pool executes the workers management
	cs.workerPool.fromBlockAnnounce(peer.ID("alice"))
	cs.workerPool.fromBlockAnnounce(peer.ID("bob"))

	stopCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go cs.workerPool.listenForRequests(stopCh, &wg)

	err = cs.requestMaxBlocksFrom(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	wg.Wait()

	// peer should be punished
	syncWorker, ok := cs.workerPool.workers[peer.ID("bob")]
	require.True(t, ok)
	require.Equal(t, punished, syncWorker.status)
}

func TestChainSync_BootstrapSync_SuccessfulSync_WithResponseIsNotAChain(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())

	mockNetwork := NewMockNetwork(ctrl)
	mockRequestMaker := NewMockRequestMaker(ctrl)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)

	// this test expects two workers responding each request with 128 blocks which means
	// we should import 256 blocks in total
	blockResponse := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 1, 256)
	const announceBlock = false

	// here we split the whole set in two parts each one will be the "response" for each peer
	worker1Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[:128],
	}

	// the first peer will respond the from the block 1 to 128 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 128
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	worker2Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[128:],
	}
	// the worker 2 will respond from block 129 to 256 so the ensureBlockImportFlow
	// will setup the expectations starting from block 128, from previous worker, until block 256
	parent := worker1Response.BlockData[127]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker2Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	// we use gomock.Any since I cannot guarantee which peer picks which request
	// but the first call to DoBlockRequest will return the first set and the second
	// call will return the second set
	doBlockRequestCount := atomic.Int32{}
	mockRequestMaker.EXPECT().
		Do(gomock.Any(), gomock.Any(), &network.BlockResponseMessage{}).
		DoAndReturn(func(peerID, _, response any) any {
			// lets ensure that the DoBlockRequest is called by
			// peer.ID(alice) and peer.ID(bob). When bob calls, this method return an
			// response that does not form an chain
			responsePtr := response.(*network.BlockResponseMessage)
			defer func() { doBlockRequestCount.Add(1) }()

			pID := peerID.(peer.ID) // cast to peer ID
			switch doBlockRequestCount.Load() {
			case 0, 1:
				if pID == peer.ID("alice") {
					*responsePtr = *worker1Response
					return nil
				}

				if pID == peer.ID("bob") {
					notAChainBlockData := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 128, 256)
					// swap positions to force the problem
					notAChainBlockData.BlockData[0], notAChainBlockData.BlockData[130] =
						notAChainBlockData.BlockData[130], notAChainBlockData.BlockData[0]

					*responsePtr = *notAChainBlockData
					return nil
				}

				require.FailNow(t, "expected calls by %s and %s, got: %s",
					peer.ID("alice"), peer.ID("bob"), pID)
			default:
				// ensure the the third call will be made by peer.ID("alice")
				require.Equalf(t, pID, peer.ID("alice"),
					"expect third call be made by %s, got: %s", peer.ID("alice"), pID)
			}

			*responsePtr = *worker2Response
			return nil
		}).Times(3)

	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block 129 as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by 128 blocks, we should execute a bootstrap
	// sync request those blocks
	const blocksAhead = 257
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockRequestMaker, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(blocksAhead), target)

	// include a new worker in the worker pool set, this worker
	// should be an available peer that will receive a block request
	// the worker pool executes the workers management
	cs.workerPool.fromBlockAnnounce(peer.ID("alice"))
	cs.workerPool.fromBlockAnnounce(peer.ID("bob"))

	stopCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go cs.workerPool.listenForRequests(stopCh, &wg)

	err = cs.requestMaxBlocksFrom(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	wg.Wait()

	// peer should be punished
	syncWorker, ok := cs.workerPool.workers[peer.ID("bob")]
	require.True(t, ok)
	require.Equal(t, punished, syncWorker.status)
}

func TestChainSync_BootstrapSync_SuccessfulSync_WithReceivedBadBlock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())

	mockNetwork := NewMockNetwork(ctrl)
	mockRequestMaker := NewMockRequestMaker(ctrl)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)

	// this test expects two workers responding each request with 128 blocks which means
	// we should import 256 blocks in total
	blockResponse := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 1, 256)
	const announceBlock = false

	// here we split the whole set in two parts each one will be the "response" for each peer
	worker1Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[:128],
	}

	// the first peer will respond the from the block 1 to 128 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 128
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	worker2Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[128:],
	}
	// the worker 2 will respond from block 129 to 256 so the ensureBlockImportFlow
	// will setup the expectations starting from block 128, from previous worker, until block 256
	parent := worker1Response.BlockData[127]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker2Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	fakeBadBlockHash := common.MustHexToHash("0x18767cb4bb4cc13bf119f6613aec5487d4c06a2e453de53d34aea6f3f1ee9855")

	// we use gomock.Any since I cannot guarantee which peer picks which request
	// but the first call to DoBlockRequest will return the first set and the second
	// call will return the second set
	doBlockRequestCount := atomic.Int32{}
	mockRequestMaker.EXPECT().
		Do(gomock.Any(), gomock.Any(), &network.BlockResponseMessage{}).
		DoAndReturn(func(peerID, _, response any) any {
			// lets ensure that the DoBlockRequest is called by
			// peer.ID(alice) and peer.ID(bob). When bob calls, this method return an
			// response that contains a know bad block
			responsePtr := response.(*network.BlockResponseMessage)
			defer func() { doBlockRequestCount.Add(1) }()

			pID := peerID.(peer.ID) // cast to peer ID
			switch doBlockRequestCount.Load() {
			case 0, 1:
				if pID == peer.ID("alice") {
					*responsePtr = *worker1Response
					return nil
				}

				if pID == peer.ID("bob") {
					blockDataWithBadBlock := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 129, 256)
					blockDataWithBadBlock.BlockData[4].Hash = fakeBadBlockHash
					*responsePtr = *blockDataWithBadBlock
					return nil
				}

				require.FailNow(t, "expected calls by %s and %s, got: %s",
					peer.ID("alice"), peer.ID("bob"), pID)
			default:
				// ensure the the third call will be made by peer.ID("alice")
				require.Equalf(t, pID, peer.ID("alice"),
					"expect third call be made by %s, got: %s", peer.ID("alice"), pID)
			}

			*responsePtr = *worker2Response
			return nil
		}).Times(3)

	mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  peerset.BadBlockAnnouncementValue,
		Reason: peerset.BadBlockAnnouncementReason,
	}, peer.ID("bob"))
	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block 129 as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by 128 blocks, we should execute a bootstrap
	// sync request those blocks
	const blocksAhead = 257
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockRequestMaker, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	cs.badBlocks = []string{fakeBadBlockHash.String()}

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(blocksAhead), target)

	// include a new worker in the worker pool set, this worker
	// should be an available peer that will receive a block request
	// the worker pool executes the workers management
	cs.workerPool.fromBlockAnnounce(peer.ID("alice"))
	cs.workerPool.fromBlockAnnounce(peer.ID("bob"))

	stopCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go cs.workerPool.listenForRequests(stopCh, &wg)

	err = cs.requestMaxBlocksFrom(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	wg.Wait()

	// peer should be not in the worker pool
	// peer should be in the ignore list
	_, ok := cs.workerPool.workers[peer.ID("bob")]
	require.False(t, ok)

	_, ok = cs.workerPool.ignorePeers[peer.ID("bob")]
	require.True(t, ok)
}

func TestChainSync_BootstrapSync_SucessfulSync_ReceivedPartialBlockData(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())

	mockNetwork := NewMockNetwork(ctrl)
	mockRequestMaker := NewMockRequestMaker(ctrl)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)

	// create a set of 128 blocks
	blockResponse := createSuccesfullBlockResponse(t, mockedGenesisHeader.Hash(), 1, 128)
	const announceBlock = false

	// the worker will return a partial size of the set
	worker1Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[:97],
	}

	// the first peer will respond the from the block 1 to 96 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 96
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	worker1MissingBlocksResponse := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[97:],
	}

	// last item from the previous response
	parent := worker1Response.BlockData[96]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker1MissingBlocksResponse.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry, announceBlock)

	doBlockRequestCount := 0
	mockRequestMaker.EXPECT().
		Do(gomock.Any(), gomock.Any(), &network.BlockResponseMessage{}).
		DoAndReturn(func(peerID, _, response any) any {
			// lets ensure that the DoBlockRequest is called by
			// peer.ID(alice). The first call will return only 97 blocks
			// the handler should issue another call to retrieve the missing blocks
			pID := peerID.(peer.ID) // cast to peer ID
			require.Equalf(t, pID, peer.ID("alice"),
				"expect third call be made by %s, got: %s", peer.ID("alice"), pID)

			responsePtr := response.(*network.BlockResponseMessage)
			defer func() { doBlockRequestCount++ }()

			if doBlockRequestCount == 0 {
				*responsePtr = *worker1Response
				return nil
			}

			*responsePtr = *worker1MissingBlocksResponse
			return nil
		}).Times(2)

	const blocksAhead = 256
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockRequestMaker, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(blocksAhead), target)

	cs.workerPool.fromBlockAnnounce(peer.ID("alice"))

	stopCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go cs.workerPool.listenForRequests(stopCh, &wg)

	err = cs.requestMaxBlocksFrom(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	wg.Wait()

	require.Len(t, cs.workerPool.workers, 1)

	_, ok := cs.workerPool.workers[peer.ID("alice")]
	require.True(t, ok)
}

func createSuccesfullBlockResponse(_ *testing.T, genesisHash common.Hash,
	startingAt, numBlocks int) *network.BlockResponseMessage {
	response := new(network.BlockResponseMessage)
	response.BlockData = make([]*types.BlockData, numBlocks)

	emptyTrieState := storage.NewTrieState(nil)
	tsRoot := emptyTrieState.MustRoot()

	firstHeader := types.NewHeader(genesisHash, tsRoot, common.Hash{},
		uint(startingAt), scale.VaryingDataTypeSlice{})
	response.BlockData[0] = &types.BlockData{
		Hash:          firstHeader.Hash(),
		Header:        firstHeader,
		Body:          types.NewBody([]types.Extrinsic{}),
		Justification: nil,
	}

	parentHash := firstHeader.Hash()
	for idx := 1; idx < numBlocks; idx++ {
		blockNumber := idx + startingAt
		header := types.NewHeader(parentHash, tsRoot, common.Hash{},
			uint(blockNumber), scale.VaryingDataTypeSlice{})
		response.BlockData[idx] = &types.BlockData{
			Hash:          header.Hash(),
			Header:        header,
			Body:          types.NewBody([]types.Extrinsic{}),
			Justification: nil,
		}
		parentHash = header.Hash()
	}

	return response
}

// ensureSuccessfulBlockImportFlow will setup the expectations for method calls
// that happens while chain sync imports a block
func ensureSuccessfulBlockImportFlow(t *testing.T, parentHeader *types.Header,
	blocksReceived []*types.BlockData, mockBlockState *MockBlockState,
	mockBabeVerifier *MockBabeVerifier, mockStorageState *MockStorageState,
	mockImportHandler *MockBlockImportHandler, mockTelemetry *MockTelemetry, announceBlock bool) {
	t.Helper()

	for idx, blockData := range blocksReceived {
		mockBlockState.EXPECT().HasHeader(blockData.Header.Hash()).Return(false, nil)
		mockBlockState.EXPECT().HasBlockBody(blockData.Header.Hash()).Return(false, nil)
		mockBabeVerifier.EXPECT().VerifyBlock(blockData.Header).Return(nil)

		var previousHeader *types.Header
		if idx == 0 {
			previousHeader = parentHeader
		} else {
			previousHeader = blocksReceived[idx-1].Header
		}

		mockBlockState.EXPECT().GetHeader(blockData.Header.ParentHash).Return(previousHeader, nil)
		mockStorageState.EXPECT().Lock()
		mockStorageState.EXPECT().Unlock()

		emptyTrieState := storage.NewTrieState(nil)
		parentStateRoot := previousHeader.StateRoot
		mockStorageState.EXPECT().TrieState(&parentStateRoot).
			Return(emptyTrieState, nil)

		ctrl := gomock.NewController(t)
		mockRuntimeInstance := NewMockInstance(ctrl)
		mockBlockState.EXPECT().GetRuntime(previousHeader.Hash()).
			Return(mockRuntimeInstance, nil)

		expectedBlock := &types.Block{
			Header: *blockData.Header,
			Body:   *blockData.Body,
		}

		mockRuntimeInstance.EXPECT().SetContextStorage(emptyTrieState)
		mockRuntimeInstance.EXPECT().ExecuteBlock(expectedBlock).
			Return(nil, nil)

		mockImportHandler.EXPECT().HandleBlockImport(expectedBlock, emptyTrieState, announceBlock).
			Return(nil)

		blockHash := blockData.Header.Hash()
		expectedTelemetryMessage := telemetry.NewBlockImport(
			&blockHash,
			blockData.Header.Number,
			"NetworkInitialSync")
		mockTelemetry.EXPECT().SendMessage(expectedTelemetryMessage)

		mockBlockState.EXPECT().CompareAndSetBlockData(blockData).Return(nil)
	}
}

func TestChainSync_validateResponseFields(t *testing.T) {
	t.Parallel()

	block1Header := &types.Header{
		ParentHash: common.MustHexToHash("0x00597cb4bb4cc13bf119f6613aec7642d4c06a2e453de53d34aea6f3f1eeb504"),
		Number:     2,
	}

	block2Header := &types.Header{
		ParentHash: block1Header.Hash(),
		Number:     3,
	}

	cases := map[string]struct {
		wantErr        error
		errString      string
		setupChainSync func(t *testing.T) *chainSync
		requestedData  byte
		blockData      *types.BlockData
	}{
		"requested_bootstrap_data_but_got_nil_header": {
			wantErr: errNilHeaderInResponse,
			errString: "expected header, received none: " +
				block2Header.Hash().String(),
			requestedData: network.BootstrapRequestData,
			blockData: &types.BlockData{
				Hash:          block2Header.Hash(),
				Header:        nil,
				Body:          &types.Body{},
				Justification: &[]byte{0},
			},
			setupChainSync: func(t *testing.T) *chainSync {
				ctrl := gomock.NewController(t)
				blockStateMock := NewMockBlockState(ctrl)
				blockStateMock.EXPECT().HasHeader(block1Header.ParentHash).Return(true, nil)

				networkMock := NewMockNetwork(ctrl)
				networkMock.EXPECT().ReportPeer(peerset.ReputationChange{
					Value:  peerset.IncompleteHeaderValue,
					Reason: peerset.IncompleteHeaderReason,
				}, peer.ID("peer"))

				return &chainSync{
					blockState: blockStateMock,
					network:    networkMock,
				}
			},
		},
		"requested_bootstrap_data_but_got_nil_body": {
			wantErr: errNilBodyInResponse,
			errString: "expected body, received none: " +
				block2Header.Hash().String(),
			requestedData: network.BootstrapRequestData,
			blockData: &types.BlockData{
				Hash:          block2Header.Hash(),
				Header:        block2Header,
				Body:          nil,
				Justification: &[]byte{0},
			},
			setupChainSync: func(t *testing.T) *chainSync {
				ctrl := gomock.NewController(t)
				blockStateMock := NewMockBlockState(ctrl)
				blockStateMock.EXPECT().HasHeader(block1Header.ParentHash).Return(true, nil)
				networkMock := NewMockNetwork(ctrl)

				return &chainSync{
					blockState: blockStateMock,
					network:    networkMock,
				}
			},
		},
		"requested_only_justification_but_got_nil": {
			wantErr: errNilJustificationInResponse,
			errString: "expected justification, received none: " +
				block2Header.Hash().String(),
			requestedData: network.RequestedDataJustification,
			blockData: &types.BlockData{
				Hash:          block2Header.Hash(),
				Header:        block2Header,
				Body:          nil,
				Justification: nil,
			},
			setupChainSync: func(t *testing.T) *chainSync {
				ctrl := gomock.NewController(t)
				blockStateMock := NewMockBlockState(ctrl)
				blockStateMock.EXPECT().HasHeader(block1Header.ParentHash).Return(true, nil)
				networkMock := NewMockNetwork(ctrl)

				return &chainSync{
					blockState: blockStateMock,
					network:    networkMock,
				}
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			err := validateResponseFields(tt.requestedData, []*types.BlockData{tt.blockData})
			require.ErrorIs(t, err, tt.wantErr)
			if tt.errString != "" {
				require.EqualError(t, err, tt.errString)
			}
		})
	}
}

func TestChainSync_isResponseAChain(t *testing.T) {
	t.Parallel()

	block1Header := &types.Header{
		ParentHash: common.MustHexToHash("0x00597cb4bb4cc13bf119f6613aec7642d4c06a2e453de53d34aea6f3f1eeb504"),
		Number:     2,
	}

	block2Header := &types.Header{
		ParentHash: block1Header.Hash(),
		Number:     3,
	}

	block4Header := &types.Header{
		ParentHash: common.MustHexToHash("0x198616547187613bf119f6613aec7642d4c06a2e453de53d34aea6f390788677"),
		Number:     4,
	}

	cases := map[string]struct {
		expected  bool
		blockData []*types.BlockData
	}{
		"not_a_chain": {
			expected: false,
			blockData: []*types.BlockData{
				{
					Hash:          block1Header.Hash(),
					Header:        block1Header,
					Body:          &types.Body{},
					Justification: &[]byte{0},
				},
				{
					Hash:          block2Header.Hash(),
					Header:        block2Header,
					Body:          &types.Body{},
					Justification: &[]byte{0},
				},
				{
					Hash:          block4Header.Hash(),
					Header:        block4Header,
					Body:          &types.Body{},
					Justification: &[]byte{0},
				},
			},
		},
		"is_a_chain": {
			expected: true,
			blockData: []*types.BlockData{
				{
					Hash:          block1Header.Hash(),
					Header:        block1Header,
					Body:          &types.Body{},
					Justification: &[]byte{0},
				},
				{
					Hash:          block2Header.Hash(),
					Header:        block2Header,
					Body:          &types.Body{},
					Justification: &[]byte{0},
				},
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			output := isResponseAChain(tt.blockData)
			require.Equal(t, tt.expected, output)
		})
	}
}

func TestChainSync_getHighestBlock(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expectedHighestBlock uint
		wantErr              error
		chainSyncPeerView    map[peer.ID]peerView
	}{
		"no_peer_view": {
			wantErr:              errNoPeers,
			expectedHighestBlock: 0,
			chainSyncPeerView:    make(map[peer.ID]peerView),
		},
		"highest_block": {
			expectedHighestBlock: 500,
			chainSyncPeerView: map[peer.ID]peerView{
				peer.ID("peer-A"): {
					number: 100,
				},
				peer.ID("peer-B"): {
					number: 500,
				},
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			chainSync := &chainSync{
				peerView: tt.chainSyncPeerView,
			}

			highestBlock, err := chainSync.getHighestBlock()
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.expectedHighestBlock, highestBlock)
		})
	}
}
