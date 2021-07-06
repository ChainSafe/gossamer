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
