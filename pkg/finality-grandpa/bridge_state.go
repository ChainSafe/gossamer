// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
)

type waker struct {
	sync.RWMutex
	wakeCh chan any
}

func newWaker() *waker {
	return &waker{wakeCh: make(chan any, 1000)}
}

func (w *waker) wake() {
	w.RLock()
	defer w.RUnlock()
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

func (w *waker) register(waker *waker) { //nolint:unused //TODO: upgrading golang-ci-lint will fix this
	w.Lock()
	defer w.Unlock()
	w.wakeCh = waker.wakeCh
}

// round state bridged across rounds.
type bridged[Hash, Number any] struct {
	inner RoundState[Hash, Number]
	// registered map[chan State[Hash, Number]]any
	waker *waker
	sync.RWMutex
}

func (b *bridged[H, N]) update(new RoundState[H, N]) { //nolint:unused
	b.Lock()
	defer b.Unlock()
	b.inner = new
	b.waker.wake()

}

func (b *bridged[H, N]) get(waker *waker) RoundState[H, N] { //nolint:unused
	b.RLock()
	defer b.RUnlock()
	b.waker.register(waker)
	return b.inner
}

// A prior view of a round-state.
type priorView[Hash, Number any] interface {
	update(new RoundState[Hash, Number])
}

// A latter view of a round-state.
type latterView[Hash, Number any] interface {
	get(waker *waker) (state RoundState[Hash, Number])
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
	return &br, &br
}
