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
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(100), types.Digest{})
	require.NoError(t, err)

	bs := new(syncmocks.MockBlockState)
	bs.On("BestBlockHeader").Return(header, nil)

	return newBootstrapSyncer(bs)
}

func TestBootstrapSyncer_handleWork(t *testing.T) {
	s := newTestBootstrapSyncer(t)

	// peer's state is equal or lower than ours
	// should not create a worker for bootstrap mode
	w, err := s.handleWork(&peerState{
		number: big.NewInt(100),
	})
	require.NoError(t, err)
	require.Nil(t, w)

	w, err = s.handleWork(&peerState{
		number: big.NewInt(99),
	})
	require.NoError(t, err)
	require.Nil(t, w)

	// if peer's number is highest, return worker w/ their block as target
	expected := &worker{
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(101),
	}
	w, err = s.handleWork(&peerState{
		number: big.NewInt(101),
		hash:   common.NewHash([]byte{1}),
	})
	require.NoError(t, err)
	require.Equal(t, expected, w)

	expected = &worker{
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(9999),
	}
	w, err = s.handleWork(&peerState{
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
		startHash:    common.EmptyHash,
		startNumber:  big.NewInt(101),
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(201),
	}

	res = &worker{
		targetHash:   common.NewHash([]byte{1}),
		targetNumber: big.NewInt(201),
		err:          &workerError{},
	}

	w, err = s.handleWorkerResult(res)
	require.NoError(t, err)
	require.Equal(t, expected, w)
}
