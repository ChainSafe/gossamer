// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"sync/atomic"
	"time"
)

type timer struct {
	wakerChan *wakerChan[error]
	mtx       sync.Mutex
	closed    bool // guards the one-shot close of wakerChan.in
	expired   atomic.Bool
}

func newTimer(in <-chan time.Time) *timer {
	inErr := make(chan error)
	wc := newWakerChan(inErr)
	t := timer{wakerChan: wc}
	go t.poll(in)
	return &t
}

func (t *timer) poll(in <-chan time.Time) {
	<-in
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if !t.closed {
		t.wakerChan.in <- nil
		close(t.wakerChan.in)
		t.closed = true
	}
	t.expired.Store(true)
}

func (t *timer) SetWaker(waker *waker) {
	t.wakerChan.setWaker(waker)
}

func (t *timer) Elapsed() (bool, error) {
	return t.expired.Load(), nil
}

func (t *timer) Close() {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if !t.closed {
		close(t.wakerChan.in)
		t.closed = true
	}
}
