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
	q.queue.Push(vt)
}

func (q *TransactionQueue) Pop() *tx.ValidTransaction {
	return q.queue.Pop()
}

func (q *TransactionQueue) Peek() *tx.ValidTransaction {
	return q.queue.Peek()
}

func (q *TransactionQueue) Pending() ([][]byte, error) {
	txs := q.queue.Pending()
	pending := [][]byte{}
	for _, tx := range txs {
		enc, err := tx.Encode()
		if err != nil {
			return nil, err
		}
		pending = append(pending, enc)
	}
	return pending, nil
}
