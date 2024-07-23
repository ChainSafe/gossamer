package pvf

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "pvf"), log.SetLevel(log.Debug))

type ValidationHost struct {
	wg     sync.WaitGroup
	stopCh chan struct{}

	workerPool *validationWorkerPool
}

func (v *ValidationHost) Start() {
	fmt.Printf("v.wg %v\n", v)
	v.wg.Add(1)
	logger.Debug("Starting validation host")
	go func() {
		defer v.wg.Done()
	}()
}

func (v *ValidationHost) Stop() {
	close(v.stopCh)
	v.wg.Wait()
}

func NewValidationHost() *ValidationHost {
	return &ValidationHost{
		stopCh:     make(chan struct{}),
		workerPool: newValidationWorkerPool(),
	}
}

func (v *ValidationHost) Validate(msg *ValidationTask) {
	logger.Debugf("Validating worker", "workerID", msg.WorkerID)

	logger.Debugf("submitting request for worker", "workerID", msg.WorkerID)
	hasWorker := v.workerPool.containsWorker(*msg.WorkerID)
	if !hasWorker {
		v.workerPool.newValidationWorker(*msg.WorkerID)
	}
	v.workerPool.submitRequest(msg)
}
