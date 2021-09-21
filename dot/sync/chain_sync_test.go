// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package sync

import (
	"errors"
	"fmt"
	"math/big"
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

var testTimeout = time.Second * 5

func newTestChainSync(t *testing.T) (*chainSync, <-chan *types.BlockData) { //nolint
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(0), types.NewDigest())
	require.NoError(t, err)

	bs := new(syncmocks.MockBlockState)
	bs.On("BestBlockHeader").Return(header, nil)

	net := new(syncmocks.MockNetwork)
	net.On("DoBlockRequest", mock.AnythingOfType("peer.ID"), mock.AnythingOfType("*network.BlockRequestMessage")).Return(nil, nil)

	readyBlocks := make(chan *types.BlockData, maxResponseSize)
	return newChainSync(bs, net, readyBlocks), readyBlocks
}

func TestChainSync_SetPeerHead(t *testing.T) {
	cs, _ := newTestChainSync(t)

	testPeer := peer.ID("noot")
	hash := common.Hash{0xa, 0xb}
	number := big.NewInt(1000)
	err := cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)

	expected := &peerState{
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	require.Equal(t, expected, <-cs.workQueue)
	require.True(t, cs.pendingBlocks.hasBlock(hash))

	// test case where peer has a lower head than us, but they are on the same chain as us
	cs.blockState = new(syncmocks.MockBlockState)
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(1000), types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("BestBlockHeader").Return(header, nil)
	fin, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(998), types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHashByNumber", mock.AnythingOfType("*big.Int")).Return(hash, nil)

	number = big.NewInt(999)
	err = cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)
	expected = &peerState{
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
	cs.blockState = new(syncmocks.MockBlockState)
	cs.blockState.(*syncmocks.MockBlockState).On("BestBlockHeader").Return(header, nil)
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(1000), types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHashByNumber", mock.AnythingOfType("*big.Int")).Return(common.Hash{}, nil)

	number = big.NewInt(999)
	err = cs.setPeerHead(testPeer, hash, number)
	require.True(t, errors.Is(err, errPeerOnInvalidFork))
	expected = &peerState{
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
	cs.blockState = new(syncmocks.MockBlockState)
	cs.blockState.(*syncmocks.MockBlockState).On("BestBlockHeader").Return(header, nil)
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(998), types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHashByNumber", mock.AnythingOfType("*big.Int")).Return(common.Hash{}, nil)
	cs.blockState.(*syncmocks.MockBlockState).On("HasHeader", mock.AnythingOfType("common.Hash")).Return(true, nil)

	number = big.NewInt(999)
	err = cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)
	expected = &peerState{
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

func TestChainSync_sync_bootstrap_withWorkerError(t *testing.T) {
	cs, _ := newTestChainSync(t)

	go cs.sync()
	defer cs.cancel()

	testPeer := peer.ID("noot")
	cs.peerState[testPeer] = &peerState{
		number: big.NewInt(1000),
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
	cs.blockState = new(syncmocks.MockBlockState)
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(1000), types.NewDigest())
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("BestBlockHeader").Return(header, nil)

	go cs.sync()
	defer cs.cancel()

	testPeer := peer.ID("noot")
	cs.peerState[testPeer] = &peerState{
		number: big.NewInt(999),
	}

	cs.workQueue <- cs.peerState[testPeer]
	time.Sleep(time.Second)
	require.Equal(t, tip, cs.state)
}

func TestChainSync_getTarget(t *testing.T) {
	cs, _ := newTestChainSync(t)
	require.Equal(t, big.NewInt(2<<32-1), cs.getTarget())

	cs.peerState = map[peer.ID]*peerState{
		"testA": {
			number: big.NewInt(1000),
		},
	}

	require.Equal(t, big.NewInt(1000), cs.getTarget())

	cs.peerState = map[peer.ID]*peerState{
		"testA": {
			number: big.NewInt(1000),
		},
		"testB": {
			number: big.NewInt(2000),
		},
	}

	require.Equal(t, big.NewInt(1500), cs.getTarget())
}

func TestWorkerToRequests(t *testing.T) {
	_, err := workerToRequests(&worker{})
	require.Equal(t, errWorkerMissingStartNumber, err)

	w := &worker{
		startNumber: big.NewInt(1),
	}
	_, err = workerToRequests(w)
	require.Equal(t, errWorkerMissingTargetNumber, err)

	w = &worker{
		startNumber:  big.NewInt(10),
		targetNumber: big.NewInt(1),
		direction:    network.Ascending,
	}
	_, err = workerToRequests(w)
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
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(1 + maxResponseSize),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint64OrHash(1),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(1 + (maxResponseSize * 2)),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint64OrHash(1),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: *variadic.MustNewUint64OrHash(1 + maxResponseSize),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(10),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint64OrHash(1),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max9,
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(10),
				targetNumber: big.NewInt(1),
				direction:    network.Descending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint64OrHash(10),
					EndBlockHash:  nil,
					Direction:     network.Descending,
					Max:           &max9,
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(1 + maxResponseSize + (maxResponseSize / 2)),
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint64OrHash(1),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max128,
				},
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: *variadic.MustNewUint64OrHash(1 + maxResponseSize),
					EndBlockHash:  nil,
					Direction:     network.Ascending,
					Max:           &max64,
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(10),
				targetHash:   common.Hash{0xa},
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint64OrHash(1),
					EndBlockHash:  &(common.Hash{0xa}),
					Direction:     network.Ascending,
					Max:           &max9,
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				startHash:    common.Hash{0xb},
				targetNumber: big.NewInt(10),
				targetHash:   common.Hash{0xc},
				direction:    network.Ascending,
				requestData:  bootstrapRequestData,
			},
			expected: []*network.BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: *variadic.MustNewUint64OrHash(common.Hash{0xb}),
					EndBlockHash:  &(common.Hash{0xc}),
					Direction:     network.Ascending,
					Max:           &max9,
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
	req := &network.BlockRequestMessage{
		RequestedData: bootstrapRequestData,
	}

	err := validateBlockData(req, nil)
	require.Equal(t, errNilBlockData, err)

	err = validateBlockData(req, &types.BlockData{})
	require.Equal(t, errNilHeaderInResponse, err)

	err = validateBlockData(req, &types.BlockData{
		Header: &types.Header{},
	})
	require.Equal(t, errNilBodyInResponse, err)

	err = validateBlockData(req, &types.BlockData{
		Header: &types.Header{},
		Body:   &types.Body{},
	})
	require.NoError(t, err)
}

func TestChainSync_validateResponse(t *testing.T) {
	cs, _ := newTestChainSync(t)
	err := cs.validateResponse(nil, nil)
	require.Equal(t, errEmptyBlockData, err)

	req := &network.BlockRequestMessage{
		RequestedData: network.RequestedDataHeader,
	}

	resp := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: &types.Header{
					Number: big.NewInt(1),
				},
				Body: &types.Body{},
			},
			{
				Header: &types.Header{
					Number: big.NewInt(2),
				},
				Body: &types.Body{},
			},
		},
	}

	hash := (&types.Header{
		Number: big.NewInt(2),
	}).Hash()
	err = cs.validateResponse(req, resp)
	require.Equal(t, errResponseIsNotChain, err)
	require.True(t, cs.pendingBlocks.hasBlock(hash))
	cs.pendingBlocks.removeBlock(hash)

	parent := (&types.Header{
		Number: big.NewInt(1),
	}).Hash()
	resp = &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: &types.Header{
					Number: big.NewInt(1),
				},
				Body: &types.Body{},
			},
			{
				Header: &types.Header{
					ParentHash: parent,
					Number:     big.NewInt(3),
				},
				Body: &types.Body{},
			},
		},
	}

	hash = (&types.Header{
		ParentHash: parent,
		Number:     big.NewInt(3),
	}).Hash()
	err = cs.validateResponse(req, resp)
	require.Equal(t, errResponseIsNotChain, err)
	require.True(t, cs.pendingBlocks.hasBlock(hash))
	cs.pendingBlocks.removeBlock(hash)

	parent = (&types.Header{
		Number: big.NewInt(2),
	}).Hash()
	resp = &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: &types.Header{
					Number: big.NewInt(2),
				},
				Body: &types.Body{},
			},
			{
				Header: &types.Header{
					ParentHash: parent,
					Number:     big.NewInt(3),
				},
				Body: &types.Body{},
			},
		},
	}

	err = cs.validateResponse(req, resp)
	require.NoError(t, err)
	require.False(t, cs.pendingBlocks.hasBlock(hash))
}

func TestChainSync_doSync(t *testing.T) {
	cs, readyBlocks := newTestChainSync(t)

	max := uint32(1)
	req := &network.BlockRequestMessage{
		RequestedData: bootstrapRequestData,
		StartingBlock: *variadic.MustNewUint64OrHash(1),
		EndBlockHash:  nil,
		Direction:     network.Ascending,
		Max:           &max,
	}

	workerErr := cs.doSync(req)
	require.NotNil(t, workerErr)
	require.Equal(t, errNoPeers, workerErr.err)

	cs.peerState["noot"] = &peerState{
		number: big.NewInt(100),
	}

	workerErr = cs.doSync(req)
	require.NotNil(t, workerErr)
	require.Equal(t, errNilResponse, workerErr.err)

	resp := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: &types.Header{
					Number: big.NewInt(1),
				},
				Body: &types.Body{},
			},
		},
	}

	cs.network = new(syncmocks.MockNetwork)
	cs.network.(*syncmocks.MockNetwork).On("DoBlockRequest", mock.AnythingOfType("peer.ID"), mock.AnythingOfType("*network.BlockRequestMessage")).Return(resp, nil)

	workerErr = cs.doSync(req)
	require.Nil(t, workerErr)
	select {
	case bd := <-readyBlocks:
		require.Equal(t, resp.BlockData[0], bd)
	default:
		t.Fatal("expected ready block")
	}

	parent := (&types.Header{
		Number: big.NewInt(2),
	}).Hash()
	resp = &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: &types.Header{
					ParentHash: parent,
					Number:     big.NewInt(3),
				},
				Body: &types.Body{},
			},
			{
				Header: &types.Header{
					Number: big.NewInt(2),
				},
				Body: &types.Body{},
			},
		},
	}

	// test to see if descending blocks get reversed
	req.Direction = network.Descending
	cs.network = new(syncmocks.MockNetwork)
	cs.network.(*syncmocks.MockNetwork).On("DoBlockRequest", mock.AnythingOfType("peer.ID"), mock.AnythingOfType("*network.BlockRequestMessage")).Return(resp, nil)
	workerErr = cs.doSync(req)
	require.Nil(t, workerErr)

	select {
	case bd := <-readyBlocks:
		require.Equal(t, resp.BlockData[0], bd)
	default:
		t.Fatal("expected ready block")
	}

	select {
	case bd := <-readyBlocks:
		require.Equal(t, resp.BlockData[1], bd)
	default:
		t.Fatal("expected ready block")
	}
}
