package scraping

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/tidwall/btree"
)

// ScrappedCandidate is an item in the CandidatesByBlockNumber btree.
type ScrappedCandidate struct {
	BlockNumber uint32
	Hash        common.Hash
}

// ScrappedCandidateComparator compares two ScrappedCandidates.
func ScrappedCandidateComparator(a, b any) bool {
	return a.(ScrappedCandidate).BlockNumber < b.(ScrappedCandidate).BlockNumber
}

// ScrappedCandidates keeps track of the scrapped candidates.
type ScrappedCandidates struct {
	Candidates              map[common.Hash]uint32
	CandidatesByBlockNumber *btree.BTree
}

// Contains returns true if the ScrappedCandidates contains the given hash.
func (sc *ScrappedCandidates) Contains(hash common.Hash) bool {
	_, ok := sc.Candidates[hash]
	return ok
}

// Insert inserts a new candidate into the ScrappedCandidates.
func (sc *ScrappedCandidates) Insert(blockNumber uint32, hash common.Hash) {
	sc.Candidates[hash] = blockNumber
	sc.CandidatesByBlockNumber.Set(&ScrappedCandidate{
		BlockNumber: blockNumber,
		Hash:        hash,
	})
}

// RemoveUptoHeight removes all candidates up to the given block number.
func (sc *ScrappedCandidates) RemoveUptoHeight(blockNumber uint32) []common.Hash {
	var modifiedCandidates []common.Hash

	notStale := btree.New(ScrappedCandidateComparator)
	stale := btree.New(ScrappedCandidateComparator)

	sc.CandidatesByBlockNumber.Descend(nil, func(i interface{}) bool {
		candidate := i.(*ScrappedCandidate)

		if candidate.BlockNumber <= blockNumber {
			stale.Set(i)
		} else {
			notStale.Set(i)
		}
		return true
	})
	sc.CandidatesByBlockNumber = notStale

	stale.Ascend(nil, func(i interface{}) bool {
		candidate := i.(*ScrappedCandidate)
		delete(sc.Candidates, candidate.Hash)
		modifiedCandidates = append(modifiedCandidates, candidate.Hash)
		return true
	})

	return modifiedCandidates
}

// NewScrappedCandidates creates a new ScrappedCandidates.
func NewScrappedCandidates() ScrappedCandidates {
	return ScrappedCandidates{
		Candidates:              make(map[common.Hash]uint32),
		CandidatesByBlockNumber: btree.New(ScrappedCandidateComparator),
	}
}
