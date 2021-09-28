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
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	runtimemocks "github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/stretchr/testify/require"
)

var testMessageTimeout = time.Second * 3

func TestImportChannel(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	ch := bs.GetImportedBlockNotifierChannel()

	defer bs.FreeImportedBlockNotifierChannel(ch)

	AddBlocksToState(t, bs, 3)

	for i := 0; i < 3; i++ {
		select {
		case <-ch:
		case <-time.After(testMessageTimeout):
			t.Fatal("did not receive imported block")
		}
	}
}

func TestFreeImportedBlockNotifierChannel(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	ch := bs.GetImportedBlockNotifierChannel()
	require.Equal(t, 1, len(bs.imported))

	bs.FreeImportedBlockNotifierChannel(ch)
	require.Equal(t, 0, len(bs.imported))
}

func TestFinalizedChannel(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	ch := make(chan *types.FinalisationInfo, 3)
	id, err := bs.RegisterFinalizedChannel(ch)
	require.NoError(t, err)

	defer bs.UnregisterFinalisedChannel(id)

	chain, _ := AddBlocksToState(t, bs, 3)

	for _, b := range chain {
		bs.SetFinalisedHash(b.Hash(), 1, 0)
	}

	for i := 0; i < 1; i++ {
		select {
		case <-ch:
		case <-time.After(testMessageTimeout):
			t.Fatal("did not receive finalised block")
		}
	}
}

func TestImportChannel_Multi(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	num := 5
	chs := make([]chan *types.Block, num)

	for i := 0; i < num; i++ {
		chs[i] = bs.GetImportedBlockNotifierChannel()
	}

	var wg sync.WaitGroup
	wg.Add(num)

	for i, ch := range chs {

		go func(i int, ch <-chan *types.Block) {
			select {
			case b := <-ch:
				require.Equal(t, big.NewInt(1), b.Header.Number)
			case <-time.After(testMessageTimeout):
				t.Error("did not receive imported block: ch=", i)
			}
			wg.Done()
		}(i, ch)

	}

	time.Sleep(time.Millisecond * 10)
	AddBlocksToState(t, bs, 1)
	wg.Wait()

}

func TestFinalizedChannel_Multi(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	num := 5
	chs := make([]chan *types.FinalisationInfo, num)
	ids := make([]byte, num)

	var err error
	for i := 0; i < num; i++ {
		chs[i] = make(chan *types.FinalisationInfo)
		ids[i], err = bs.RegisterFinalizedChannel(chs[i])
		require.NoError(t, err)
	}

	chain, _ := AddBlocksToState(t, bs, 1)

	var wg sync.WaitGroup
	wg.Add(num)

	for i, ch := range chs {

		go func(i int, ch chan *types.FinalisationInfo) {
			select {
			case <-ch:
			case <-time.After(testMessageTimeout):
				t.Error("did not receive finalised block: ch=", i)
			}
			wg.Done()
		}(i, ch)

	}

	time.Sleep(time.Millisecond * 10)
	bs.SetFinalisedHash(chain[0].Hash(), 1, 0)
	wg.Wait()

	for _, id := range ids {
		bs.UnregisterFinalisedChannel(id)
	}
}

func TestService_RegisterUnRegisterRuntimeUpdatedChannel(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)
	ch := make(chan<- runtime.Version)
	chID, err := bs.RegisterRuntimeUpdatedChannel(ch)
	require.NoError(t, err)
	require.NotNil(t, chID)

	res := bs.UnregisterRuntimeUpdatedChannel(chID)
	require.True(t, res)
}

func TestService_RegisterUnRegisterConcurrentCalls(t *testing.T) {
	bs := newTestBlockState(t, testGenesisHeader)

	go func() {
		for i := 0; i < 100; i++ {
			testVer := NewMockVersion(uint32(i))
			go bs.notifyRuntimeUpdated(testVer)
		}
	}()

	for i := 0; i < 100; i++ {
		go func() {

			ch := make(chan<- runtime.Version)
			chID, err := bs.RegisterRuntimeUpdatedChannel(ch)
			require.NoError(t, err)
			unReg := bs.UnregisterRuntimeUpdatedChannel(chID)
			require.True(t, unReg)
		}()
	}
}

// NewMockVersion creates and returns an runtime Version interface mock
func NewMockVersion(specVer uint32) *runtimemocks.MockVersion {
	m := new(runtimemocks.MockVersion)
	m.On("SpecName").Return([]byte(`mock-spec`))
	m.On("ImplName").Return(nil)
	m.On("AuthoringVersion").Return(uint32(0))
	m.On("SpecVersion").Return(specVer)
	m.On("ImplVersion").Return(uint32(0))
	m.On("TransactionVersion").Return(uint32(0))
	m.On("APIItems").Return(nil)
	return m
}
