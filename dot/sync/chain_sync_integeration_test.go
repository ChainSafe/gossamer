//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestChainSync_SetPeerHead_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)

	testPeer := peer.ID("noot")
	hash := common.Hash{0xa, 0xb}
	const number uint = 1000
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
	cs.blockState = new(mocks.BlockState)
	header, err := types.NewHeader(common.Hash{0}, trie.EmptyHash, trie.EmptyHash, number,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*mocks.BlockState).On("BestBlockHeader").Return(header, nil)
	fin, err := types.NewHeader(common.Hash{0}, trie.EmptyHash, trie.EmptyHash, number-2,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*mocks.BlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*mocks.BlockState).On("GetHashByNumber", mock.AnythingOfType("uint")).Return(hash, nil)

	err = cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put chain we already have into work queue")
	default:
	}

	// test case where peer has a lower head than us, and they are on an invalid fork
	cs.blockState = new(mocks.BlockState)
	cs.blockState.(*mocks.BlockState).On("BestBlockHeader").Return(header, nil)
	fin, err = types.NewHeader(common.Hash{0}, trie.EmptyHash, trie.EmptyHash, number,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*mocks.BlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*mocks.BlockState).On("GetHashByNumber", mock.AnythingOfType("uint")).
		Return(common.Hash{}, nil)

	mockNetwork := NewMockNetwork(ctrl)
	mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  -4096,
		Reason: "Bad block announcement",
	}, peer.ID("noot"))
	cs.network = mockNetwork

	err = cs.setPeerHead(testPeer, hash, number)
	require.True(t, errors.Is(err, errPeerOnInvalidFork))
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put invalid fork into work queue")
	default:
	}

	// test case where peer has a lower head than us, but they are on a valid fork (that is not our chain)
	cs.blockState = new(mocks.BlockState)
	cs.blockState.(*mocks.BlockState).On("BestBlockHeader").Return(header, nil)
	fin, err = types.NewHeader(common.Hash{0}, trie.EmptyHash, trie.EmptyHash, number-2,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*mocks.BlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*mocks.BlockState).On("GetHashByNumber", mock.AnythingOfType("uint")).
		Return(common.Hash{}, nil)
	cs.blockState.(*mocks.BlockState).On("HasHeader", mock.AnythingOfType("common.Hash")).Return(true, nil)

	err = cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put fork we already have into work queue")
	default:
	}
}

func TestChainSync_sync_bootstrap_withWorkerError_Integration(t *testing.T) {
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

func TestChainSync_sync_tip_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	cs.blockState = new(mocks.BlockState)
	header, err := types.NewHeader(common.Hash{0}, trie.EmptyHash, trie.EmptyHash, 1000,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*mocks.BlockState).On("BestBlockHeader").Return(header, nil)
	cs.blockState.(*mocks.BlockState).On("GetHighestFinalisedHeader").Return(header, nil)

	go cs.sync()
	defer cs.cancel()

	testPeer := peer.ID("noot")
	cs.peerState[testPeer] = &peerState{
		number: 999,
	}

	cs.workQueue <- cs.peerState[testPeer]
	time.Sleep(time.Second)
	require.Equal(t, tip, cs.state)
}

func TestWorkerToRequests_Integration(t *testing.T) {
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

	testCases := []testCase{
		{
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
		{
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
		{
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
		{
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
		{
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
		{
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
		{
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
		{
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
		{
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

	for i, tc := range testCases {
		reqs, err := workerToRequests(tc.w)
		require.NoError(t, err, fmt.Sprintf("case %d failed", i))
		require.Equal(t, len(tc.expected), len(reqs), fmt.Sprintf("case %d failed", i))
		require.Equal(t, tc.expected, reqs, fmt.Sprintf("case %d failed", i))
	}
}

func TestValidateBlockData(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	mockNetwork := NewMockNetwork(ctrl)
	mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  -1048576,
		Reason: "Incomplete header",
	}, peer.ID(""))
	cs.network = mockNetwork

	req := &network.BlockRequestMessage{
		RequestedData: bootstrapRequestData,
	}

	err := cs.validateBlockData(req, nil, "")
	require.Equal(t, errNilBlockData, err)

	err = cs.validateBlockData(req, &types.BlockData{}, "")
	require.Equal(t, errNilHeaderInResponse, err)

	err = cs.validateBlockData(req, &types.BlockData{
		Header: &types.Header{},
	}, "")
	require.ErrorIs(t, err, errNilBodyInResponse)

	err = cs.validateBlockData(req, &types.BlockData{
		Header: &types.Header{},
		Body:   &types.Body{},
	}, "")
	require.NoError(t, err)
}

func TestChainSync_validateResponse_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().HasHeader(common.Hash{}).Return(true, nil).Times(4)
	cs.blockState = mockBlockState

	err := cs.validateResponse(nil, nil, "")
	require.Equal(t, errEmptyBlockData, err)

	req := &network.BlockRequestMessage{
		RequestedData: network.RequestedDataHeader,
	}

	resp := &network.BlockResponseMessage{
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
	}

	hash := (&types.Header{
		Number: 2,
	}).Hash()
	err = cs.validateResponse(req, resp, "")
	require.Equal(t, errResponseIsNotChain, err)
	require.True(t, cs.pendingBlocks.hasBlock(hash))
	cs.pendingBlocks.removeBlock(hash)

	parent := (&types.Header{
		Number: 1,
	}).Hash()
	header3 := &types.Header{
		ParentHash: parent,
		Number:     3,
	}
	resp = &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: &types.Header{
					Number: 1,
				},
				Body: &types.Body{},
			},
			{
				Hash:          header3.Hash(),
				Header:        header3,
				Body:          &types.Body{},
				Justification: &[]byte{0},
			},
		},
	}

	hash = (&types.Header{
		ParentHash: parent,
		Number:     3,
	}).Hash()
	err = cs.validateResponse(req, resp, "")
	require.Equal(t, errResponseIsNotChain, err)
	require.True(t, cs.pendingBlocks.hasBlock(hash))
	bd := cs.pendingBlocks.getBlock(hash)
	require.NotNil(t, bd.justification)
	cs.pendingBlocks.removeBlock(hash)

	parent = (&types.Header{
		Number: 2,
	}).Hash()
	resp = &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: &types.Header{
					Number: 2,
				},
				Body: &types.Body{},
			},
			{
				Header: &types.Header{
					ParentHash: parent,
					Number:     3,
				},
				Body: &types.Body{},
			},
		},
	}

	err = cs.validateResponse(req, resp, "")
	require.NoError(t, err)
	require.False(t, cs.pendingBlocks.hasBlock(hash))

	req = &network.BlockRequestMessage{
		RequestedData: network.RequestedDataJustification,
	}
	resp = &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Justification: &[]byte{0},
			},
		},
	}

	err = cs.validateResponse(req, resp, "")
	require.NoError(t, err)
	require.False(t, cs.pendingBlocks.hasBlock(hash))
}

func TestChainSync_validateResponse_firstBlock_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	bs := new(mocks.BlockState)
	bs.On("HasHeader", mock.AnythingOfType("common.Hash")).Return(false, nil)
	cs.blockState = bs

	req := &network.BlockRequestMessage{
		RequestedData: bootstrapRequestData,
	}

	header := &types.Header{
		Number: 2,
	}

	resp := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Hash: header.Hash(),
				Header: &types.Header{
					Number: 2,
				},
				Body:          &types.Body{},
				Justification: &[]byte{0},
			},
		},
	}

	err := cs.validateResponse(req, resp, "")
	require.True(t, errors.Is(err, errUnknownParent))
	require.True(t, cs.pendingBlocks.hasBlock(header.Hash()))
	bd := cs.pendingBlocks.getBlock(header.Hash())
	require.NotNil(t, bd.header)
	require.NotNil(t, bd.body)
	require.NotNil(t, bd.justification)
}

func TestChainSync_doSync_Integration(t *testing.T) {
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

	cs.network = new(mocks.Network)
	cs.network.(*mocks.Network).On("DoBlockRequest", mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.BlockRequestMessage")).Return(resp, nil)

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
	cs.network = new(mocks.Network)
	cs.network.(*mocks.Network).On("DoBlockRequest", mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.BlockRequestMessage")).Return(resp, nil)
	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.Nil(t, workerErr)

	bd = readyBlocks.pop(context.Background())
	require.NotNil(t, bd)
	require.Equal(t, resp.BlockData[0], bd)

	bd = readyBlocks.pop(context.Background())
	require.NotNil(t, bd)
	require.Equal(t, resp.BlockData[1], bd)
}

func TestHandleReadyBlock_Integration(t *testing.T) {
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

	ctx := context.Background()
	require.Equal(t, block1.ToBlockData(), readyBlocks.pop(ctx))
	require.Equal(t, block2.ToBlockData(), readyBlocks.pop(ctx))
	require.Equal(t, block3.ToBlockData(), readyBlocks.pop(ctx))
}

func TestChainSync_determineSyncPeers_Integration(t *testing.T) {
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
