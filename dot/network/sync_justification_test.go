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
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestSyncQueue_PushResponse_Justification(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)
	s.syncQueue.stop()
	time.Sleep(time.Second)

	peerID := peer.ID("noot")
	msg := &BlockResponseMessage{
		BlockData: []*types.BlockData{},
	}

	for i := 0; i < int(blockRequestSize); i++ {
		bd := types.NewEmptyBlockData()
		bd.Hash = common.Hash{byte(i)}
		bd.Justification = &[]byte{1}
		msg.BlockData = append(msg.BlockData, bd)
	}

	s.syncQueue.justificationRequestData.Store(common.Hash{byte(0)}, requestData{})
	err := s.syncQueue.pushResponse(msg, peerID)
	require.NoError(t, err)
	require.Equal(t, 1, len(s.syncQueue.responseCh))
	data, ok := s.syncQueue.justificationRequestData.Load(common.Hash{byte(0)})
	require.True(t, ok)
	require.Equal(t, requestData{
		sent:     true,
		received: true,
		from:     peerID,
	}, data)
}

func TestSyncQueue_PushResponse_EmptyJustification(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)
	s.syncQueue.stop()
	time.Sleep(time.Second)

	peerID := peer.ID("noot")
	msg := &BlockResponseMessage{
		BlockData: []*types.BlockData{},
	}

	for i := 0; i < int(blockRequestSize); i++ {
		bd := types.NewEmptyBlockData()
		bd.Hash = common.Hash{byte(i)}
		msg.BlockData = append(msg.BlockData, bd)
	}

	s.syncQueue.justificationRequestData.Store(common.Hash{byte(0)}, &requestData{})
	err := s.syncQueue.pushResponse(msg, peerID)
	require.Equal(t, errEmptyJustificationData, err)
}

func TestSyncQueue_processBlockResponses_Justification(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.ctx = context.Background()

	go func() {
		q.responseCh <- []*types.BlockData{
			{
				Hash:          common.Hash{byte(0)},
				Justification: &[]byte{1},
			},
		}
	}()

	peerID := peer.ID("noot")
	q.justificationRequestData.Store(common.Hash{byte(0)}, requestData{
		from: peerID,
	})

	go q.processBlockResponses()
	time.Sleep(time.Second)

	_, has := q.justificationRequestData.Load(common.Hash{byte(0)})
	require.False(t, has)

	score, ok := q.peerScore.Load(peerID)
	require.True(t, ok)
	require.Equal(t, 2, score)
}
