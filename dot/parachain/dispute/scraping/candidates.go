package scraping

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ScrapedCandidate is an item in the ScrapedCandidates btree.
type ScrapedCandidate struct {
	BlockNumber uint32
	Hash        common.Hash
}

// ScrapedCandidateComparator compares two ScrapedCandidates.
func ScrapedCandidateComparator(a, b any) bool {
	return a.(*ScrapedCandidate).BlockNumber < b.(*ScrapedCandidate).BlockNumber
}

// ScrapedCandidates stores the scraped candidates.
type ScrapedCandidates struct {
	Candidates              map[common.Hash]uint32
	CandidatesByBlockNumber scale.BTree
}

// Contains returns true if the ScrapedCandidates contains the given hash.
func (sc *ScrapedCandidates) Contains(hash common.Hash) bool {
	_, ok := sc.Candidates[hash]
	return ok
}

// Insert inserts a new candidate into the ScrapedCandidates.
func (sc *ScrapedCandidates) Insert(blockNumber uint32, hash common.Hash) {
	sc.Candidates[hash]++
	candidate := &ScrapedCandidate{
		BlockNumber: blockNumber,
		Hash:        hash,
	}
	if sc.CandidatesByBlockNumber.Get(candidate) != nil {
		return
	}

	sc.CandidatesByBlockNumber.Set(candidate)
}

// RemoveUptoHeight removes all candidates up to the given block number.
func (sc *ScrapedCandidates) RemoveUptoHeight(blockNumber uint32) []common.Hash {
	var modifiedCandidates []common.Hash

	notStale := scale.NewBTree[ScrapedCandidates](ScrapedCandidateComparator)
	stale := scale.NewBTree[ScrapedCandidates](ScrapedCandidateComparator)

	sc.CandidatesByBlockNumber.Descend(nil, func(i interface{}) bool {
		candidate := i.(*ScrapedCandidate)
		if candidate.BlockNumber < blockNumber {
			stale.Set(i)
		} else {
			notStale.Set(i)
		}
		return true
	})
	sc.CandidatesByBlockNumber = notStale

	stale.Ascend(nil, func(i interface{}) bool {
		candidate := i.(*ScrapedCandidate)
		sc.Candidates[candidate.Hash]--
		if sc.Candidates[candidate.Hash] == 0 {
			delete(sc.Candidates, candidate.Hash)
		}

		modifiedCandidates = append(modifiedCandidates, candidate.Hash)
		return true
	})

	return modifiedCandidates
}

// NewScrapedCandidates creates a new ScrapedCandidates.
func NewScrapedCandidates() ScrapedCandidates {
	return ScrapedCandidates{
		Candidates:              make(map[common.Hash]uint32),
		CandidatesByBlockNumber: scale.NewBTree[ScrapedCandidate](ScrapedCandidateComparator),
	}
}
