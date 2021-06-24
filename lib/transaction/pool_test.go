package transaction

import (
	"sort"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

func TestPool(t *testing.T) {
	tests := []*ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &Validity{Priority: 1},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &Validity{Priority: 4},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &Validity{Priority: 17},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &Validity{Priority: 2},
		},
	}

	p := NewPool()
	hashes := make([]common.Hash, len(tests))
	for i, tx := range tests {
		h := p.Insert(tx)
		hashes[i] = h
	}

	transactions := p.Transactions()
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Extrinsic[0] < transactions[j].Extrinsic[0]
	})
	require.Equal(t, tests, transactions)

	for _, h := range hashes {
		p.Remove(h)
	}
	require.Equal(t, 0, len(p.Transactions()))
}

func TestPoolCollectMetrics(t *testing.T) {
	//reset metric
	ethmetrics.Unregister(readyTransactionsMetrics)

	ethmetrics.Enabled = true
	txmetrics := ethmetrics.GetOrRegisterGauge(readyTransactionsMetrics, nil)

	require.Equal(t, int64(0), txmetrics.Value())

	validtx := []*ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &Validity{Priority: 1},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &Validity{Priority: 4},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &Validity{Priority: 17},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &Validity{Priority: 2},
		},
	}

	h := make([]common.Hash, len(validtx))
	p := NewPool()
	for i, v := range validtx {
		h[i] = p.Insert(v)
	}

	time.Sleep(collectTxMetricsTimeout + time.Second)
	require.Equal(t, int64(len(validtx)), txmetrics.Value())

	p.Remove(h[0])

	time.Sleep(collectTxMetricsTimeout + time.Second)
	require.Equal(t, int64(len(validtx)-1), txmetrics.Value())
}
