package pvf

import (
	"sync"
	"testing"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

func TestWorker(t *testing.T) {
	workerID1 := parachaintypes.ValidationCodeHash{1, 2, 3, 4}

	w := newWorker(workerID1)

	wg := sync.WaitGroup{}
	queue := make(chan *ValidationTask, 2)

	wg.Add(1)
	go w.run(queue, &wg)

	resultCh := make(chan *ValidationTaskResult)
	defer close(resultCh)

	queue <- &ValidationTask{
		ResultCh: resultCh,
	}

	queue <- &ValidationTask{
		ResultCh: resultCh,
	}

	time.Sleep(500 * time.Millisecond)
	<-resultCh

	time.Sleep(500 * time.Millisecond)
	<-resultCh

	close(queue)
	wg.Wait()
}
