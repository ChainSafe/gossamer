package state

import (
	tx "github.com/ChainSafe/gossamer/common/transaction"
)

type TransactionQueue struct {
	queue *tx.PriorityQueue
}

func NewTransactionQueue() *TransactionQueue {
	return &TransactionQueue{
		queue: tx.NewPriorityQueue(),
	}
}

func (q *TransactionQueue) Push(vt *tx.ValidTransaction) {
	q.queue.Insert(vt)
}

func (q *TransactionQueue) Pop() *tx.ValidTransaction {
	return q.queue.Pop()
}

func (q *TransactionQueue) Peek() *tx.ValidTransaction {
	return q.queue.Peek()
}
