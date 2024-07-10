package pvf

import (
	"sync"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type worker struct {
	workerID    parachaintypes.ValidationCodeHash
	sharedGuard chan struct{}
}

func newWorker(pID parachaintypes.ValidationCodeHash, sharedGuard chan struct{}) *worker {
	return &worker{
		workerID:    pID,
		sharedGuard: sharedGuard,
	}
}

func (w *worker) run(queue chan *validationTask, wg *sync.WaitGroup) {
	defer func() {
		logger.Debugf("[STOPPED] worker %x", w.workerID)
		wg.Done()
	}()

	for task := range queue {
		executeRequest(w.workerID, task, w.sharedGuard)
	}
}

func executeRequest(who parachaintypes.ValidationCodeHash, task *validationTask, sharedGuard chan struct{}) {
	defer func() {
		<-sharedGuard
	}()

	sharedGuard <- struct{}{}

	request := task.request
	logger.Debugf("[EXECUTING] worker %x, block request: %s", who, request)

	task.resultCh <- &validationTaskResult{
		who:    who,
		result: request + " result",
	}

	logger.Debugf("[FINISHED] worker %x", who)
}
