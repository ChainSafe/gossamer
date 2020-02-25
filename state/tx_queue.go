package state

import (
	tx "github.com/ChainSafe/gossamer/common/transaction"
)

// TransactionQueue represents the queue of transactions
type TransactionQueue struct {
	queue *tx.PriorityQueue
}

// NewTransactionQueue returns a new TransactionQueue
func NewTransactionQueue() *TransactionQueue {
	return &TransactionQueue{
		queue: tx.NewPriorityQueue(),
	}
}

// Push pushes a transaction to the queue, ordered by priority
func (q *TransactionQueue) Push(vt *tx.ValidTransaction) {
	q.queue.Push(vt)
}

// Pop removes and returns the head of the queue
func (q *TransactionQueue) Pop() *tx.ValidTransaction {
	return q.queue.Pop()
}

// Peek returns the head of the queue without removing it
func (q *TransactionQueue) Peek() *tx.ValidTransaction {
	return q.queue.Peek()
}

// Pending returns the current transactions in the queue
func (q *TransactionQueue) Pending() []*tx.ValidTransaction {
	return q.queue.Pending()
}
