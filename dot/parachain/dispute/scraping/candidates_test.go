package scraping

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

func TestScrapedCandidates(t *testing.T) {
	sc := NewScrapedCandidates()

	// Test inserting candidates and checking if they exist in the ScrapedCandidates.
	sc.Insert(1, common.NewHash([]byte{1}))
	sc.Insert(2, common.NewHash([]byte{2}))
	sc.Insert(3, common.NewHash([]byte{3}))
	sc.Insert(4, common.NewHash([]byte{4}))

	require.True(t, sc.Contains(common.NewHash([]byte{1})))
	require.True(t, sc.Contains(common.NewHash([]byte{2})))
	require.True(t, sc.Contains(common.NewHash([]byte{3})))
	require.True(t, sc.Contains(common.NewHash([]byte{4})))
	require.False(t, sc.Contains(common.NewHash([]byte{5}))) // Non-existing candidate.

	// Test removing candidates up to a certain height and check if they are removed.
	modifiedCandidates := sc.RemoveUptoHeight(3)
	require.ElementsMatch(t, []common.Hash{common.NewHash([]byte{1}), common.NewHash([]byte{2})}, modifiedCandidates)

	// Check if candidates are removed after the removal operation.
	require.False(t, sc.Contains(common.NewHash([]byte{1})))
	require.False(t, sc.Contains(common.NewHash([]byte{2})))
	require.True(t, sc.Contains(common.NewHash([]byte{3})))
	require.True(t, sc.Contains(common.NewHash([]byte{4})))

	// Test inserting new candidates after the removal and check if they exist.
	sc.Insert(5, common.NewHash([]byte{5}))
	require.False(t, sc.Contains(common.NewHash([]byte{1}))) // Should still be removed.
	require.True(t, sc.Contains(common.NewHash([]byte{5})))  // Newly inserted candidate.

	// Test edge case: Removing candidates with an empty CandidatesByBlockNumber.
	emptySc := NewScrapedCandidates()
	modifiedCandidates = emptySc.RemoveUptoHeight(1)
	require.Empty(t, modifiedCandidates)

	// Test edge case: Removing candidates when CandidatesByBlockNumber is nil.
	sc2 := NewScrapedCandidates()
	sc2.Insert(1, common.NewHash([]byte{1}))
	modifiedCandidates = sc2.RemoveUptoHeight(2)
	require.ElementsMatch(t, []common.Hash{common.NewHash([]byte{1})}, modifiedCandidates)

	// Test edge case: RemoveUptoHeight with blockNumber greater than all candidates.
	sc3 := &ScrapedCandidates{
		Candidates:              make(map[common.Hash]uint32),
		CandidatesByBlockNumber: btree.New(ScrapedCandidateComparator),
	}
	sc3.Insert(1, common.NewHash([]byte{1}))
	modifiedCandidates = sc3.RemoveUptoHeight(100)
	require.ElementsMatch(t, []common.Hash{common.NewHash([]byte{1})}, modifiedCandidates)

	// Test edge case: RemoveUptoHeight with empty Candidates map.
	sc4 := NewScrapedCandidates()
	modifiedCandidates = sc4.RemoveUptoHeight(1)
	require.Empty(t, modifiedCandidates)
}
