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
	"math/big"
	"testing"

	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestTipSyncer(t *testing.T) *tipSyncer {
	// header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(100), types.NewDigest())
	// require.NoError(t, err)

	finHeader, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(200), types.NewDigest())
	require.NoError(t, err)

	bs := new(syncmocks.MockBlockState)
	//bs.On("BestBlockHeader").Return(header, nil)
	bs.On("GetHighestFinalisedHeader").Return(finHeader, nil)
	bs.On("HasHeader", mock.AnythingOfType("common.Hash")).Return(true, nil)

	readyBlocks := newBlockQueue(maxResponseSize)
	pendingBlocks := newDisjointBlockSet(pendingBlocksLimit)
	return newTipSyncer(bs, pendingBlocks, readyBlocks, newWorkerState())
}

func TestTipSyncer_handleNewPeerState(t *testing.T) {
	s := newTestTipSyncer(t)

	// peer reports state lower than our highest finalised, we should ignore
	ps := &peerState{
		number: big.NewInt(1),
	}

	w, err := s.handleNewPeerState(ps)
	require.NoError(t, err)
	require.Nil(t, w)

	ps = &peerState{
		number: big.NewInt(201),
		hash:   common.Hash{0xa, 0xb},
	}

	// otherwise, return a worker
	expected := &worker{
		startNumber:  ps.number,
		startHash:    ps.hash,
		targetNumber: ps.number,
		targetHash:   ps.hash,
		requestData:  bootstrapRequestData,
	}

	w, err = s.handleNewPeerState(ps)
	require.NoError(t, err)
	require.Equal(t, expected, w)
}
