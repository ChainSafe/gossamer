package grandpa

import (
	"sync"
)

type Waker struct {
	mtx    sync.RWMutex
	wakeCh chan any
}

func NewWaker() *Waker {
	return &Waker{wakeCh: make(chan any, 1000)}
}

func (w *Waker) Wake() {
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
	return
}

func (w *Waker) Chan() chan any {
	return w.wakeCh
}

func (w *Waker) Register(waker *Waker) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.wakeCh = waker.wakeCh
}

// round state bridged across rounds.
type bridged[Hash, Number any] struct {
	inner RoundState[Hash, Number]
	// registered map[chan State[Hash, Number]]any
	waker *Waker
	sync.RWMutex
}

func newBridged[Hash, Number any](inner RoundState[Hash, Number]) bridged[Hash, Number] {
	return bridged[Hash, Number]{
		inner: inner,
		waker: &Waker{},
	}
}

func (b *bridged[H, N]) update(new RoundState[H, N]) {
	b.Lock()
	b.inner = new
	b.waker.Wake()
	b.Unlock()
}

func (b *bridged[H, N]) get(waker *Waker) RoundState[H, N] {
	b.RLock()
	defer b.RUnlock()
	b.waker.Register(waker)
	return b.inner
}

// A prior view of a round-state.
type PriorView[Hash, Number any] struct {
	*bridged[Hash, Number]
}

// Push an update to the latter view.
func (pv *PriorView[H, N]) Update(new RoundState[H, N]) {
	pv.bridged.update(new)
}

// A latter view of a round-state.
type LatterView[Hash, Number any] struct {
	*bridged[Hash, Number]
}

// // Fetch a handle to the last round-state.
func (lv *LatterView[H, N]) Get(waker *Waker) (state RoundState[H, N]) {
	return lv.get(waker)
}

// Constructs two views of a bridged round-state.
//
// The prior view is held by a round which produces the state and pushes updates to a latter view.
// When updating, the latter view's task is updated.
//
// The latter view is held by the subsequent round, which blocks certain activity
// while waiting for events on an older round.
func BridgeState[Hash, Number any](initial RoundState[Hash, Number]) (PriorView[Hash, Number], LatterView[Hash, Number]) {
	br := bridged[Hash, Number]{
		inner: initial,
		waker: NewWaker(),
	}
	return PriorView[Hash, Number]{&br}, LatterView[Hash, Number]{&br}
}
