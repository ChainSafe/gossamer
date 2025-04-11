package grandpa

import (
	"sync"
	"time"
)

type timer struct {
	wakerChan *wakerChan[error]
	mtx       sync.Mutex
	expired   bool
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
	if t.wakerChan.in != nil {
		t.wakerChan.in <- nil
		close(t.wakerChan.in)
		t.wakerChan.in = nil
	}
	t.expired = true
}

func (t *timer) SetWaker(waker *waker) {
	t.wakerChan.setWaker(waker)
}

func (t *timer) Elapsed() (bool, error) {
	return t.expired, nil
}

func (t *timer) Close() {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if t.wakerChan.in != nil {
		close(t.wakerChan.in)
		t.wakerChan.in = nil
	}
}
