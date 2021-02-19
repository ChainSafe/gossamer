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
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestDecodeSyncMessage(t *testing.T) {
	s := &Service{
		ctx: context.Background(),
	}

	s.syncQueue = newSyncQueue(s)

	testPeer := peer.ID("noot")

	testBlockResponseMessage := &BlockResponseMessage{
		BlockData: []*types.BlockData{},
	}

	reqEnc, err := testBlockRequestMessage.Encode()
	require.NoError(t, err)

	msg, err := s.decodeSyncMessage(reqEnc, testPeer)
	require.NoError(t, err)

	req, ok := msg.(*BlockRequestMessage)
	require.True(t, ok)
	require.Equal(t, testBlockRequestMessage, req)

	s.syncQueue.syncing[testPeer] = struct{}{}

	respEnc, err := testBlockResponseMessage.Encode()
	require.NoError(t, err)

	msg, err = s.decodeSyncMessage(respEnc, testPeer)
	require.NoError(t, err)
	resp, ok := msg.(*BlockResponseMessage)
	require.True(t, ok)
	require.Equal(t, testBlockResponseMessage, resp)
}

func TestBeginSyncingProtectsPeer(t *testing.T) {
	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true
	nodeA.syncQueue.stop()

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true
	nodeB.syncQueue.stop()

	// connect A and B
	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeA.host.connect(*addrInfosB[0])
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	var (
		peerID = nodeB.host.id()
		msg    = testBlockRequestMessage
	)

	err = nodeA.syncQueue.beginSyncingWithPeer(peerID, msg)
	require.NoError(t, err)
	require.True(t, nodeA.host.h.ConnManager().IsProtected(peerID, ""))
}

func TestHandleSyncMessage_BlockResponse(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	testHeader := types.Header{
		Number: big.NewInt(77),
	}

	peerID := peer.ID("noot")
	msg := &BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: testHeader.AsOptional(),
			},
		},
	}

	var br *blockRange
	go func() {
		br = <-s.syncQueue.gotRespCh
	}()

	s.syncQueue.syncing[peerID] = struct{}{}
	s.handleSyncMessage(peerID, msg)
	require.Equal(t, 1, len(s.syncQueue.responses))
	require.Equal(t, &blockRange{
		start: 77,
		end:   77,
		from:  peerID,
	}, br)
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

func TestSyncQueue_SetBlockRequests_ShouldBeEmpty(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	q.goal = 0

	testPeerID := peer.ID("noot")
	q.setBlockRequests(testPeerID)
	require.Equal(t, 0, len(q.requests))
}

func TestSyncQueue_SetBlockRequests(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	q.goal = 10000

	testPeerID := peer.ID("noot")
	q.setBlockRequests(testPeerID)
	require.Equal(t, int(blockRequestQueueSize), len(q.requests))
}

func TestSyncQueue_HandleBlockAnnounceHandshake(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()

	testNum := int64(99)

	testPeerID := peer.ID("noot")
	q.handleBlockAnnounceHandshake(uint32(testNum), testPeerID)
	require.Equal(t, 1, q.peerScore[testPeerID])
	require.Equal(t, testNum, q.goal)
	require.Equal(t, 1, len(q.requests))

	head, err := q.s.blockState.BestBlockNumber()
	require.NoError(t, err)
	expected := createBlockRequest(head.Int64()+1, uint32(testNum)-uint32(head.Int64()))
	require.Equal(t, &syncRequest{req: expected, to: testPeerID}, q.requests[0])
}

func TestSyncQueue_HandleBlockAnnounce(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()

	testPeerID := peer.ID("noot")
	q.handleBlockAnnounce(testBlockAnnounceMessage, testPeerID)
	require.Equal(t, 1, q.peerScore[testPeerID])
	require.Equal(t, testBlockAnnounceMessage.Number.Int64(), q.goal)
	require.Equal(t, 1, len(q.requests))

	head, err := q.s.blockState.BestBlockNumber()
	require.NoError(t, err)
	expected := createBlockRequest(head.Int64()+1, uint32(testBlockAnnounceMessage.Number.Int64())-uint32(head.Int64()))
	require.Equal(t, &syncRequest{req: expected, to: testPeerID}, q.requests[0])
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
	nodeA.syncQueue.ctx = context.Background()
	time.Sleep(time.Second * 3)

	nodeA.syncQueue.peerScore[nodeB.host.id()] = 1 // expect to try to sync with nodeB first
	go nodeA.syncQueue.processBlockRequests()
	nodeA.syncQueue.requestCh <- &syncRequest{
		req: testBlockRequestMessage,
	}

	time.Sleep(time.Second)
	require.Equal(t, 3, len(nodeA.syncQueue.responses))
	testResp := testBlockResponseMessage()
	require.Equal(t, testResp.BlockData, nodeA.syncQueue.responses)
}
