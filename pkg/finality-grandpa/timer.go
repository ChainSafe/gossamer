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
	closeOnce sync.Once
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
	t.closeOnce.Do(func() {
		t.wakerChan.in <- nil
		close(t.wakerChan.in)
	})
	t.expired.Store(true)
}

func (t *timer) SetWaker(waker *waker) {
	t.wakerChan.setWaker(waker)
}

func (t *timer) Elapsed() (bool, error) {
	return t.expired.Load(), nil
}

func (t *timer) Close() {
	t.closeOnce.Do(func() {
		close(t.wakerChan.in)
	})
}
