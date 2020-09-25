package transaction

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

type Pool struct {
	transactions map[common.Hash]*ValidTransaction
}

func NewPool() *Pool {
	return &Pool{
		transactions: make(map[common.Hash]*ValidTransaction),
	}
}

func (p *Pool) Transactions() []*ValidTransaction {
	txs := make([]*ValidTransaction, len(p.transactions))
	i := 0
	for _, tx := range p.transactions {
		txs[i] = tx
		i++
	}
	return txs
}
