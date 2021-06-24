package transaction

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

const collectTxMetricsTimeout = time.Second * 5
const readyTransactionsMetrics = "gossamer/ready/transaction/metrics"

// Pool represents the transaction pool
type Pool struct {
	transactions map[common.Hash]*ValidTransaction
	mu           sync.RWMutex
}

// NewPool returns a new empty Pool
func NewPool() *Pool {
	p := &Pool{
		transactions: make(map[common.Hash]*ValidTransaction),
	}

	go p.collectMetrics()
	return p
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

func (p *Pool) collectMetrics() {
	t := time.NewTicker(collectTxMetricsTimeout)
	defer t.Stop()

	for range t.C {
		p.collect()
	}

}

func (p *Pool) collect() {
	ethmetrics.Enabled = true
	pooltx := ethmetrics.GetOrRegisterGauge(readyTransactionsMetrics, nil)
	pooltx.Update(int64(len(p.transactions)))
}
