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
	"testing"

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
	basePath := utils.NewTestBasePath(t, "node_a")
	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	var (
		s      = createTestService(t, config)
		peerID = peer.ID("rudolf")
		msg    = &BlockRequestMessage{}
	)

	s.syncQueue.beginSyncingWithPeer(peerID, msg)
	require.True(t, s.host.h.ConnManager().IsProtected(peerID, ""))
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
	}, br)
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
