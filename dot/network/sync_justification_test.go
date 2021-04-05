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
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestSyncQueue_PushResponse_Justification(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
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
		msg.BlockData = append(msg.BlockData, &types.BlockData{
			Hash:          common.Hash{byte(i)},
			Justification: optional.NewBytes(true, []byte{1}),
		})
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
		RandSeed:    1,
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
		msg.BlockData = append(msg.BlockData, &types.BlockData{
			Hash:          common.Hash{byte(i)},
			Justification: optional.NewBytes(false, nil),
		})
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
				Header:        optional.NewHeader(false, nil),
				Body:          optional.NewBody(false, nil),
				Receipt:       optional.NewBytes(false, nil),
				MessageQueue:  optional.NewBytes(false, nil),
				Justification: optional.NewBytes(true, []byte{1}),
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

func TestSyncQueue_finalizeAtHead(t *testing.T) {
	q := newTestSyncQueue(t)
	q.stop()
	time.Sleep(time.Second)
	q.ctx = context.Background()
	q.slotDuration = time.Millisecond * 200

	hash, err := q.s.blockState.GetHashByNumber(big.NewInt(1))
	require.NoError(t, err)

	go q.finalizeAtHead()
	time.Sleep(time.Second)

	data, has := q.justificationRequestData.Load(hash)
	require.True(t, has)
	require.Equal(t, requestData{}, data)

	expected := createBlockRequestWithHash(hash, blockRequestSize)
	expected.RequestedData = RequestedDataJustification

	select {
	case req := <-q.requestCh:
		require.Equal(t, &syncRequest{
			req: expected,
			to:  "",
		}, req)
	case <-time.After(time.Second):
		t.Fatal("did not receive request")
	}
}
