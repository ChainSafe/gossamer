package sync

import (
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type worker struct {
	peerID      peer.ID
	sharedGuard chan struct{}

	stopCh chan struct{}
	doneCh chan struct{}

	queue          <-chan *syncTask
	exclusiveQueue chan *syncTask

	requestMaker network.RequestMaker
}

func newWorker(pID peer.ID, sharedGuard chan struct{}, queue <-chan *syncTask, network network.RequestMaker) *worker {
	return &worker{
		peerID:       pID,
		sharedGuard:  sharedGuard,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		queue:        queue,
		requestMaker: network,
	}
}

func (w *worker) processTask(task *syncTask) {
	w.exclusiveQueue <- task
}

func (w *worker) start() {
	go func() {
		defer func() {
			w.doneCh <- struct{}{}
		}()

		logger.Debugf("[STARTED] worker %s", w.peerID)
		for {
			select {
			case <-w.stopCh:
				logger.Debugf("[STOPPED] worker %s", w.peerID)
				return
			case task := <-w.queue:
				executeRequest(w.peerID, w.requestMaker, task, w.sharedGuard)
			case task := <-w.exclusiveQueue:
				executeRequest(w.peerID, w.requestMaker, task, w.sharedGuard)
			}
		}
	}()
}

func (w *worker) stop() {
	w.stopCh <- struct{}{}

	timeoutTimer := time.NewTimer(30 * time.Second)
	select {
	case <-w.doneCh:
		if !timeoutTimer.Stop() {
			<-timeoutTimer.C
		}

		return
	case <-timeoutTimer.C:
		logger.Criticalf("timeout while stopping worker %s", w.peerID)
	}
}

func executeRequest(who peer.ID, requestMaker network.RequestMaker,
	task *syncTask, sharedGuard chan struct{}) {
	defer func() {
		<-sharedGuard
	}()

	sharedGuard <- struct{}{}

	request := task.request
	logger.Debugf("[EXECUTING] worker %s, block request: %s", who, request)
	response := new(network.BlockResponseMessage)
	err := requestMaker.Do(who, request, response)

	task.resultCh <- &syncTaskResult{
		who:      who,
		request:  request,
		response: response,
		err:      err,
	}

	logger.Debugf("[FINISHED] worker %s, err: %s, block data amount: %d", who, err, len(response.BlockData))
}
