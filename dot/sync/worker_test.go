// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWorker(t *testing.T) {
	peerA := peer.ID("peerA")
	ctrl := gomock.NewController(t)
	m := uint32(60)
	blockReq := &network.BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: variadic.Uint32OrHash{},
		Direction:     3,
		Max:           &m,
	}

	reqMaker := NewMockRequestMaker(ctrl)
	reqMaker.EXPECT().
		Do(peerA, blockReq, gomock.AssignableToTypeOf((*network.BlockResponseMessage)(nil))).
		DoAndReturn(func(_, _, _ any) any {
			time.Sleep(2 * time.Second)
			return nil
		}).
		Times(2).
		Return(nil)

	sharedGuard := make(chan struct{}, 1)
	w := newWorker(peerA, sharedGuard, reqMaker)

	wg := sync.WaitGroup{}
	queue := make(chan *syncTask, 1) // 1 means that maximum 1 worker can send request at a given time

	// run two workers, but they shouldn't work concurrently,
	// because sharedGuard is buffered channel with capacity
	wg.Add(2)
	go w.run(queue, &wg)
	go w.run(queue, &wg)

	resultCh := make(chan *syncTaskResult)
	defer close(resultCh)
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
	time.Sleep(10 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))

	actual := <-resultCh
	expected := &syncTaskResult{
		who:      peerA,
		request:  blockReq,
		response: new(network.BlockResponseMessage),
		err:      nil,
	}
	require.Equal(t, expected, actual)

	time.Sleep(10 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))

	actual = <-resultCh
	require.Equal(t, expected, actual)
	close(queue)
	wg.Wait()

	require.Equal(t, 0, len(sharedGuard)) // check that workers release lock
}

type fakeReqMaker struct {
	doFunc func(id peer.ID, req network.Message, resp network.ResponseMessage) error
}

func (f fakeReqMaker) Do(id peer.ID, req network.Message, resp network.ResponseMessage) error {
	return f.doFunc(id, req, resp)
}
