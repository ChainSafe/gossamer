package pvf

import (
	"sync"
	"testing"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
)

func TestWorker(t *testing.T) {
	workerID1 := parachaintypes.ValidationCodeHash{1, 2, 3, 4}

	sharedGuard := make(chan struct{}, 1)
	w := newWorker(workerID1, sharedGuard)

	wg := sync.WaitGroup{}
	queue := make(chan *validationTask, 2)

	wg.Add(1)
	go w.run(queue, &wg)

	resultCh := make(chan *validationTaskResult)
	defer close(resultCh)

	queue <- &validationTask{
		resultCh: resultCh,
	}

	queue <- &validationTask{
		resultCh: resultCh,
	}

	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))
	<-resultCh

	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, len(sharedGuard))
	<-resultCh

	close(queue)
	wg.Wait()
}
