// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
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

func Test_chainSync_onImportBlock(t *testing.T) {
	t.Parallel()
	const somePeer = peer.ID("abc")

	errTest := errors.New("test error")
	block2AnnounceHeader := types.NewHeader(common.Hash{}, common.Hash{},
		common.Hash{}, 2, scale.VaryingDataTypeSlice{})

	testCases := map[string]struct {
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
		//"announced_block_while_in_bootstrap_mode": {},
		//"announced_block_while_in_tip_mode":       {},
	}

	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			chainSync := tt.chainSyncBuilder(ctrl)

			announced := announcedBlock{
				who:    tt.peerID,
				header: tt.blockAnnounceHeader,
			}
			err := chainSync.onImportBlock(announced)

			assert.ErrorIs(t, err, tt.errWrapped)
			if tt.errWrapped != nil {
				assert.EqualError(t, err, tt.errMessage)
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
				workerPool := newSyncWorkerPool(networkMock)

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
		"set_peer_head_with_a_to_ignore_peer_should_be_included_in_the_workerpoll": {
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
			peerID:          peer.ID("peer-test"),
			bestHash:        randomHash,
			bestNumber:      uint(20),
			shouldBeAWorker: true,
		},
		"set_peer_head_that_stills_punished_in_the_worker_poll": {
			newChainSync: func(t *testing.T, ctrl *gomock.Controller) *chainSync {
				networkMock := NewMockNetwork(ctrl)
				workerPool := newSyncWorkerPool(networkMock)
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
				workerPool := newSyncWorkerPool(networkMock)
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
	bs BlockState, net Network, babeVerifier BabeVerifier,
	storageState StorageState, blockImportHandler BlockImportHandler, telemetry Telemetry) *chainSync {
	t.Helper()
	mockedPeerID := []peer.ID{
		peer.ID("some_peer_1"),
		peer.ID("some_peer_2"),
		peer.ID("some_peer_3"),
	}

	peerViewMap := map[peer.ID]*peerView{}
	for _, p := range mockedPeerID {
		peerViewMap[p] = &peerView{
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
		babeVerifier:       babeVerifier,
		storageState:       storageState,
		blockImportHandler: blockImportHandler,
		telemetry:          telemetry,
	}

	chainSync := newChainSync(cfg)
	chainSync.peerView = peerViewMap

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

	mockedNetwork.EXPECT().DoBlockRequest(workerPeerID, &network.BlockRequestMessage{
		RequestedData: network.BootstrapRequestData,
		StartingBlock: *startingBlock,
		Direction:     network.Ascending,
		Max:           &max,
	}).Return(totalBlockResponse, nil)
	mockedNetwork.EXPECT().AllConnectedPeersID().Return([]peer.ID{})

	mockedBlockState := NewMockBlockState(ctrl)
	mockedBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))

	mockedBlockState.EXPECT().BestBlockHeader().Return(mockedGenesisHeader, nil)

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockStorageState := NewMockStorageState(ctrl)
	mockImportHandler := NewMockBlockImportHandler(ctrl)
	mockTelemetry := NewMockTelemetry(ctrl)

	// setup mocks for new synced blocks that doesn't exists in our local database
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, totalBlockResponse.BlockData, mockedBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry)

	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block X as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by X blocks, we should execute a bootstrap
	// sync request those blocks
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockedBlockState, mockedNetwork, mockBabeVerifier,
		mockStorageState, mockImportHandler, mockTelemetry)

	target, err := cs.getTarget()
	require.NoError(t, err)
	require.Equal(t, uint(129), target)

	// include a new worker in the worker pool set, this worker
	// should be an available peer that will receive a block request
	// the worker pool executes the workers management
	cs.workerPool.fromBlockAnnounce(peer.ID("noot"))

	stopCh := make(chan struct{})
	go cs.workerPool.listenForRequests(stopCh)

	err = cs.executeBootstrapSync(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	<-cs.workerPool.doneCh

}

func TestChainSync_BootstrapSync_SuccessfulSync_WithTwoWorkers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())
	mockBlockState.EXPECT().BestBlockHeader().Return(mockedGenesisHeader, nil)

	mockNetwork := NewMockNetwork(ctrl)

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
	// the first peer will respond the from the block 1 to 128 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 128
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry)

	worker2Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[128:],
	}
	// the worker 2 will respond from block 129 to 256 so the ensureBlockImportFlow
	// will setup the expectations starting from block 128, from previous worker, until block 256
	parent := worker1Response.BlockData[127]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker2Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry)

	// we use gomock.Any since I cannot guarantee which peer picks which request
	// but the first call to DoBlockRequest will return the first set and the second
	// call will return the second set
	mockNetwork.EXPECT().DoBlockRequest(gomock.Any(), gomock.Any()).
		Return(worker1Response, nil)
	mockNetwork.EXPECT().DoBlockRequest(gomock.Any(), gomock.Any()).
		Return(worker2Response, nil)

	mockNetwork.EXPECT().AllConnectedPeersID().Return([]peer.ID{})
	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block 129 as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by 128 blocks, we should execute a bootstrap
	// sync request those blocks
	const blocksAhead = 257
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockBabeVerifier,
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
	go cs.workerPool.listenForRequests(stopCh)

	err = cs.executeBootstrapSync(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	<-cs.workerPool.doneCh
}

func TestChainSync_BootstrapSync_SuccessfulSync_WithOneWorker_Failing(t *testing.T) {

	t.Parallel()

	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(make(chan *types.FinalisationInfo))
	mockedGenesisHeader := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 0, types.NewDigest())
	mockBlockState.EXPECT().BestBlockHeader().Return(mockedGenesisHeader, nil)

	mockNetwork := NewMockNetwork(ctrl)

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
	// the first peer will respond the from the block 1 to 128 so the ensureBlockImportFlow
	// will setup the expectations starting from the genesis header until block 128
	ensureSuccessfulBlockImportFlow(t, mockedGenesisHeader, worker1Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry)

	worker2Response := &network.BlockResponseMessage{
		BlockData: blockResponse.BlockData[128:],
	}
	// the worker 2 will respond from block 129 to 256 so the ensureBlockImportFlow
	// will setup the expectations starting from block 128, from previous worker, until block 256
	parent := worker1Response.BlockData[127]
	ensureSuccessfulBlockImportFlow(t, parent.Header, worker2Response.BlockData, mockBlockState,
		mockBabeVerifier, mockStorageState, mockImportHandler, mockTelemetry)

	// we use gomock.Any since I cannot guarantee which peer picks which request
	// but the first call to DoBlockRequest will return the first set and the second
	// call will return the second set
	doBlockRequestCount := 0
	mockNetwork.EXPECT().DoBlockRequest(gomock.Any(), gomock.Any()).
		DoAndReturn(func(peerID, _ any) (any, any) {
			// this simple logic does: ensure that the DoBlockRequest is called by
			// peer.ID(alice) and peer.ID(bob). When bob calls, this method will fail
			// then alice should pick the failed request and re-execute it which will
			// be the third call

			defer func() { doBlockRequestCount++ }()

			pID := peerID.(peer.ID) // cast to peer ID
			switch doBlockRequestCount {
			case 0, 1:
				if pID == peer.ID("alice") {
					return worker1Response, nil
				}

				if pID == peer.ID("bob") {
					return nil, errors.New("a bad error while getting a response")
				}

				require.FailNow(t, "expected calls by %s and %s, got: %s",
					peer.ID("alice"), peer.ID("bob"), pID)
			default:
				// ensure the the third call will be made by peer.ID("alice")
				require.Equalf(t, pID, peer.ID("alice"),
					"expect third call be made by %s, got: %s", peer.ID("alice"), pID)
			}

			return worker2Response, nil
		}).Times(3)

	mockNetwork.EXPECT().AllConnectedPeersID().Return([]peer.ID{})
	// setup a chain sync which holds in its peer view map
	// 3 peers, each one announce block 129 as its best block number.
	// We start this test with genesis block being our best block, so
	// we're far behind by 128 blocks, we should execute a bootstrap
	// sync request those blocks
	const blocksAhead = 257
	cs := setupChainSyncToBootstrapMode(t, blocksAhead,
		mockBlockState, mockNetwork, mockBabeVerifier,
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
	go cs.workerPool.listenForRequests(stopCh)

	err = cs.executeBootstrapSync(mockedGenesisHeader)
	require.NoError(t, err)

	close(stopCh)
	<-cs.workerPool.doneCh

	// peer should be in the ignore set
	_, ok := cs.workerPool.ignorePeers[peer.ID("bob")]
	require.True(t, ok)

	_, ok = cs.workerPool.workers[peer.ID("bob")]
	require.False(t, ok)
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
	mockImportHandler *MockBlockImportHandler, mockTelemetry *MockTelemetry) {
	t.Helper()

	mockBlockState.EXPECT().HasHeader(parentHeader.Hash()).Return(true, nil)

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

		mockImportHandler.EXPECT().HandleBlockImport(expectedBlock, emptyTrieState, false).
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
