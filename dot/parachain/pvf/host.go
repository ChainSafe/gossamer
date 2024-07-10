package pvf

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "pvf"), log.SetLevel(log.Debug))

type validationHost struct {
	wg     sync.WaitGroup
	stopCh chan struct{}

	workerPool *validationWorkerPool
}

func (v *validationHost) start() {

	v.wg.Add(1)
	logger.Debug("Starting validation host")
	go func() {
		defer v.wg.Done()
	}()
}

func (v *validationHost) stop() {
	close(v.stopCh)
	v.wg.Wait()
}
