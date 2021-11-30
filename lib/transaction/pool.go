// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transaction

import (
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Pool represents the transaction pool
type Pool struct {
	transactions map[common.Hash]*ValidTransaction
	mu           sync.RWMutex
}

// NewPool returns a new empty Pool
func NewPool() *Pool {
	return &Pool{
		transactions: make(map[common.Hash]*ValidTransaction),
	}
}

// Get returns a pointer to ValidTransaction or nil given an extinsic hash
func (p *Pool) Get(extHash common.Hash) *ValidTransaction {
	p.mu.Lock()
	defer p.mu.Unlock()

	v, has := p.transactions[extHash]
	if !has {
		return nil
	}

	return v
}

// Transactions returns all the transactions in the pool
func (p *Pool) Transactions() []*ValidTransaction {
	txs := make([]*ValidTransaction, len(p.transactions))
	i := 0

	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, tx := range p.transactions {
		txs[i] = tx
		i++
	}
	return txs
}

// Insert inserts a transaction into the pool
func (p *Pool) Insert(tx *ValidTransaction) common.Hash {
	hash := tx.Extrinsic.Hash()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.transactions[hash] = tx
	return hash
}

// Remove removes a transaction from the pool
func (p *Pool) Remove(hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.transactions, hash)
}

// Len return the current length of the pool
func (p *Pool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.transactions)
}
