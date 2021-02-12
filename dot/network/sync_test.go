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
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

var (
	blockRequestQueueSize int64  = 3
	blockRequestSize      uint32 = 128
)

func createBlockRequests(t *testing.T, start, end int64) []*BlockRequestMessage {
	numReqs := (end - start) / int64(blockRequestSize)
	if numReqs > blockRequestQueueSize {
		numReqs = blockRequestQueueSize
	}

	if end-start < int64(blockRequestSize) {
		numReqs = 1
	}

	reqs := make([]*BlockRequestMessage, numReqs)
	for i := 0; i < int(numReqs); i++ {
		offset := i * int(blockRequestSize)
		reqs[i] = createBlockRequest(t, start+int64(offset), blockRequestSize)
	}
	return reqs
}

func createBlockRequest(t *testing.T, startInt int64, size uint32) *BlockRequestMessage {
	start, err := variadic.NewUint64OrHash(uint64(startInt))
	require.NoError(t, err)

	blockRequest := &BlockRequestMessage{
		RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     0, // ascending
		Max:           optional.NewUint32(true, size),
	}

	return blockRequest
}

func newTestSyncQueue(t *testing.T) *syncQueue {
	s := createTestService(t, nil)
	return s.syncQueue
}

func TestSyncQueue_PushBlockRequest(t *testing.T) {
	q := newTestSyncQueue(t)

	req := createBlockRequest(t, 1, blockRequestSize)
	testPeerID := peer.ID("noot")
	q.pushBlockRequest(req, testPeerID)

	time.Sleep(time.Millisecond * 100)
	var res *syncRequest

	select {
	case res = <-q.requestCh:
	case <-time.After(TestMessageTimeout * 2):
		t.Fatal("test timed out")
	}

	expected := &syncRequest{
		pid: "", // should be empty because it failed to sync with "noot"
		req: req,
	}
	require.Equal(t, expected, res)
}

func TestSyncQueue_PushBlockRequest_ShouldReject(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()

	q.currEnd = 2
	req := createBlockRequest(t, 1, blockRequestSize)
	testPeerID := peer.ID("noot")
	q.pushBlockRequest(req, testPeerID)
	require.Equal(t, 0, len(q.requests))

	q.requests = append(q.requests, &syncRequest{req: req})
	q.pushBlockRequest(req, testPeerID)
	require.Equal(t, 1, len(q.requests))
}

func TestSyncQueue_PushBlockRequests(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()

	reqs := createBlockRequests(t, 1, 1+int64(blockRequestSize)*blockRequestQueueSize)
	testPeerID := peer.ID("noot")
	q.pushBlockRequests(reqs, testPeerID)
	require.Equal(t, int(blockRequestQueueSize), len(q.requests))
}

func TestSyncQueue_PushBlockRequests_ShouldReject(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()

	q.currEnd = 2
	req := createBlockRequest(t, 1, blockRequestSize)
	testPeerID := peer.ID("noot")
	q.pushBlockRequest(req, testPeerID)
	require.Equal(t, 0, len(q.requests))

	q.requests = append(q.requests, &syncRequest{req: req})
	q.pushBlockRequest(req, testPeerID)
	require.Equal(t, 1, len(q.requests))
}
