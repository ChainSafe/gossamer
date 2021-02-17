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

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

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
