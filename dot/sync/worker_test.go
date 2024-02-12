// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
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
	ctrl := gomock.NewController(t)

	reqMaker := NewMockRequestMaker(ctrl)
	reqMaker.EXPECT().
		Do(peerA, nil, gomock.AssignableToTypeOf((*network.BlockResponseMessage)(nil))).
		DoAndReturn(func(_, _, _ any) any {
			time.Sleep(2 * time.Second)
			return nil
		}).
		Times(2).
		Return(nil)

	sharedGuard := make(chan struct{}, 1)
	w := newWorker(peerA, sharedGuard, reqMaker)

	wg := sync.WaitGroup{}
	queue := make(chan *syncTask, 2)

	wg.Add(1)
	go w.run(queue, &wg)

	resultCh := make(chan *syncTaskResult)
	defer close(resultCh)

	queue <- &syncTask{
		resultCh: resultCh,
	}

	queue <- &syncTask{
		resultCh: resultCh,
	}

	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))
	<-resultCh

	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))
	<-resultCh

	close(queue)
	wg.Wait()
}
