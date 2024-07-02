// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWorker(t *testing.T) {
	peerA := peer.ID("peerA")
	peerB := peer.ID("peerB")

	ctrl := gomock.NewController(t)
	m := uint32(60)
	blockReq := &network.BlockRequestMessage{
		RequestedData: 1,
		Direction:     3,
		Max:           &m,
	}

	// acquireOrFail is a test channel used to
	// ensure the shared guard is working properly
	// should have the same len as the shared guard
	acquireOrFail := make(chan struct{}, 1)

	reqMaker := NewMockRequestMaker(ctrl)
	// define a mock expectation to peerA
	reqMaker.EXPECT().
		Do(peerA, blockReq, gomock.AssignableToTypeOf((*network.BlockResponseMessage)(nil))).
		DoAndReturn(func(_, _, _ any) any {
			select {
			case acquireOrFail <- struct{}{}:
				defer func() {
					<-acquireOrFail // release once it finishes
				}()
			default:
				t.Errorf("should acquire the channel, othewise the shared guard is not working")
			}
			time.Sleep(2 * time.Second)
			return nil
		}).
		Return(nil)

	// define a mock expectation to peerB
	reqMaker.EXPECT().
		Do(peerB, blockReq, gomock.AssignableToTypeOf((*network.BlockResponseMessage)(nil))).
		DoAndReturn(func(_, _, _ any) any {
			select {
			case acquireOrFail <- struct{}{}:
				defer func() {
					<-acquireOrFail // release once it finishes
				}()
			default:
				t.Errorf("should acquire the channel, othewise the shared guard is not working")
			}
			time.Sleep(2 * time.Second)
			return nil
		}).
		Return(nil)

	sharedGuard := make(chan struct{}, 1)

	// instantiate the workers
	fstWorker := newWorker(peerA, sharedGuard, reqMaker)
	sndWorker := newWorker(peerB, sharedGuard, reqMaker)

	wg := sync.WaitGroup{}
	queue := make(chan *syncTask, 2)

	// run two workers, but they shouldn't work concurrently,
	// because sharedGuard is buffered channel with capacity
	wg.Add(2)
	go fstWorker.run(queue, &wg)
	go sndWorker.run(queue, &wg)

	resultCh := make(chan *syncTaskResult)
	queue <- &syncTask{
		request:  blockReq,
		resultCh: resultCh,
	}
	queue <- &syncTask{
		request:  blockReq,
		resultCh: resultCh,
	}

	// we are waiting 500 ms to guarantee that workers had time to read sync tasks from the queue
	// and send the request. With this assertion we can be sure that even that we start 2 workers
	// only one of them is working and sent a requests
	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))

	var actual []*syncTaskResult
	result := <-resultCh
	actual = append(actual, result)

	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))

	result = <-resultCh
	actual = append(actual, result)

	expected := []*syncTaskResult{
		{who: peerA, request: blockReq, response: new(network.BlockResponseMessage)},
		{who: peerB, request: blockReq, response: new(network.BlockResponseMessage)},
	}

	sort.Slice(actual, func(i, j int) bool {
		return actual[i].who < actual[j].who
	})

	require.Equal(t, expected, actual)

	close(queue)
	wg.Wait()

	require.Equal(t, 0, len(sharedGuard)) // check that workers release lock
}
