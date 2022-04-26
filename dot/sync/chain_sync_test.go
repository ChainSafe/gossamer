// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"

	"github.com/ChainSafe/gossamer/dot/network"
	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
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
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChainSync_SetPeerHead(t *testing.T) {
	t.Parallel()

	cs := newTestChainSync(t)

	testPeer := peer.ID("noot")
	hash := common.Hash{0xa, 0xb}
	const number = 1000
	err := cs.setPeerHead(testPeer, hash, number)
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
	cs.blockState = new(syncmocks.BlockState)
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, number,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.BlockState).On("BestBlockHeader").Return(header, nil)
	fin, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, number-2,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.BlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.BlockState).On("GetHashByNumber", mock.AnythingOfType("uint")).Return(hash, nil)

	err = cs.setPeerHead(testPeer, hash, number-1)
	require.NoError(t, err)
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number - 1,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put chain we already have into work queue")
	default:
	}

	// test case where peer has a lower head than us, and they are on an invalid fork
	cs.blockState = new(syncmocks.BlockState)
	cs.blockState.(*syncmocks.BlockState).On("BestBlockHeader").Return(header, nil)
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, number,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.BlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.BlockState).On("GetHashByNumber", mock.AnythingOfType("uint")).Return(common.Hash{}, nil)

	err = cs.setPeerHead(testPeer, hash, number-1)
	require.True(t, errors.Is(err, errPeerOnInvalidFork))
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number - 1,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put invalid fork into work queue")
	default:
	}

	// test case where peer has a lower head than us, but they are on a valid fork (that is not our chain)
	cs.blockState = new(syncmocks.BlockState)
	cs.blockState.(*syncmocks.BlockState).On("BestBlockHeader").Return(header, nil)
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, number-2,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.BlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.BlockState).On("GetHashByNumber", mock.AnythingOfType("uint")).Return(common.
		Hash{}, nil)
	cs.blockState.(*syncmocks.BlockState).On("HasHeader", mock.AnythingOfType("common.Hash")).Return(true, nil)

	err = cs.setPeerHead(testPeer, hash, number-1)
	require.NoError(t, err)
	expected = &peerState{
		who:    testPeer,
		hash:   hash,
		number: number - 1,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put fork we already have into work queue")
	default:
	}
}

func TestChainSync_sync_bootstrap_withWorkerError(t *testing.T) {
	t.Parallel()

	cs := newTestChainSync(t)
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
	case <-time.After(testTimeout):
		t.Fatal("did not get worker response")
	}

	require.Equal(t, bootstrap, cs.state)
}

func TestChainSync_sync_tip(t *testing.T) {
	t.Parallel()

	cs := newTestChainSync(t)
	cs.blockState = new(syncmocks.BlockState)
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, 1000,
		types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.BlockState).On("BestBlockHeader").Return(header, nil)
	cs.blockState.(*syncmocks.BlockState).On("GetHighestFinalisedHeader").Return(header, nil)

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

func TestChainSync_getTarget(t *testing.T) {
	cs := newTestChainSync(t)
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

func TestChainSync_validateResponse(t *testing.T) {
	t.Parallel()

	cs := newTestChainSync(t)
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

func TestChainSync_validateResponse_firstBlock(t *testing.T) {
	t.Parallel()

	cs := newTestChainSync(t)
	bs := new(syncmocks.BlockState)
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

func TestChainSync_doSync(t *testing.T) {
	t.Parallel()

	readyBlocks := newBlockQueue(maxResponseSize)
	cs := newTestChainSyncWithReadyBlocks(t, readyBlocks)

	max := uint32(1)
	req := &network.BlockRequestMessage{
		RequestedData: bootstrapRequestData,
		StartingBlock: *variadic.MustNewUint32OrHash(1),
		EndBlockHash:  nil,
		Direction:     network.Ascending,
		Max:           &max,
	}

	workerErr := cs.doSync(req, make(map[peer.ID]struct{}))
	require.NotNil(t, workerErr)
	require.Equal(t, errNoPeers, workerErr.err)

	cs.peerState["noot"] = &peerState{
		number: 100,
	}

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

	cs.network = new(syncmocks.Network)
	cs.network.(*syncmocks.Network).On("DoBlockRequest", mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.BlockRequestMessage")).Return(resp, nil)

	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.Nil(t, workerErr)
	bd := readyBlocks.pop()
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
	cs.network = new(syncmocks.Network)
	cs.network.(*syncmocks.Network).On("DoBlockRequest", mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.BlockRequestMessage")).Return(resp, nil)
	workerErr = cs.doSync(req, make(map[peer.ID]struct{}))
	require.Nil(t, workerErr)

	bd = readyBlocks.pop()
	require.NotNil(t, bd)
	require.Equal(t, resp.BlockData[0], bd)

	bd = readyBlocks.pop()
	require.NotNil(t, bd)
	require.Equal(t, resp.BlockData[1], bd)
}

func TestHandleReadyBlock(t *testing.T) {
	t.Parallel()

	readyBlocks := newBlockQueue(maxResponseSize)
	cs := newTestChainSyncWithReadyBlocks(t, readyBlocks)

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

	require.Equal(t, block1.ToBlockData(), readyBlocks.pop())
	require.Equal(t, block2.ToBlockData(), readyBlocks.pop())
	require.Equal(t, block3.ToBlockData(), readyBlocks.pop())
}

func TestChainSync_determineSyncPeers(t *testing.T) {
	t.Parallel()

	cs := newTestChainSync(t)

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

	ctrl := gomock.NewController(t)

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().BestBlockHeader().Return(&types.Header{
		Number: 1,
	}, nil).AnyTimes()
	mockBlockState.EXPECT().GetHighestFinalisedHeader().Return(&types.Header{
		Number: 1,
	}, nil).AnyTimes()

	mockNetwork := NewMockNetwork(ctrl)
	mockNetwork.EXPECT().Peers().Return([]common.PeerInfo{}).AnyTimes()

	type fields struct {
		blockState  BlockState
		network     Network
		state       chainSyncState
		benchmarker *syncBenchmarker
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "state bootstrap",
			fields: fields{
				blockState:  mockBlockState,
				network:     mockNetwork,
				benchmarker: newSyncBenchmarker(10),
				state:       bootstrap,
			},
		},
		{
			name: "case tip",
			fields: fields{
				blockState:  mockBlockState,
				network:     mockNetwork,
				benchmarker: newSyncBenchmarker(10),
				state:       tip,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			cs := &chainSync{
				ctx:                   ctx,
				cancel:                cancel,
				blockState:            tt.fields.blockState,
				network:               tt.fields.network,
				state:                 tt.fields.state,
				benchmarker:           tt.fields.benchmarker,
				logSyncSpeedFrequency: time.Nanosecond,
			}
			go cs.logSyncSpeed()
			time.Sleep(time.Nanosecond)
			cancel()
		})
	}
}

func newMockBlockStateForChainSyncTests(ctrl *gomock.Controller) BlockState {
	mock := NewMockBlockState(ctrl)
	mock.EXPECT().BestBlockHeader().Return(&types.Header{}, nil).AnyTimes()
	mock.EXPECT().HasHeader(gomock.AssignableToTypeOf(common.Hash{})).Return(true, nil).AnyTimes()
	return mock
}

func newMockDisjointBlockSet(ctrl *gomock.Controller) DisjointBlockSet {
	mock := NewMockDisjointBlockSet(ctrl)
	mock.EXPECT().run(gomock.Any()).AnyTimes()
	mock.EXPECT().addHeader(gomock.AssignableToTypeOf(&types.Header{})).Return(nil).AnyTimes()
	mock.EXPECT().addHashAndNumber(gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf(uint(0))).Return(nil).AnyTimes()
	return mock
}

func Test_chainSync_start(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	type fields struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
		benchmarker   *syncBenchmarker
		slotDuration  time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "base case",
			fields: fields{
				blockState:    newMockBlockStateForChainSyncTests(ctrl),
				pendingBlocks: newMockDisjointBlockSet(ctrl),
				slotDuration:  defaultSlotDuration,
				benchmarker:   newSyncBenchmarker(1),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			cs := &chainSync{
				ctx:                   ctx,
				cancel:                cancel,
				blockState:            tt.fields.blockState,
				pendingBlocks:         tt.fields.pendingBlocks,
				benchmarker:           tt.fields.benchmarker,
				slotDuration:          tt.fields.slotDuration,
				logSyncSpeedFrequency: time.Second,
			}
			cs.start()
			time.Sleep(time.Millisecond)
			cs.stop()
		})
	}
}

func Test_chainSync_setBlockAnnounce(t *testing.T) {
	ctrl := gomock.NewController(t)
	type fields struct {
		blockState    BlockState
		pendingBlocks DisjointBlockSet
	}
	type args struct {
		from   peer.ID
		header *types.Header
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "base case",
			args: args{
				header: &types.Header{Number: 2},
			},
			fields: fields{
				blockState:    newMockBlockStateForChainSyncTests(ctrl),
				pendingBlocks: newMockDisjointBlockSet(ctrl),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				blockState:    tt.fields.blockState,
				pendingBlocks: tt.fields.pendingBlocks,
			}

			if err := cs.setBlockAnnounce(tt.args.from, tt.args.header); (err != nil) != tt.wantErr {
				t.Errorf("chainSync.setBlockAnnounce() error = %v, wantErr %v", err, tt.wantErr)
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
			}
			if gotHighestBlock != tt.wantHighestBlock {
				t.Errorf("chainSync.getHighestBlock() = %v, want %v", gotHighestBlock, tt.wantHighestBlock)
			}
		})
	}
}

func Test_chainSync_handleResult(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	tests := map[string]struct {
		maxWorkerRetries uint16
		res              *worker
		err              error
	}{
		"res.err == nil": {
			res: &worker{},
		},
		"res.err.err.Error() == context.Canceled": {
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: context.Canceled,
				},
			},
		},
		"res.err.err.Error() == context.DeadlineExceeded": {
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: context.DeadlineExceeded,
				},
			},
		},
		"res.err.err.Error() == errNoPeers": {
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errNoPeers,
				},
			},
		},
		"res.err.err.Error() == protocol not supported": {
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New("protocol not supported"),
				},
			},
		},
		"no error, no retries": {
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New(""),
				},
			},
		},
		"no error, maxWorkerRetries 2": {
			maxWorkerRetries: 2,
			res: &worker{
				ctx: context.Background(),
				err: &workerError{
					err: errors.New(""),
				},
			},
		},
	}
	for testName, tt := range tests {
		tt := tt
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			mockNetwork := NewMockNetwork(ctrl)
			mockNetwork.EXPECT().ReportPeer(gomock.AssignableToTypeOf(peerset.ReputationChange{}),
				gomock.AssignableToTypeOf(peer.ID(""))).AnyTimes()
			mockWorkHandler := NewMockworkHandler(ctrl)
			mockWorkHandler.EXPECT().handleWorkerResult(tt.res).DoAndReturn(func(res *worker) (*worker, error) {
				return res, nil
			}).AnyTimes()
			mockWorkHandler.EXPECT().hasCurrentWorker(gomock.AssignableToTypeOf(&worker{}),
				gomock.AssignableToTypeOf(map[uint64]*worker{})).Return(true).AnyTimes()

			cs := &chainSync{
				network:          mockNetwork,
				workerState:      newWorkerState(),
				handler:          mockWorkHandler,
				maxWorkerRetries: tt.maxWorkerRetries,
			}
			err := cs.handleResult(tt.res)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
