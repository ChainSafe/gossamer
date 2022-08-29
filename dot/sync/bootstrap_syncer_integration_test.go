//go:build integration
// +build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func newTestBootstrapSyncer(t *testing.T) *bootstrapSyncer {
	header, err := types.NewHeader(
		common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 100, types.NewDigest())
	require.NoError(t, err)

	finHeader, err := types.NewHeader(
		common.NewHash([]byte{0}), trie.EmptyHash,
		trie.EmptyHash, 200, types.NewDigest())
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	bs := NewMockBlockState(ctrl)
	bs.EXPECT().BestBlockHeader().Return(header, nil).AnyTimes()
	bs.EXPECT().GetHighestFinalisedHeader().Return(finHeader, nil).AnyTimes()

	return newBootstrapSyncer(bs)
}

func TestBootstrapSyncer_handleWork(t *testing.T) {
	s := newTestBootstrapSyncer(t)

	// peer's state is equal or lower than ours
	// should not create a worker for bootstrap mode
	w, err := s.handleNewPeerState(&peerState{
		number: 100,
	})
	require.NoError(t, err)
	require.Nil(t, w)

	w, err = s.handleNewPeerState(&peerState{
		number: 99,
	})
	require.NoError(t, err)
	require.Nil(t, w)

	// if peer's number is highest, return worker w/ their block as target
	expected := &worker{
		requestData:  bootstrapRequestData,
		startNumber:  uintPtr(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: uintPtr(101),
	}
	w, err = s.handleNewPeerState(&peerState{
		number: 101,
		hash:   common.NewHash([]byte{1}),
	})
	require.NoError(t, err)
	require.Equal(t, expected, w)

	expected = &worker{
		requestData:  bootstrapRequestData,
		startNumber:  uintPtr(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: uintPtr(9999),
	}
	w, err = s.handleNewPeerState(&peerState{
		number: 9999,
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
		startNumber:  uintPtr(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: uintPtr(201),
	}

	res = &worker{
		requestData:  bootstrapRequestData,
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: uintPtr(201),
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
		startNumber:  uintPtr(200),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: uintPtr(300),
	}

	res := &worker{
		requestData:  bootstrapRequestData,
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: uintPtr(300),
		err: &workerError{
			err: errUnknownParent,
		},
	}

	w, err := s.handleWorkerResult(res)
	require.NoError(t, err)
	require.Equal(t, expected, w)
}
