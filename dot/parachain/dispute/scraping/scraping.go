package scraping

import (
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	"github.com/ChainSafe/gossamer/lib/common"
	parachain "github.com/ChainSafe/gossamer/lib/parachain/types"
)

// DisputeCandidateLifetimeAfterFinalization How many blocks after finalization an information about
// backed/included candidate should be pre-loaded (when scraping onchain votes) and kept locally (when pruning).
const DisputeCandidateLifetimeAfterFinalization = uint32(10)

// ChainScrapper Scrapes non finalized chain in order to collect information from blocks
type ChainScrapper struct {
	// All candidates we have seen included, which not yet have been finalized.
	IncludedCandidates ScrappedCandidates
	// All candidates we have seen backed
	BackedCandidates ScrappedCandidates
	// Latest relay blocks observed by the provider.
	LastObservedBlocks LRUCache
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

func (cs *ChainScrapper) getBlocksIncludingCandidate(candidateHash common.Hash) []Inclusion {
	return cs.Inclusions.Get(candidateHash)
}

func (cs *ChainScrapper) ProcessActiveLeavesUpdate(sender overseer.Sender, update overseer.ActiveLeavesUpdate) (*types.ScrappedUpdates, error) {

	return nil, nil
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

func (cs *ChainScrapper) ProcessCandidateEvents(sender overseer.Sender, blockNumber uint32, blockHash common.Hash) ([]parachain.CandidateReceipt, error) {

	return nil, nil
}

func (cs *ChainScrapper) GetRelevantBlockAncestors(sender overseer.Sender, head common.Hash, headNumber uint32) ([]common.Hash, error) {

	return nil, nil
}
