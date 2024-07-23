package pvf

import (
	"sync"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type worker struct {
	workerID parachaintypes.ValidationCodeHash
}

func newWorker(pID parachaintypes.ValidationCodeHash) *worker {
	return &worker{
		workerID: pID,
	}
}

func (w *worker) run(queue chan *ValidationTask, wg *sync.WaitGroup) {
	defer func() {
		logger.Debugf("[STOPPED] worker %x", w.workerID)
		wg.Done()
	}()

	for task := range queue {
		executeRequest(task)
	}
}

func executeRequest(task *ValidationTask) {
	// WIP: This is a dummy implementation of the worker execution for the validation task.  The logic for
	//  validating the parachain block request should be implemented here.
	request := task.PoV
	logger.Debugf("[EXECUTING] worker %x, block request: %s", task.WorkerID, request)
	time.Sleep(500 * time.Millisecond)
	dummyResult := &ValidationResult{}
	task.ResultCh <- &ValidationTaskResult{
		who:    *task.WorkerID,
		result: dummyResult,
	}

	logger.Debugf("[FINISHED] worker %x", task.WorkerID)
}
