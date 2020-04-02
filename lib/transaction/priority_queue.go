// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package transaction

import (
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
)

// PriorityQueue implements a priority queue using a double linked list
type PriorityQueue struct {
	head  *node
	mutex sync.Mutex
	pool  Pool
}

type node struct {
	data   *ValidTransaction
	parent *node
	child  *node
	hash   common.Hash
}

// NewPriorityQueue creates new instance of PriorityQueue
func NewPriorityQueue() *PriorityQueue {
	pq := PriorityQueue{
		head:  nil,
		mutex: sync.Mutex{},
		pool:  make(map[common.Hash]*ValidTransaction),
	}

	return &pq
}

// Pop removes the head of the queue and returns it
func (q *PriorityQueue) Pop() *ValidTransaction {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.head == nil {
		return nil
	}
	head := q.head
	q.head = head.child

	delete(q.pool, head.hash)

	return head.data
}

// Peek returns the next item without removing it from the queue
func (q *PriorityQueue) Peek() *ValidTransaction {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.head == nil {
		return nil
	}
	return q.head.data
}

// Pool returns the queue's underlying transaction pool
func (q *PriorityQueue) Pool() map[common.Hash]*ValidTransaction {
	return q.pool
}

// Pending returns all the transactions currently in the queue
func (q *PriorityQueue) Pending() []*ValidTransaction {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	txs := []*ValidTransaction{}

	curr := q.head
	for {
		if curr == nil {
			return txs
		}

		txs = append(txs, curr.data)
		curr = curr.child
	}
}

// Push traverses the list and places a valid transaction with priority p directly before the
// first node with priority p-1. If there are other nodes with priority p, the new node is placed
// behind them.
func (q *PriorityQueue) Push(vt *ValidTransaction) (common.Hash, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	curr := q.head

	hash, err := vt.Hash()
	if err != nil {
		return common.Hash{}, err
	}

	if curr == nil {
		q.head = &node{data: vt, hash: hash}
		q.pool[hash] = vt
		return hash, nil
	}

	for ; curr != nil; curr = curr.child {
		currPriority := curr.data.Validity.Priority
		if vt.Validity.Priority > currPriority {
			newNode := &node{
				data:   vt,
				parent: curr.parent,
				child:  curr,
				hash:   hash,
			}

			if curr.parent == nil {
				q.head = newNode
			} else {
				curr.parent.child = newNode
			}
			curr.parent = newNode

			q.pool[hash] = vt
			return hash, nil
		} else if curr.child == nil {
			newNode := &node{
				data:   vt,
				parent: curr,
				hash:   hash,
			}
			curr.child = newNode

			q.pool[hash] = vt
			return hash, nil
		}
	}

	return common.Hash{}, nil
}
