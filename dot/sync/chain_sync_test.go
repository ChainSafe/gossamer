// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	defaultMinPeers     = 1
	defaultMaxPeers     = 5
	testTimeout         = time.Second * 5
	defaultSlotDuration = time.Second * 6
)

func newTestChainSync(t *testing.T) (*chainSync, *blockQueue) {
	header, err := types.NewHeader(common.NewHash([]byte{0}),
		trie.EmptyHash, trie.EmptyHash, 0, types.NewDigest())
	require.NoError(t, err)

	bs := new(syncmocks.BlockState)
	bs.On("BestBlockHeader").Return(header, nil)
	bs.On("GetFinalisedNotifierChannel").Return(make(chan *types.FinalisationInfo, 128), nil)
	bs.On("HasHeader", mock.AnythingOfType("common.Hash")).Return(true, nil)

	net := new(syncmocks.Network)
	net.On("DoBlockRequest", mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.BlockRequestMessage")).Return(nil, nil)
	net.On("ReportPeer", mock.AnythingOfType("peerset.ReputationChange"), mock.AnythingOfType("peer.ID"))

	readyBlocks := newBlockQueue(maxResponseSize)

	cfg := &chainSyncConfig{
		bs:            bs,
		net:           net,
		readyBlocks:   readyBlocks,
		pendingBlocks: newDisjointBlockSet(pendingBlocksLimit),
		minPeers:      defaultMinPeers,
		maxPeers:      defaultMaxPeers,
		slotDuration:  defaultSlotDuration,
	}

	cs := newChainSync(cfg)
	return cs, readyBlocks
}

func TestChainSync_SetPeerHead(t *testing.T) {
	cs, _ := newTestChainSync(t)

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
	header, err := types.NewHeader(common.NewHash([]byte{0}),
		trie.EmptyHash, trie.EmptyHash, number, types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.BlockState).On("BestBlockHeader").Return(header, nil)
	fin, err := types.NewHeader(common.NewHash([]byte{0}),
		trie.EmptyHash, trie.EmptyHash, number-2, types.NewDigest())
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
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, number, types.NewDigest())
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
	fin, err = types.NewHeader(
		common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash,
		number-2, types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.BlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.BlockState).On("GetHashByNumber", mock.AnythingOfType("uint")).Return(common.Hash{}, nil)
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
	cs, _ := newTestChainSync(t)

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
	cs, _ := newTestChainSync(t)
	cs.blockState = new(syncmocks.BlockState)
	header, err := types.NewHeader(common.NewHash([]byte{0}),
		trie.EmptyHash, trie.EmptyHash, 1000, types.NewDigest())
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
	cs, _ := newTestChainSync(t)

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

	require.Equal(t, uint(130), cs.getTarget()) // sum:650/count:5 = avg:130

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
	cs, _ := newTestChainSync(t)
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

func TestChainSync_validateResponse(t *testing.T) {
	cs, _ := newTestChainSync(t)
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
	cs, _ := newTestChainSync(t)
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
	cs, readyBlocks := newTestChainSync(t)

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
	cs.network.(*syncmocks.Network).On("DoBlockRequest",
		mock.AnythingOfType("peer.ID"),
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
	cs.network.(*syncmocks.Network).On("DoBlockRequest",
		mock.AnythingOfType("peer.ID"),
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
	cs, readyBlocks := newTestChainSync(t)

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
	cs, _ := newTestChainSync(t)

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

func TestChainSync_highestBlock(t *testing.T) {
	type input struct {
		peerState map[peer.ID]*peerState
	}
	type output struct {
		highestBlock uint
		err          error
	}
	type test struct {
		name string
		in   input
		out  output
	}
	tests := []test{
		{
			name: "when has an empty map should return 0, errNoPeers",
			in: input{
				peerState: map[peer.ID]*peerState{},
			},
			out: output{
				highestBlock: 0,
				err:          errNoPeers,
			},
		},
		{
			name: "when has a nil map should return 0, errNoPeers",
			in: input{
				peerState: nil,
			},
			out: output{
				highestBlock: 0,
				err:          errNoPeers,
			},
		},
		{
			name: "when has only one peer with number 90 should return 90, nil",
			in: input{
				peerState: map[peer.ID]*peerState{
					"idtest": {number: 90},
				},
			},
			out: output{
				highestBlock: 90,
				err:          nil,
			},
		},
		{
			name: "when has two peers (p1, p2) with p1.number 90 and p2.number 190 should return 190, nil",
			in: input{
				peerState: map[peer.ID]*peerState{
					"idtest#1": {number: 90},
					"idtest#2": {number: 190},
				},
			},
			out: output{
				highestBlock: 190,
				err:          nil,
			},
		},
		{
			name: "when has two peers (p1, p2) with p1.number 190 and p2.number 90 should return 190, nil",
			in: input{
				peerState: map[peer.ID]*peerState{
					"idtest#1": {number: 190},
					"idtest#2": {number: 90},
				},
			},
			out: output{
				highestBlock: 190,
				err:          nil,
			},
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			cs, _ := newTestChainSync(t)
			cs.peerState = ts.in.peerState

			highestBlock, err := cs.getHighestBlock()
			require.ErrorIs(t, err, ts.out.err)
			require.Equal(t, highestBlock, ts.out.highestBlock)
		})
	}
}
