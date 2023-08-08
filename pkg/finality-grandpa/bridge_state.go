// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
)

type waker struct {
	mtx    sync.RWMutex
	wakeCh chan any
}

func newWaker() *waker {
	return &waker{wakeCh: make(chan any, 1000)}
}

func (w *waker) wake() {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	if w.wakeCh == nil {
		return
	}
	go func() {
		select {
		case w.wakeCh <- nil:
		default:
		}
	}()
}

func (w *waker) channel() chan any {
	return w.wakeCh
}

func (w *waker) register(waker *waker) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.wakeCh = waker.wakeCh
}

// round state bridged across rounds.
type bridged[Hash, Number any] struct {
	inner RoundState[Hash, Number]
	// registered map[chan State[Hash, Number]]any
	waker *waker
	sync.RWMutex
}

func (b *bridged[H, N]) update(new RoundState[H, N]) {
	b.Lock()
	b.inner = new
	b.waker.wake()
	b.Unlock()
}

func (b *bridged[H, N]) get(waker *waker) RoundState[H, N] {
	b.RLock()
	defer b.RUnlock()
	b.waker.register(waker)
	return b.inner
}

// A prior view of a round-state.
type priorView[Hash, Number any] struct {
	bridged *bridged[Hash, Number]
}

// Push an update to the latter view.
func (pv *priorView[H, N]) update(new RoundState[H, N]) { //skipcq: RVV-B0001
	pv.bridged.update(new)
}

// A latter view of a round-state.
type latterView[Hash, Number any] struct {
	bridged *bridged[Hash, Number]
}

// Fetch a handle to the last round-state.
func (lv *latterView[H, N]) get(waker *waker) (state RoundState[H, N]) { //skipcq: RVV-B0001
	return lv.bridged.get(waker)
}

// Constructs two views of a bridged round-state.
//
// The prior view is held by a round which produces the state and pushes updates to a latter view.
// When updating, the latter view's task is updated.
//
// The latter view is held by the subsequent round, which blocks certain activity
// while waiting for events on an older round.
func bridgeState[Hash, Number any](initial RoundState[Hash, Number]) (
	priorView[Hash, Number],
	latterView[Hash, Number],
) {
	br := bridged[Hash, Number]{
		inner: initial,
		waker: newWaker(),
	}
	return priorView[Hash, Number]{&br}, latterView[Hash, Number]{&br}
}
