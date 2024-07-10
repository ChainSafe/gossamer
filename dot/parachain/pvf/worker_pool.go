package pvf

import (
	"fmt"
	"sync"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

const (
	maxRequestsAllowed uint = 60
)

type validationWorkerPool struct {
	mtx sync.RWMutex
	wg  sync.WaitGroup

	workers     map[parachaintypes.ValidationCodeHash]*validationWorker
	sharedGuard chan struct{}
}

type validationTask struct {
	request  string
	resultCh chan<- *validationTaskResult
}

type validationTaskResult struct {
	who    parachaintypes.ValidationCodeHash
	result string
}

type validationWorker struct {
	worker *worker
	queue  chan *validationTask
}

func newValidationWorkerPool() *validationWorkerPool {
	return &validationWorkerPool{
		workers:     make(map[parachaintypes.ValidationCodeHash]*validationWorker),
		sharedGuard: make(chan struct{}, maxRequestsAllowed),
	}
}

// stop will shutdown all the available workers goroutines
func (v *validationWorkerPool) stop() error {
	v.mtx.RLock()
	defer v.mtx.RUnlock()

	for _, sw := range v.workers {
		close(sw.queue)
	}

	allWorkersDoneCh := make(chan struct{})
	go func() {
		defer close(allWorkersDoneCh)
		v.wg.Wait()
	}()

	timeoutTimer := time.NewTimer(30 * time.Second)
	select {
	case <-timeoutTimer.C:
		return fmt.Errorf("timeout reached while finishing workers")
	case <-allWorkersDoneCh:
		if !timeoutTimer.Stop() {
			<-timeoutTimer.C
		}

		return nil
	}
}

func (v *validationWorkerPool) newValidationWorker(who parachaintypes.ValidationCodeHash) {

	worker := newWorker(who, v.sharedGuard)
	workerQueue := make(chan *validationTask, maxRequestsAllowed)

	v.wg.Add(1)
	go worker.run(workerQueue, &v.wg)

	v.workers[who] = &validationWorker{
		worker: worker,
		queue:  workerQueue,
	}
	logger.Tracef("potential worker added, total in the pool %d", len(v.workers))
}
