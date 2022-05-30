// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"runtime"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"

	"github.com/ChainSafe/chaindb"
	"github.com/stretchr/testify/require"
)

func TestConcurrencySetHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	threads := runtime.NumCPU()
	dbs := make([]chaindb.Database, threads)
	for i := 0; i < threads; i++ {
		dbs[i] = NewInMemoryDB(t)
	}

	tries := newTriesEmpty()

	pend := new(sync.WaitGroup)
	pend.Add(threads)
	for i := 0; i < threads; i++ {
		go func(index int) {
			defer pend.Done()

			bs, err := NewBlockStateFromGenesis(dbs[index], tries, testGenesisHeader, telemetryMock)
			require.NoError(t, err)

			header := &types.Header{
				Number:    1,
				StateRoot: trie.EmptyHash,
				Digest:    types.NewDigest(),
			}

			err = bs.SetHeader(header)
			require.NoError(t, err)

			res, err := bs.GetHeader(header.Hash())
			require.NoError(t, err)
			require.Equal(t, header, res)

		}(i)
	}
	pend.Wait()
}
