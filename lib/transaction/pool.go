package transaction

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"sync"
)

type Pool struct {
	transactions map[common.Hash]*ValidTransaction
	mu           sync.RWMutex
}

func NewPool() *Pool {
	return &Pool{
		transactions: make(map[common.Hash]*ValidTransaction),
	}
}

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

func (p *Pool) Hashes() []common.Hash {
	txs := make([]common.Hash, len(p.transactions))
	i := 0

	p.mu.RLock()
	defer p.mu.RUnlock()

	for h := range p.transactions {
		txs[i] = h
		i++
	}
	return txs
}

func (p *Pool) Insert(tx *ValidTransaction) common.Hash {
	hash := tx.Extrinsic.Hash()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.transactions[hash] = tx
	return hash
}

func (p *Pool) Remove(hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.transactions, hash)
}
