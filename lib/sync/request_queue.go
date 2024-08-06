package sync

import (
	"container/list"
	"sync"
)

type requestsQueue[M any] struct {
	mu    sync.RWMutex
	queue *list.List
}

func (r *requestsQueue[M]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.queue.Len()
}

func (r *requestsQueue[M]) PopFront() (value M, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.queue.Front()
	if e == nil {
		return value, false
	}

	r.queue.Remove(e)
	return e.Value.(M), true
}

func (r *requestsQueue[M]) PushBack(message ...M) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range message {
		r.queue.PushBack(m)
	}
}
