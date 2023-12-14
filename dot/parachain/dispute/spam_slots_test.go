package dispute

import (
	"sync"
	"testing"

	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/stretchr/testify/require"
)

func TestSpamSlots_AddUnconfirmed(t *testing.T) {
	t.Parallel()
	// with 3 slots, we can add 3 votes for the same validator
	ss := NewSpamSlots(3)

	// add 3 votes for validator 1
	require.True(t, ss.AddUnconfirmed(1, common.Hash{1}, 1))
	require.True(t, ss.AddUnconfirmed(1, common.Hash{2}, 1))
	require.True(t, ss.AddUnconfirmed(1, common.Hash{3}, 1))

	// for the 4th vote, the slot must be full
	require.False(t, ss.AddUnconfirmed(1, common.Hash{}, 1))
}

func TestSpamSlots_Clear(t *testing.T) {
	t.Parallel()

	ss := NewSpamSlots(3)

	// add 3 votes for validator 1
	require.True(t, ss.AddUnconfirmed(1, common.Hash{1}, 1))
	require.True(t, ss.AddUnconfirmed(1, common.Hash{2}, 1))
	require.True(t, ss.AddUnconfirmed(1, common.Hash{3}, 1))

	// the slot must be full
	require.False(t, ss.AddUnconfirmed(1, common.Hash{0}, 1))

	// clear the votes for session 1 and candidate 1
	ss.Clear(1, common.Hash{1})

	// now we can add another vote
	require.True(t, ss.AddUnconfirmed(1, common.Hash{1}, 1))
}

func TestSpamSlots_PruneOld(t *testing.T) {
	t.Parallel()

	ss := NewSpamSlots(3)

	// add 3 votes for validator 1 session 1
	require.True(t, ss.AddUnconfirmed(1, common.Hash{1}, 1))
	require.True(t, ss.AddUnconfirmed(1, common.Hash{2}, 1))
	require.True(t, ss.AddUnconfirmed(1, common.Hash{3}, 1))

	// add 3 votes for validator 1 session 2
	require.True(t, ss.AddUnconfirmed(2, common.Hash{1}, 1))
	require.True(t, ss.AddUnconfirmed(2, common.Hash{2}, 1))
	require.True(t, ss.AddUnconfirmed(2, common.Hash{3}, 1))

	// add 3 votes for validator 1 session 3
	require.True(t, ss.AddUnconfirmed(3, common.Hash{1}, 1))
	require.True(t, ss.AddUnconfirmed(3, common.Hash{2}, 1))
	require.True(t, ss.AddUnconfirmed(3, common.Hash{3}, 1))

	// the validator shouldn't be able to vote for session 1, 2 and 3 anymore
	require.False(t, ss.AddUnconfirmed(1, common.Hash{0}, 1))
	require.False(t, ss.AddUnconfirmed(2, common.Hash{0}, 1))
	require.False(t, ss.AddUnconfirmed(3, common.Hash{0}, 1))

	// prune old sessions
	ss.PruneOld(3)

	// the validator should be able to vote for session 1 and 2 but not 3
	require.True(t, ss.AddUnconfirmed(1, common.Hash{0}, 1))
	require.True(t, ss.AddUnconfirmed(2, common.Hash{0}, 1))
	require.False(t, ss.AddUnconfirmed(3, common.Hash{0}, 1))
}

func TestSpamSlots_Concurrency(t *testing.T) {
	t.Parallel()

	const maxSpamVotes = 5000
	const numAdd = 100000
	const numClear = 100000
	const numPrune = 100000

	spam := NewSpamSlots(maxSpamVotes)
	var wg sync.WaitGroup

	// Concurrent add operations
	for i := 0; i < numAdd; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = spam.AddUnconfirmed(1, common.Hash{1}, 1)
		}()
	}

	// Concurrent clear operations
	for i := 0; i < numClear; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spam.Clear(1, common.Hash{1})
		}()
	}

	// Concurrent prune operations
	for i := 0; i < numPrune; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spam.PruneOld(1)
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

func TestNewSpamSlotsFromState(t *testing.T) {
	t.Parallel()

	// with
	unconfirmedDisputes := make(map[unconfirmedKey]*treeset.Set)
	unconfirmedDisputes[unconfirmedKey{session: 1, candidate: common.Hash{1}}] = treeset.NewWith(byIndex, 1, 2, 3)
	unconfirmedDisputes[unconfirmedKey{session: 1, candidate: common.Hash{2}}] = treeset.NewWith(byIndex, 1, 2, 3)
	unconfirmedDisputes[unconfirmedKey{session: 1, candidate: common.Hash{3}}] = treeset.NewWith(byIndex, 1, 2, 3)

	// when
	ss := NewSpamSlotsFromState(unconfirmedDisputes, 3)

	// then
	require.False(t, ss.AddUnconfirmed(1, common.Hash{1}, 1))
}

func byIndex(a, b interface{}) int {
	return a.(int) - b.(int)
}

func BenchmarkSpamSlots_AddUnconfirmed(b *testing.B) {
	ss := NewSpamSlots(MaxSpamVotes)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		require.True(b, ss.AddUnconfirmed(1, common.Hash{0}, parachainTypes.ValidatorIndex(b.N%10)))
	}
}

func BenchmarkSpamSlots_Clear(b *testing.B) {
	ss := NewSpamSlots(uint32(b.N))
	for n := 0; n < b.N; n++ {
		require.True(b, ss.AddUnconfirmed(1, common.Hash{0}, parachainTypes.ValidatorIndex(b.N)))
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ss.Clear(1, common.Hash{0})
	}
}

func BenchmarkSpamSlots_PruneOld(b *testing.B) {
	ss := NewSpamSlots(uint32(b.N))
	for n := 0; n < b.N; n++ {
		require.True(b, ss.AddUnconfirmed(parachainTypes.SessionIndex(n), common.Hash{0}, parachainTypes.ValidatorIndex(b.N)))
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ss.PruneOld(parachainTypes.SessionIndex(n))
	}
}

func BenchmarkNewSpamSlotsFromState(b *testing.B) {
	unconfirmedDisputes := make(map[unconfirmedKey]*treeset.Set)
	for n := 0; n < b.N; n++ {
		unconfirmedDisputes[unconfirmedKey{session: 1, candidate: common.Hash{byte(n)}}] = treeset.NewWith(byIndex, 1, 2, 3)
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		NewSpamSlotsFromState(unconfirmedDisputes, MaxSpamVotes)
	}
}
