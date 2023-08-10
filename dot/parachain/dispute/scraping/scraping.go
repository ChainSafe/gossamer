package scraping

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
)

// DisputeCandidateLifetimeAfterFinalization How many blocks after finalisation an information about
// backed/included candidate should be pre-loaded (when scraping onchain votes) and kept locally (when pruning).
const DisputeCandidateLifetimeAfterFinalization = uint32(10)

// ChainScrapper Scrapes non finalized chain in order to collect information from blocks
type ChainScrapper struct {
	// All candidates we have seen included, which not yet have been finalized.
	IncludedCandidates ScrappedCandidates
	// All candidates we have seen backed
	BackedCandidates ScrappedCandidates
	// Latest relay blocks observed by the provider.
	LastObservedBlocks lrucache.LRUCache[common.Hash, uint32]
	// Maps included candidate hashes to one or more relay block heights and hashes.
	Inclusions Inclusions
}

// IsCandidateIncluded Check whether we have seen a candidate included on any chain.
func (cs *ChainScrapper) IsCandidateIncluded(candidateHash common.Hash) bool {
	return cs.IncludedCandidates.Contains(candidateHash)
}

// IsCandidateBacked Check whether the candidate is backed
func (cs *ChainScrapper) IsCandidateBacked(candidateHash common.Hash) bool {
	return cs.BackedCandidates.Contains(candidateHash)
}

func (cs *ChainScrapper) GetBlocksIncludingCandidate(candidateHash common.Hash) []Inclusion {
	return cs.Inclusions.Get(candidateHash)
}

func (cs *ChainScrapper) ProcessActiveLeavesUpdate(
	sender overseer.Sender,
	update overseer.ActiveLeavesUpdate,
) (*types.ScrappedUpdates, error) {

	panic("ChainScrapper.ProcessActiveLeavesUpdate not implemented")
}

// ProcessFinalisedBlock prune finalised candidates.
func (cs *ChainScrapper) ProcessFinalisedBlock(finalisedBlock uint32) {
	if finalisedBlock < DisputeCandidateLifetimeAfterFinalization-1 {
		// Nothing to prune. We are still in the beginning of the chain and there are not
		// enough finalized blocks yet.
		return
	}

	removeUptoHeight := finalisedBlock - (DisputeCandidateLifetimeAfterFinalization - 1)
	cs.BackedCandidates.RemoveUptoHeight(removeUptoHeight)
	candidatesModified := cs.IncludedCandidates.RemoveUptoHeight(removeUptoHeight)
	cs.Inclusions.RemoveUpToHeight(removeUptoHeight, candidatesModified)
}

func (cs *ChainScrapper) ProcessCandidateEvents(
	sender overseer.Sender,
	blockNumber uint32, blockHash common.Hash,
) ([]parachainTypes.CandidateReceipt, error) {

	panic("ChainScrapper.ProcessCandidateEvents not implemented")
}

func (cs *ChainScrapper) GetRelevantBlockAncestors(
	sender overseer.Sender,
	head common.Hash,
	headNumber uint32,
) ([]common.Hash, error) {
	panic("ChainScrapper.GetRelevantBlockAncestors not implemented")
}

func NewChainScraper(
	sender overseer.Sender,
	initialHead overseer.ActivatedLeaf,
) (*ChainScrapper, *types.ScrappedUpdates, error) {
	chainScraper := &ChainScrapper{
		IncludedCandidates: NewScrappedCandidates(),
		BackedCandidates:   NewScrappedCandidates(),
		LastObservedBlocks: lrucache.LRUCache[common.Hash, uint32]{},
		Inclusions:         NewInclusions(),
	}

	update := overseer.ActiveLeavesUpdate{
		Activated: &initialHead,
	}
	updates, err := chainScraper.ProcessActiveLeavesUpdate(sender, update)
	if err != nil {
		return nil, nil, fmt.Errorf("processing active leaves update: %w", err)
	}

	return chainScraper, updates, nil
}
