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

package state

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/stretchr/testify/require"
)

var testMessageTimeout = time.Second * 3

func TestImportChannel(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	ch := make(chan *types.Block)
	id, err := bs.RegisterImportedChannel(ch)
	require.NoError(t, err)

	defer bs.UnregisterImportedChannel(id)

	AddBlocksToState(t, bs, 3)

	for i := 0; i < 3; i++ {
		select {
		case b := <-ch:
			require.Equal(t, big.NewInt(int64(i+1)), b.Header.Number)
		case <-time.After(testMessageTimeout):
			t.Fatal("did not receive finality message")
		}
	}
}

func TestFinalizedChannel(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	ch := make(chan *types.Header, 3)
	id, err := bs.RegisterFinalizedChannel(ch)
	require.NoError(t, err)

	defer bs.UnregisterFinalizedChannel(id)

	chain, _ := AddBlocksToState(t, bs, 3)

	for _, b := range chain {
		bs.SetFinalizedHash(b.Hash(), 0)
	}

	for i := 0; i < 1; i++ {
		select {
		case b := <-ch:
			// ignore genesis block
			if b.Number.Cmp(big.NewInt(0)) == 1 {
				require.Equal(t, big.NewInt(int64(i+1)), b.Number, b)	
			}
		case <-time.After(testMessageTimeout):
			t.Fatal("did not receive finality message")
		}
	}
}