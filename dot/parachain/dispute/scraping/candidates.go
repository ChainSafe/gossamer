package scraping

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/google/btree"
)

// ScrappedCandidate is an item in the CandidatesByBlockNumber btree.
type ScrappedCandidate struct {
	BlockNumber uint32
	Hash        common.Hash
}

// Less returns true if the block number of the ScrappedCandidate is less than the block number
// of the other ScrappedCandidate.
func (s ScrappedCandidate) Less(than btree.Item) bool {
	return s.BlockNumber < than.(*ScrappedCandidate).BlockNumber
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
	sc.CandidatesByBlockNumber.ReplaceOrInsert(&ScrappedCandidate{
		BlockNumber: blockNumber,
		Hash:        hash,
	})
}

// RemoveUptoHeight removes all candidates up to the given block number.
func (sc *ScrappedCandidates) RemoveUptoHeight(blockNumber uint32) []common.Hash {
	var modifiedCandidates []common.Hash

	notStale := btree.New(30)
	stale := btree.New(30)

	sc.CandidatesByBlockNumber.Descend(func(i btree.Item) bool {
		candidate := i.(*ScrappedCandidate)

		if candidate.BlockNumber <= blockNumber {
			stale.ReplaceOrInsert(i)
		} else {
			notStale.ReplaceOrInsert(i)
		}
		return true
	})
	sc.CandidatesByBlockNumber = notStale

	stale.Ascend(func(i btree.Item) bool {
		candidate := i.(*ScrappedCandidate)
		delete(sc.Candidates, candidate.Hash)
		modifiedCandidates = append(modifiedCandidates, candidate.Hash)
		return true
	})

	return modifiedCandidates
}

// NewScrappedCandidates creates a new ScrappedCandidates.
func NewScrappedCandidates() *ScrappedCandidates {
	return &ScrappedCandidates{
		Candidates:              make(map[common.Hash]uint32),
		CandidatesByBlockNumber: btree.New(30),
	}
}
