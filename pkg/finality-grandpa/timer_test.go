// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"testing"
	"time"
)

// TestTimer_ElapsedConcurrentWithFiring exercises the read of `expired` from
// one goroutine while the timer's `poll` goroutine is writing it. With the
// pre-fix code (unsynchronized read in Elapsed) this test trips the race
// detector under `go test -race`.
func TestTimer_ElapsedConcurrentWithFiring(t *testing.T) {
	t.Parallel()
	tick := make(chan time.Time, 1)
	timer := newTimer(tick)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = timer.Elapsed()
			}
		}
	}()

	tick <- time.Now()
	close(tick)

	deadline := time.Now().Add(2 * time.Second)
	for {
		elapsed, _ := timer.Elapsed()
		if elapsed {
			break
		}
		if time.Now().After(deadline) {
			close(stop)
			wg.Wait()
			t.Fatal("timer never reported elapsed after the tick was consumed")
		}
		time.Sleep(time.Millisecond)
	}
	close(stop)
	wg.Wait()
}

// TestTimer_CloseIsIdempotent ensures Close() can be called more than once
// (and after poll has already drained the channel) without panicking — the
// `closed` flag prevents the double-close.
func TestTimer_CloseIsIdempotent(t *testing.T) {
	t.Parallel()
	tick := make(chan time.Time)
	timer := newTimer(tick)

	timer.Close()
	timer.Close()
}

// TestWakerChan_SetWakerConcurrentWithItems exercises the write of `waker`
// from one goroutine while the `start` goroutine is reading it on every item.
// With the pre-fix code (plain *waker field) this trips the race detector.
func TestWakerChan_SetWakerConcurrentWithItems(t *testing.T) {
	t.Parallel()
	in := make(chan int, 100)
	wc := newWakerChan(in)

	w1 := &waker{wakeCh: make(chan struct{}, 1000)}
	w2 := &waker{wakeCh: make(chan struct{}, 1000)}

	// Drain the output channel so start() can keep making progress.
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range wc.channel() {
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			in <- i
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			if i%2 == 0 {
				wc.setWaker(w1)
			} else {
				wc.setWaker(w2)
			}
		}
	}()

	wg.Wait()
	close(in)
	<-drained
}
