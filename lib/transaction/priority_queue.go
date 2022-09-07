// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transaction

import (
	"container/heap"
	"errors"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ErrTransactionExists is returned when trying to add a transaction to the queue that already exists
var ErrTransactionExists = errors.New("transaction is already in queue")

var transactionQueueGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "gossamer_state_transaction",
	Name:      "queue_total",
	Help:      "total number of transactions in ready queue",
})

// An Item is something we manage in a priority queue.
type Item struct {
	data *ValidTransaction
	hash common.Hash

	priority uint64 // The priority of the item in the queue.

	// The order is an monotonically increasing sequence and is used to differentiate between `Item`
	// having the same priority value.
	order uint64

	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds Items.
type priorityQueue []*Item

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// For Item having same priority value we compare them based on their insertion order(FIFO).
	if pq[i].priority == pq[j].priority {
		return pq[i].order < pq[j].order
	}
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// PriorityQueue is a thread safe wrapper over `priorityQueue`
type PriorityQueue struct {
	pq              priorityQueue
	currOrder       uint64
	txs             map[common.Hash]*Item
	nextPushWatcher chan struct{}
	sync.Mutex
}

// NewPriorityQueue creates new instance of PriorityQueue
func NewPriorityQueue() *PriorityQueue {
	spq := &PriorityQueue{
		pq:  make(priorityQueue, 0),
		txs: make(map[common.Hash]*Item),
	}

	heap.Init(&spq.pq)
	return spq
}

// RemoveExtrinsic removes an extrinsic from the queue
func (spq *PriorityQueue) RemoveExtrinsic(ext types.Extrinsic) {
	spq.Lock()
	defer spq.Unlock()

	hash := ext.Hash()
	item, ok := spq.txs[hash]
	if !ok {
		return
	}

	heap.Remove(&spq.pq, item.index)
	delete(spq.txs, hash)
}

// Exists returns true if a hash is in the txs map, false otherwise
func (spq *PriorityQueue) Exists(extHash common.Hash) bool {
	_, ok := spq.txs[extHash]
	return ok
}

// Push inserts a valid transaction with priority p into the queue
func (spq *PriorityQueue) Push(txn *ValidTransaction) (common.Hash, error) {
	spq.Lock()
	defer spq.Unlock()

	hash := txn.Extrinsic.Hash()
	if spq.txs[hash] != nil {
		return hash, ErrTransactionExists
	}

	item := &Item{
		data:     txn,
		hash:     hash,
		order:    spq.currOrder,
		priority: txn.Validity.Priority,
	}
	spq.currOrder++
	heap.Push(&spq.pq, item)
	spq.txs[hash] = item
	close(spq.nextPushWatcher)

	transactionQueueGauge.Set(float64(spq.pq.Len()))
	return hash, nil
}

// NewPushWatcher returns a read only channel to be signalled
// when the next Push() is called on the queue.
// Note the returned channel is closed when the Push() function
// is called.
func (spq *PriorityQueue) NewPushWatcher() (nextPushWatcher <-chan struct{}) {
	spq.Lock()
	defer spq.Unlock()

	var ok bool

	select {
	case _, ok = <-spq.nextPushWatcher:

	default:
	}
	if ok {
		return spq.nextPushWatcher
	}
	nextPushWatcherCh := make(chan struct{})
	spq.nextPushWatcher = nextPushWatcherCh
	return nextPushWatcherCh
}

// Pop removes the transaction with has the highest priority value from the queue and returns it.
// If there are multiple transaction with same priority value then it return them in FIFO order.
func (spq *PriorityQueue) Pop() *ValidTransaction {
	spq.Lock()
	defer spq.Unlock()
	if spq.pq.Len() == 0 {
		return nil
	}

	item := heap.Pop(&spq.pq).(*Item)
	delete(spq.txs, item.hash)

	transactionQueueGauge.Set(float64(spq.pq.Len()))
	return item.data
}

// Peek returns the next item without removing it from the queue
func (spq *PriorityQueue) Peek() *ValidTransaction {
	spq.Lock()
	defer spq.Unlock()
	if spq.pq.Len() == 0 {
		return nil
	}
	return spq.pq[0].data
}

// Pending returns all the transactions currently in the queue
func (spq *PriorityQueue) Pending() []*ValidTransaction {
	spq.Lock()
	defer spq.Unlock()

	var txns []*ValidTransaction
	for idx := 0; idx < spq.pq.Len(); idx++ {
		txns = append(txns, spq.pq[idx].data)
	}
	return txns
}

// Len return the current length of the queue
func (spq *PriorityQueue) Len() int {
	spq.Lock()
	defer spq.Unlock()

	return spq.pq.Len()
}
