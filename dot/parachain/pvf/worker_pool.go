package pvf

import (
	"crypto/rand"
	"fmt"
	"golang.org/x/exp/maps"
	"math/big"
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

// submitRequest given a request, the worker pool will get the peer given the peer.ID
// parameter or if nil the very first available worker or
// to perform the request, the response will be dispatch in the resultCh.
func (v *validationWorkerPool) submitRequest(request string,
	who *parachaintypes.ValidationCodeHash, resultCh chan<- *validationTaskResult) {

	task := &validationTask{
		request:  request,
		resultCh: resultCh,
	}

	// if the request is bounded to a specific peer then just
	// request it and sent through its queue otherwise send
	// the request in the general queue where all worker are
	// listening on
	v.mtx.RLock()
	defer v.mtx.RUnlock()

	if who != nil {
		syncWorker, inMap := v.workers[*who]
		if inMap {
			if syncWorker == nil {
				panic("sync worker should not be nil")
			}
			syncWorker.queue <- task
			return
		}
	}

	// if the exact peer is not specified then
	// randomly select a worker and assign the
	// task to it, if the amount of workers is
	var selectedWorkerIdx int
	workers := maps.Values(v.workers)
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(workers))))
	if err != nil {
		panic(fmt.Errorf("fail to get a random number: %w", err))
	}
	selectedWorkerIdx = int(nBig.Int64())
	selectedWorker := workers[selectedWorkerIdx]
	selectedWorker.queue <- task
}
