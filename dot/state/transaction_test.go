package state

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/stretchr/testify/require"
)

func TestTransactionState_Pending(t *testing.T) {
	ts := NewTransactionState()

	txs := []*transaction.ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &transaction.Validity{Priority: 1},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &transaction.Validity{Priority: 4},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &transaction.Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &transaction.Validity{Priority: 17},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &transaction.Validity{Priority: 2},
		},
	}

	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		h := ts.AddToPool(tx)
		hashes[i] = h
	}

	pendingPool := ts.PendingInPool()

	sort.Slice(pendingPool, func(i, j int) bool {
		return pendingPool[i].Extrinsic[0] < pendingPool[j].Extrinsic[0]
	})
	require.Equal(t, pendingPool, txs)

	pending := ts.Pending()
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Extrinsic[0] < pending[j].Extrinsic[0]
	})
	require.Equal(t, pending, txs)

	// queue should be empty
	head := ts.Peek()
	require.Nil(t, head)
}

func TestTransactionState_NotifierChannels(t *testing.T) {
	ts := NewTransactionState()

	ext := types.Extrinsic{}
	notifierChannel := ts.GetStatusNotifierChannel(ext)
	defer ts.FreeStatusNotifierChannel(notifierChannel)

	// number of "future" status updates
	var futureCount int
	// number of "ready" status updates
	var readyCount int

	rand.Seed(time.Now().UnixNano())

	// In practice, one won't see ready and future in this order. This is merely
	// meant to check that notifier channels work as expected.
	expectedFutureCount := rand.Intn(10) + 10
	expectedReadyCount := rand.Intn(5) + 5

	dummyTransactions := make([]*transaction.ValidTransaction, expectedFutureCount)

	for i := 0; i < expectedFutureCount; i++ {
		dummyTransactions[i] = &transaction.ValidTransaction{
			Extrinsic: ext,
			Validity:  transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false),
		}

		ts.AddToPool(dummyTransactions[i])
	}

	for i := 0; i < expectedReadyCount; i++ {
		ts.Push(dummyTransactions[i])
	}

	// it takes time for the status updates to happen
	time.Sleep(1 * time.Second)
	close(notifierChannel)

	for status := range notifierChannel {
		if status.String() == transaction.Future.String() {
			futureCount++
		}
		if status.String() == transaction.Ready.String() {
			readyCount++
		}
	}

	require.Equal(t, expectedFutureCount, futureCount)
	require.Equal(t, expectedReadyCount, readyCount)
}
