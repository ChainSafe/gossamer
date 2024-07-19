package pvf

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
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

func (v *ValidationHost) Validate(workerID parachaintypes.ValidationCodeHash) {
	logger.Debugf("Validating worker", "workerID", workerID)

	resultCh := make(chan *validationTaskResult)

	//task := &validationTask{
	//	request:  "test",
	//	resultCh: resultCh,
	//}
	v.workerPool.submitRequest("test", &workerID, resultCh)

	<-resultCh
}
