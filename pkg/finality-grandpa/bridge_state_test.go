package grandpa

import (
	"sync"
	"testing"
	"time"
)

func TestBridgeState(t *testing.T) {
	initial := RoundState[string, int32]{}

	prior, latter := BridgeState(initial)

	barrier := make(chan any)
	var wg sync.WaitGroup

	waker := &Waker{
		wakeCh: make(chan any),
	}

	var waitForFinality = func() bool {
		if latter.Get(waker).Finalized != nil {
			return true
		} else {
			return false
		}
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		<-barrier
		time.Sleep(5 * time.Millisecond)
		prior.Update(RoundState[string, int32]{
			PrevoteGHOST: &HashNumber[string, int32]{"5", 5},
			Finalized:    &HashNumber[string, int32]{"1", 1},
			Estimate:     &HashNumber[string, int32]{"3", 3},
			Completable:  true,
		})
	}()

	// block_on
	go func() {
		defer wg.Done()
		<-barrier
		if waitForFinality() {
			return
		}
		for range waker.wakeCh {
			if waitForFinality() {
				return
			}
		}
	}()

	close(barrier)
	wg.Wait()
}
