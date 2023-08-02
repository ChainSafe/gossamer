package sync

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestWorkerStop(t *testing.T) {
	peerA := peer.ID("peerA")
	ctrl := gomock.NewController(t)

	reqMaker := NewMockRequestMaker(ctrl)
	reqMaker.EXPECT().
		Do(peerA, nil, gomock.AssignableToTypeOf((*network.BlockResponseMessage)(nil))).
		Return(nil)

	sharedGuard := make(chan struct{}, 1)
	generalQueue := make(chan *syncTask)

	w := newWorker(peerA, sharedGuard, reqMaker)
	w.start()

	resultCh := make(chan *syncTaskResult)
	defer close(resultCh)

	generalQueue <- &syncTask{
		resultCh: resultCh,
	}

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
		Return(nil)

	sharedGuard := make(chan struct{}, 2)

	w := newWorker(peerA, sharedGuard, reqMaker)
	w.start()

	doneCh := make(chan struct{})
	resultCh := make(chan *syncTaskResult, 2)
	go handleResultsHelper(t, w, resultCh, doneCh)

	// issue two requests in the general channel
	w.processTask(&syncTask{
		resultCh: resultCh,
	})

	w.processTask(&syncTask{
		resultCh: resultCh,
	})

	close(resultCh)
	<-doneCh
}

func handleResultsHelper(t *testing.T, w *worker, resultCh chan *syncTaskResult, doneCh chan<- struct{}) {
	t.Helper()
	defer close(doneCh)

	for r := range resultCh {
		if r.err != nil {
			fmt.Printf("==> %s\n", r.err)
			w.stop()
		}
	}
}
