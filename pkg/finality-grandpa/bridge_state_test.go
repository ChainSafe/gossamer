// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"testing"
	"time"
)

func TestBridgeState(_ *testing.T) {
	initial := RoundState[string, int32]{}

	prior, latter := BridgeState(initial)

	barrier := make(chan any)
	var wg sync.WaitGroup

	waker := &waker{
		wakeCh: make(chan any),
	}

	var waitForFinality = func() bool {
		return latter.get(waker).Finalized != nil
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		<-barrier
		time.Sleep(5 * time.Millisecond)
		prior.update(RoundState[string, int32]{
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
