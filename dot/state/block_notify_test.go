// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/require"
)

var testMessageTimeout = time.Second * 3

func TestImportChannel(t *testing.T) {
	bs := newTestBlockState(t)
	ch := bs.GetImportedBlockNotifierChannel()

	defer bs.FreeImportedBlockNotifierChannel(ch)

	AddBlocksToState(t, bs, 3, false)

	for i := 0; i < 3; i++ {
		select {
		case <-ch:
		case <-time.After(testMessageTimeout):
			t.Fatal("did not receive imported block")
		}
	}
}

func TestFreeImportedBlockNotifierChannel(t *testing.T) {
	bs := newTestBlockState(t)
	ch := bs.GetImportedBlockNotifierChannel()
	require.Equal(t, 1, len(bs.imported))

	bs.FreeImportedBlockNotifierChannel(ch)
	require.Equal(t, 0, len(bs.imported))
}

func TestFinalizedChannel(t *testing.T) {
	bs := newTestBlockState(t)

	ch := bs.GetFinalisedNotifierChannel()

	defer bs.FreeFinalisedNotifierChannel(ch)

	chain, _ := AddBlocksToState(t, bs, 3, false)

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
	bs := newTestBlockState(t)

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
				require.Equal(t, uint(1), b.Header.Number)
			case <-time.After(testMessageTimeout):
				t.Error("did not receive imported block: ch=", i)
			}
			wg.Done()
		}(i, ch)

	}

	time.Sleep(time.Millisecond * 10)
	AddBlocksToState(t, bs, 1, false)
	wg.Wait()
}

func TestFinalizedChannel_Multi(t *testing.T) {
	bs := newTestBlockState(t)

	num := 5
	chs := make([]chan *types.FinalisationInfo, num)

	for i := 0; i < num; i++ {
		chs[i] = bs.GetFinalisedNotifierChannel()
	}

	chain, _ := AddBlocksToState(t, bs, 1, false)

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

	for _, ch := range chs {
		bs.FreeFinalisedNotifierChannel(ch)
	}
}

func TestService_RegisterUnRegisterRuntimeUpdatedChannel(t *testing.T) {
	bs := newTestBlockState(t)
	ch := make(chan<- runtime.Version)
	chID, err := bs.RegisterRuntimeUpdatedChannel(ch)
	require.NoError(t, err)
	require.NotNil(t, chID)

	res := bs.UnregisterRuntimeUpdatedChannel(chID)
	require.True(t, res)
}

func TestService_RegisterUnRegisterConcurrentCalls(t *testing.T) {
	bs := newTestBlockState(t)

	go func() {
		for i := 0; i < 100; i++ {
			testVer := runtime.Version{
				SpecName:    []byte("mock-spec"),
				SpecVersion: uint32(i),
			}
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
