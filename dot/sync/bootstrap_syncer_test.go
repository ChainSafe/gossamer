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

	"github.com/stretchr/testify/require"
)

func newTestBootstrapSyncer(t *testing.T) *bootstrapSyncer {
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(100), types.NewDigest())
	require.NoError(t, err)

	finHeader, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(200), types.NewDigest())
	require.NoError(t, err)

	bs := new(syncmocks.MockBlockState)
	bs.On("BestBlockHeader").Return(header, nil)
	bs.On("GetHighestFinalisedHeader").Return(finHeader, nil)

	return newBootstrapSyncer(bs)
}

func TestBootstrapSyncer_handleWork(t *testing.T) {
	s := newTestBootstrapSyncer(t)

	// peer's state is equal or lower than ours
	// should not create a worker for bootstrap mode
	w, err := s.handleNewPeerState(&peerState{
		number: big.NewInt(100),
	})
	require.NoError(t, err)
	require.Nil(t, w)

	w, err = s.handleNewPeerState(&peerState{
		number: big.NewInt(99),
	})
	require.NoError(t, err)
	require.Nil(t, w)

	// if peer's number is highest, return worker w/ their block as target
	expected := &worker{
		requestData:  bootstrapRequestData,
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(101),
	}
	w, err = s.handleNewPeerState(&peerState{
		number: big.NewInt(101),
		hash:   common.NewHash([]byte{1}),
	})
	require.NoError(t, err)
	require.Equal(t, expected, w)

	expected = &worker{
		requestData:  bootstrapRequestData,
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(9999),
	}
	w, err = s.handleNewPeerState(&peerState{
		number: big.NewInt(9999),
		hash:   common.NewHash([]byte{1}),
	})
	require.NoError(t, err)
	require.Equal(t, expected, w)
}

func TestBootstrapSyncer_handleWorkerResult(t *testing.T) {
	s := newTestBootstrapSyncer(t)

	// if the worker error is nil, then this function should do nothing
	res := &worker{}
	w, err := s.handleWorkerResult(res)
	require.NoError(t, err)
	require.Nil(t, w)

	// if there was a worker error, this should return a worker with
	// startNumber = bestBlockNumber + 1 and the same target as previously
	expected := &worker{
		requestData:  bootstrapRequestData,
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(201),
	}

	res = &worker{
		requestData:  bootstrapRequestData,
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(201),
		err:          &workerError{},
	}

	w, err = s.handleWorkerResult(res)
	require.NoError(t, err)
	require.Equal(t, expected, w)
}

func TestBootstrapSyncer_handleWorkerResult_errUnknownParent(t *testing.T) {
	s := newTestBootstrapSyncer(t)

	// if there was a worker error, this should return a worker with
	// startNumber = bestBlockNumber + 1 and the same target as previously
	expected := &worker{
		requestData:  bootstrapRequestData,
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(200),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(300),
	}

	res := &worker{
		requestData:  bootstrapRequestData,
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(300),
		err: &workerError{
			err: errUnknownParent,
		},
	}

	w, err := s.handleWorkerResult(res)
	require.NoError(t, err)
	require.Equal(t, expected, w)
}
