// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
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
	go w.start()

	resultCh := make(chan *syncTaskResult)
	defer close(resultCh)

	enqueued := w.processTask(&syncTask{
		resultCh: resultCh,
	})
	require.True(t, enqueued)

	enqueued = w.processTask(&syncTask{
		resultCh: resultCh,
	})
	require.True(t, enqueued)

	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))
	<-resultCh

	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))
	<-resultCh

	w.stop()
}

func TestWorkerAsyncStop(t *testing.T) {
	peerA := peer.ID("peerA")
	ctrl := gomock.NewController(t)

	reqMaker := NewMockRequestMaker(ctrl)
	reqMaker.EXPECT().
		Do(peerA, nil, gomock.AssignableToTypeOf((*network.BlockResponseMessage)(nil))).
		Return(errors.New("mocked error"))

	reqMaker.EXPECT().
		Do(peerA, nil, gomock.AssignableToTypeOf((*network.BlockResponseMessage)(nil))).
		DoAndReturn(func(_, _, _ any) any {
			time.Sleep(2 * time.Second)
			return nil
		}).
		Return(nil)

	sharedGuard := make(chan struct{}, 2)

	w := newWorker(peerA, sharedGuard, reqMaker)
	go w.start()

	doneCh := make(chan struct{})
	resultCh := make(chan *syncTaskResult, 2)
	defer close(resultCh)

	go handleResultsHelper(t, w, resultCh, doneCh)

	// issue two requests in the general channel
	w.processTask(&syncTask{
		resultCh: resultCh,
	})

	w.processTask(&syncTask{
		resultCh: resultCh,
	})

	<-doneCh
}

func handleResultsHelper(t *testing.T, w *worker, resultCh chan *syncTaskResult, doneCh chan<- struct{}) {
	t.Helper()
	defer close(doneCh)

	for r := range resultCh {
		if r.err != nil {
			err := w.stop()
			require.NoError(t, err)
			return
		}
	}
}
