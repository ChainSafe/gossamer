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

package network

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func createBlockRequests(start, end int64) []*BlockRequestMessage {
	if start > end {
		return nil
	}

	numReqs := (end - start) / int64(blockRequestSize)
	if numReqs > int64(blockRequestBufferSize) {
		numReqs = int64(blockRequestBufferSize)
	}

	if end-start < int64(blockRequestSize) {
		// +1 because we want to include the block w/ the ending number
		req := createBlockRequest(start, uint32(end-start)+1)
		return []*BlockRequestMessage{req}
	}

	reqs := make([]*BlockRequestMessage, numReqs)
	for i := 0; i < int(numReqs); i++ {
		offset := i * int(blockRequestSize)
		reqs[i] = createBlockRequest(start+int64(offset), blockRequestSize)
	}
	return reqs
}

func TestDecodeSyncMessage(t *testing.T) {
	s := &Service{
		ctx: context.Background(),
	}

	s.syncQueue = newSyncQueue(s)
	testPeer := peer.ID("noot")

	reqEnc, err := testBlockRequestMessage.Encode()
	require.NoError(t, err)

	msg, err := s.decodeSyncMessage(reqEnc, testPeer)
	require.NoError(t, err)

	req, ok := msg.(*BlockRequestMessage)
	require.True(t, ok)
	require.Equal(t, testBlockRequestMessage, req)
}

func TestSyncQueue_PushResponse(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	peerID := peer.ID("noot")
	msg := &BlockResponseMessage{
		BlockData: []*types.BlockData{},
	}

	for i := 0; i < int(blockRequestSize); i++ {
		testHeader := types.Header{
			Number: big.NewInt(int64(77 + i)),
		}

		msg.BlockData = append(msg.BlockData, &types.BlockData{
			Header: testHeader.AsOptional(),
			Body:   optional.NewBody(true, []byte{0}),
		})
	}

	err := s.syncQueue.pushResponse(msg, peerID)
	require.NoError(t, err)
	require.Equal(t, 1, len(s.syncQueue.responses))
}

func TestSortRequests(t *testing.T) {
	reqs := createBlockRequests(1, int64(blockRequestSize*5)+1)
	sreqs := []*syncRequest{}
	for _, req := range reqs {
		sreqs = append(sreqs, &syncRequest{
			req: req,
		})
	}

	expected := make([]*syncRequest, len(sreqs))
	copy(expected, sreqs)

	rand.Shuffle(len(sreqs), func(i, j int) { sreqs[i], sreqs[j] = sreqs[j], sreqs[i] })
	sortRequests(sreqs)
	require.Equal(t, expected, sreqs)
}

func TestSortRequests_RemoveDuplicates(t *testing.T) {
	reqs := createBlockRequests(1, int64(blockRequestSize*5)+1)
	sreqs := []*syncRequest{}
	for _, req := range reqs {
		sreqs = append(sreqs, &syncRequest{
			req: req,
		})
	}

	expected := make([]*syncRequest, len(sreqs))
	copy(expected, sreqs)

	dup := createBlockRequest(1, blockRequestSize)
	sreqs = append(sreqs, &syncRequest{req: dup})

	rand.Shuffle(len(sreqs), func(i, j int) { sreqs[i], sreqs[j] = sreqs[j], sreqs[i] })
	sreqs = sortRequests(sreqs)
	require.Equal(t, expected, sreqs)
}

func TestSortResponses(t *testing.T) {
	testHeader0 := types.Header{
		Number: big.NewInt(77),
	}

	testHeader1 := types.Header{
		Number: big.NewInt(78),
	}

	testHeader2 := types.Header{
		Number: big.NewInt(79),
	}

	data := []*types.BlockData{
		{
			Hash:   testHeader2.Hash(),
			Header: testHeader2.AsOptional(),
		},
		{
			Hash:   testHeader0.Hash(),
			Header: testHeader0.AsOptional(),
		},
		{
			Hash:   testHeader1.Hash(),
			Header: testHeader1.AsOptional(),
		},
	}

	expected := []*types.BlockData{
		{
			Hash:   testHeader0.Hash(),
			Header: testHeader0.AsOptional(),
		},
		{
			Hash:   testHeader1.Hash(),
			Header: testHeader1.AsOptional(),
		},
		{
			Hash:   testHeader2.Hash(),
			Header: testHeader2.AsOptional(),
		},
	}

	data = sortResponses(data)
	require.Equal(t, expected, data)
}

func TestSortResponses_RemoveDuplicated(t *testing.T) {
	testHeader0 := types.Header{
		Number: big.NewInt(77),
	}

	testHeader1 := types.Header{
		Number: big.NewInt(78),
	}

	testHeader2 := types.Header{
		Number: big.NewInt(79),
	}

	data := []*types.BlockData{
		{
			Hash:   testHeader0.Hash(),
			Header: testHeader2.AsOptional(),
		},
		{
			Hash:   testHeader0.Hash(),
			Header: testHeader0.AsOptional(),
		},
		{
			Hash:   testHeader0.Hash(),
			Header: testHeader1.AsOptional(),
		},
	}

	// should keep first block in sorted slice w/ duplicated hash
	expected := []*types.BlockData{
		{
			Hash:   testHeader0.Hash(),
			Header: testHeader0.AsOptional(),
		},
	}

	data = sortResponses(data)
	require.Equal(t, expected, data)
}

func newTestSyncQueue(t *testing.T) *syncQueue {
	s := createTestService(t, nil)
	return s.syncQueue
}

func TestSyncQueue_HandleBlockAnnounceHandshake(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)

	testNum := int64(128 * 7)

	testPeerID := peer.ID("noot")
	q.handleBlockAnnounceHandshake(uint32(testNum), testPeerID)
	score, ok := q.peerScore.Load(testPeerID)
	require.True(t, ok)
	require.Equal(t, 1, score.(int))
	require.Equal(t, testNum, q.goal)
	require.Equal(t, 6, len(q.requestCh))

	head, err := q.s.blockState.BestBlockNumber()
	require.NoError(t, err)
	expected := createBlockRequest(head.Int64(), blockRequestSize)
	req := <-q.requestCh
	require.Equal(t, &syncRequest{req: expected, to: testPeerID}, req)
}

func TestSyncQueue_HandleBlockAnnounce(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)

	testPeerID := peer.ID("noot")
	q.handleBlockAnnounce(testBlockAnnounceMessage, testPeerID)
	score, ok := q.peerScore.Load(testPeerID)
	require.True(t, ok)
	require.Equal(t, 1, score.(int))
	require.Equal(t, testBlockAnnounceMessage.Number.Int64(), q.goal)
	require.Equal(t, 6, len(q.requestCh))

	head, err := q.s.blockState.BestBlockNumber()
	require.NoError(t, err)
	expected := createBlockRequest(head.Int64(), blockRequestSize)
	req := <-q.requestCh
	require.Equal(t, &syncRequest{req: expected, to: testPeerID}, req)
}

func TestSyncQueue_ProcessBlockRequests(t *testing.T) {
	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
		LogLvl:      4,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
		LogLvl:      4,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	configC := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeC"),
		Port:        7003,
		RandSeed:    3,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeC := createTestService(t, configC)
	nodeC.noGossip = true

	// connect A and B
	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	// connect A and C
	addrInfosC, err := nodeC.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosC[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosC[0])
	}
	require.NoError(t, err)

	nodeA.syncQueue.stop()
	nodeA.syncQueue.ctx, nodeA.syncQueue.cancel = context.WithCancel(context.Background())
	defer nodeA.syncQueue.cancel()
	time.Sleep(time.Second * 3)

	nodeA.syncQueue.updatePeerScore(nodeB.host.id(), 1) // expect to try to sync with nodeB first
	go nodeA.syncQueue.processBlockRequests()
	nodeA.syncQueue.requestCh <- &syncRequest{
		req: testBlockRequestMessage,
	}

	time.Sleep(time.Second * 2)
	require.Equal(t, 128, len(nodeA.syncQueue.responses))
	testResp := testBlockResponseMessage()
	require.Equal(t, testResp.BlockData, nodeA.syncQueue.responses)
}

func TestSyncQueue_handleResponseQueue_noRequestsOrResponses(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.goal = int64(blockRequestSize) * 10
	q.ctx = context.Background()
	go q.handleResponseQueue()
	time.Sleep(time.Second * 2)
	require.Equal(t, blockRequestBufferSize, len(q.requestCh))
}

func TestSyncQueue_handleResponseQueue_responseQueueAhead(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.goal = int64(blockRequestSize) * 10
	q.ctx = context.Background()

	testHeader0 := types.Header{
		Number: big.NewInt(77),
		Digest: types.Digest{},
	}
	q.responses = append(q.responses, &types.BlockData{
		Hash:          testHeader0.Hash(),
		Header:        testHeader0.AsOptional(),
		Body:          optional.NewBody(true, []byte{4, 4, 2}),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	})

	go q.handleResponseQueue()
	time.Sleep(time.Second * 2)

	require.Equal(t, 1, len(q.requestCh))
}

func TestSyncQueue_processBlockResponses(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.goal = int64(blockRequestSize) * 10
	q.ctx = context.Background()

	testHeader0 := types.Header{
		Number: big.NewInt(0),
		Digest: types.Digest{},
	}
	go func() {
		q.responseCh <- []*types.BlockData{
			{
				Hash:          testHeader0.Hash(),
				Header:        testHeader0.AsOptional(),
				Body:          optional.NewBody(true, []byte{4, 4, 2}),
				Receipt:       optional.NewBytes(false, nil),
				MessageQueue:  optional.NewBytes(false, nil),
				Justification: optional.NewBytes(false, nil),
			},
		}
	}()

	go q.processBlockResponses()
	time.Sleep(time.Second)
	require.Equal(t, blockRequestBufferSize, len(q.requestCh))
}

func TestSyncQueue_SyncAtHead(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.ctx = context.Background()

	go q.syncAtHead()
	time.Sleep(time.Millisecond * 6100)
	select {
	case req := <-q.requestCh:
		require.Equal(t, uint64(2), req.req.StartingBlock.Uint64())
	case <-time.After(TestMessageTimeout):
		t.Fatal("did not queue request")
	}
}

func TestSyncQueue_PushRequest_NearHead(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.ctx = context.Background()
	q.goal = 0

	q.pushRequest(1, 1, "")
	select {
	case req := <-q.requestCh:
		require.Equal(t, uint64(2), req.req.StartingBlock.Uint64())
	case <-time.After(TestMessageTimeout):
		t.Fatal("did not queue request")
	}
}

func TestSyncQueue_handleBlockData_ok(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.ctx = context.Background()
	q.currStart = 129
	q.goal = 1000

	data := testBlockResponseMessage().BlockData
	q.handleBlockData(data)
	select {
	case req := <-q.requestCh:
		require.True(t, req.req.StartingBlock.IsUint64())
		require.Equal(t, uint64(129), req.req.StartingBlock.Uint64())
	case <-time.After(TestMessageTimeout):
		t.Fatal("did not queue request")
	}

	require.Equal(t, int64(0), q.currStart)
	require.Equal(t, int64(0), q.currEnd)
}

func TestSyncQueue_handleBlockDataFailure(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.ctx = context.Background()
	q.currStart = 129
	q.goal = 1000

	data := testBlockResponseMessage().BlockData
	q.handleBlockDataFailure(0, fmt.Errorf("some other error"), data)
	select {
	case req := <-q.requestCh:
		require.True(t, req.req.StartingBlock.IsUint64())
		require.Equal(t, uint64(q.currStart), req.req.StartingBlock.Uint64())
	case <-time.After(TestMessageTimeout):
		t.Fatal("did not queue request")
	}
}

func TestSyncQueue_handleBlockDataFailure_MissingParent(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.ctx = context.Background()

	data := testBlockResponseMessage().BlockData
	q.handleBlockDataFailure(0, fmt.Errorf("failed to get parent hash: Key not found"), data)
	select {
	case req := <-q.requestCh:
		require.True(t, req.req.StartingBlock.IsHash())
	case <-time.After(TestMessageTimeout):
		t.Fatal("did not queue request")
	}
}
