package scraping

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/google/btree"
)

type ScrappedCandidate struct {
	BlockNumber uint32
	Hash        common.Hash
}

func (s ScrappedCandidate) Less(than btree.Item) bool {
	return s.BlockNumber < than.(*ScrappedCandidate).BlockNumber
}

type ScrappedCandidates struct {
	Candidates              map[common.Hash]uint32
	CandidatesByBlockNumber *btree.BTree
}

func (sc *ScrappedCandidates) Contains(hash common.Hash) bool {
	_, ok := sc.Candidates[hash]
	return ok
}

func (sc *ScrappedCandidates) Insert(blockNumber uint32, hash common.Hash) {
	sc.Candidates[hash] = blockNumber
	sc.CandidatesByBlockNumber.ReplaceOrInsert(&ScrappedCandidate{
		BlockNumber: blockNumber,
		Hash:        hash,
	})
}

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

func NewScrappedCandidates() *ScrappedCandidates {
	return &ScrappedCandidates{
		Candidates:              make(map[common.Hash]uint32),
		CandidatesByBlockNumber: btree.New(30),
	}
}
