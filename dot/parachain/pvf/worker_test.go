package pvf

import (
	"sync"
	"testing"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
)

func TestWorker(t *testing.T) {
	workerID1 := parachaintypes.ValidationCode{1, 2, 3, 4}

	workerQueue := make(chan *workerTask, maxRequestsAllowed)
	w, err := newWorker(workerID1, workerQueue)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	queue := make(chan *workerTask, 2)

	wg.Add(1)
	go w.run(queue, &wg)

	resultCh := make(chan *ValidationTaskResult)
	defer close(resultCh)

	queue <- &workerTask{
		ResultCh: resultCh,
	}

	queue <- &workerTask{
		ResultCh: resultCh,
	}

	time.Sleep(500 * time.Millisecond)
	<-resultCh

	time.Sleep(500 * time.Millisecond)
	<-resultCh

	close(queue)
	wg.Wait()
}
