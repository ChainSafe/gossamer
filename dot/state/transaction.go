// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"sync"

	"github.com/ChainSafe/gossamer/dot/telemetry"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// TransactionState represents the queue of transactions
type TransactionState struct {
	queue *transaction.PriorityQueue
	pool  *transaction.Pool

	// notifierChannels are used to notify transaction status. It maps a channel to
	// hex string of the extrinsic it is supposed to notify about.
	notifierChannels map[chan transaction.Status]string
	notifierLock     sync.RWMutex
}

// NewTransactionState returns a new TransactionState
func NewTransactionState() *TransactionState {
	return &TransactionState{
		queue:            transaction.NewPriorityQueue(),
		pool:             transaction.NewPool(),
		notifierChannels: make(map[chan transaction.Status]string),
	}
}

// Push pushes a transaction to the queue, ordered by priority
func (s *TransactionState) Push(vt *transaction.ValidTransaction) (common.Hash, error) {
	s.notifyStatus(vt.Extrinsic, transaction.Ready)
	return s.queue.Push(vt)
}

// Pop removes and returns the head of the queue
func (s *TransactionState) Pop() *transaction.ValidTransaction {
	return s.queue.Pop()
}

// Peek returns the head of the queue without removing it
func (s *TransactionState) Peek() *transaction.ValidTransaction {
	return s.queue.Peek()
}

// Pending returns the current transactions in the queue and pool
func (s *TransactionState) Pending() []*transaction.ValidTransaction {
	return append(s.queue.Pending(), s.pool.Transactions()...)
}

// PendingInPool returns the current transactions in the pool
func (s *TransactionState) PendingInPool() []*transaction.ValidTransaction {
	return s.pool.Transactions()
}

// RemoveExtrinsic removes an extrinsic from the queue and pool
func (s *TransactionState) RemoveExtrinsic(ext types.Extrinsic) {
	s.pool.Remove(ext.Hash())
	s.queue.RemoveExtrinsic(ext)
}

// RemoveExtrinsicFromPool removes an extrinsic from the pool
func (s *TransactionState) RemoveExtrinsicFromPool(ext types.Extrinsic) {
	s.pool.Remove(ext.Hash())
}

// AddToPool adds a transaction to the pool
func (s *TransactionState) AddToPool(vt *transaction.ValidTransaction) common.Hash {
	s.notifyStatus(vt.Extrinsic, transaction.Future)

	hash := s.pool.Insert(vt)

	if err := telemetry.GetInstance().SendMessage(
		telemetry.NewTxpoolImportTM(uint(s.queue.Len()), uint(s.pool.Len())),
	); err != nil {
		logger.Debugf("problem sending txpool.import telemetry message, error: %s", err)
	}

	return hash
}

// GetStatusNotifierChannel creates and returns a status notifier channel.
func (s *TransactionState) GetStatusNotifierChannel(ext types.Extrinsic) chan transaction.Status {
	s.notifierLock.Lock()
	defer s.notifierLock.Unlock()

	ch := make(chan transaction.Status, defaultBufferSize)
	s.notifierChannels[ch] = ext.String()
	return ch
}

// FreeStatusNotifierChannel deletes given status notifier channel from our map.
func (s *TransactionState) FreeStatusNotifierChannel(ch chan transaction.Status) {
	s.notifierLock.Lock()
	defer s.notifierLock.Unlock()

	delete(s.notifierChannels, ch)
}

func (s *TransactionState) notifyStatus(ext types.Extrinsic, status transaction.Status) {
	s.notifierLock.Lock()
	defer s.notifierLock.Unlock()

	if len(s.notifierChannels) == 0 {
		return
	}

	var wg sync.WaitGroup
	for ch, extrinsicStrWithCh := range s.notifierChannels {
		if extrinsicStrWithCh != ext.String() {
			continue
		}
		wg.Add(1)
		go func(ch chan transaction.Status) {
			defer wg.Done()

			select {
			case ch <- status:
			default:
			}
		}(ch)
	}
	wg.Wait()
}
