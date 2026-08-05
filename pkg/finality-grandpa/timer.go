// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"sync/atomic"
	"time"
)

// timer reports whether a deadline has passed and wakes whoever is polling it
// when that changes. Rounds create several and discard them as they advance, so
// Close releases one whose round finished before it fired.
type timer struct {
	waker     atomic.Pointer[waker]
	stop      chan struct{}
	closeOnce sync.Once
	expired   atomic.Bool
}

func newTimer(in <-chan time.Time) *timer {
	t := timer{stop: make(chan struct{})}
	go t.poll(in)
	return &t
}

func (t *timer) poll(in <-chan time.Time) {
	select {
	case <-in:
	case <-t.stop:
		return
	}
	// Ordered: waking before expired is set would send the poller back to sleep
	// having seen the timer as still pending.
	t.expired.Store(true)
	if w := t.waker.Load(); w != nil {
		w.wake()
	}
}

func (t *timer) SetWaker(waker *waker) {
	t.waker.Store(waker)
}

func (t *timer) Elapsed() (bool, error) {
	return t.expired.Load(), nil
}

// Close releases a timer that has not fired. Idempotent, and a no-op once the
// timer has elapsed.
func (t *timer) Close() {
	t.closeOnce.Do(func() { close(t.stop) })
}
