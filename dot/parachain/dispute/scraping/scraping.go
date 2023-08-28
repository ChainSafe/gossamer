package scraping

import (
	"fmt"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	"github.com/ChainSafe/gossamer/internal/log"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
)

const (
	// DisputeCandidateLifetimeAfterFinalization How many blocks after finalisation an information about
	// backed/included candidate should be preloaded (when scraping onchain votes) and kept locally (when pruning).
	DisputeCandidateLifetimeAfterFinalization = uint32(10)

	// AncestryChunkSize Limits the number of ancestors received for a single request
	AncestryChunkSize = uint32(10)

	// AncestrySizeLimit Limits the overall number of ancestors walked through for a given head.
	AncestrySizeLimit = uint32(500) // TODO: This should be a MaxFinalityLag
)

var logger = log.NewFromGlobal(log.AddContext("disputes", "scraping"))

// ChainScrapper Scrapes non finalized chain in order to collect information from blocks
type ChainScrapper struct {
	// All candidates we have seen included, which not yet have been finalized.
	IncludedCandidates ScrappedCandidates
	// All candidates we have seen backed
	BackedCandidates ScrappedCandidates
	// Latest relay blocks observed by the provider.
	LastObservedBlocks lrucache.LRUCache[common.Hash, *uint32]
	// Maps included candidate hashes to one or more relay block heights and hashes.
	Inclusions Inclusions
	// Runtime instance
	Runtime parachain.RuntimeInstance
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
	if update.Activated == nil {
		return &types.ScrappedUpdates{}, nil
	}

	ancestors, err := cs.GetRelevantBlockAncestors(sender, update.Activated.Hash, update.Activated.Number)
	if err != nil {
		return nil, fmt.Errorf("getting relevant block ancestors: %w", err)
	}

	earliestBlockNumber := update.Activated.Number - uint32(len(ancestors))
	var blockNumbers []uint32
	for i := earliestBlockNumber; i <= update.Activated.Number; i++ {
		blockNumbers = append(blockNumbers, i)
	}
	blockHashes := append([]common.Hash{update.Activated.Hash}, ancestors...)

	var scrapedUpdates types.ScrappedUpdates
	for i, blockNumber := range blockNumbers {
		blockHash := blockHashes[i]

		receiptsForBlock, err := cs.ProcessCandidateEvents(sender, blockNumber, blockHash)
		if err != nil {
			return nil, fmt.Errorf("processing candidate events: %w", err)
		}
		scrapedUpdates.IncludedReceipts = append(scrapedUpdates.IncludedReceipts, receiptsForBlock...)

		onChainVotes, err := cs.Runtime.ParachainHostOnChainVotes(blockHash)
		if err != nil {
			return nil, fmt.Errorf("getting onchain votes: %w", err)
		}

		scrapedUpdates.OnChainVotes = append(scrapedUpdates.OnChainVotes, onChainVotes)
	}

	cs.LastObservedBlocks.Put(update.Activated.Hash, nil)
	return &scrapedUpdates, nil
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
	var (
		candidateEvents  []parachainTypes.CandidateEvent
		includedReceipts []parachainTypes.CandidateReceipt
	)

	events, err := cs.Runtime.ParachainHostCandidateEvents()
	if err != nil {
		return nil, fmt.Errorf("getting candidate events: %w", err)
	}

	for _, event := range events.Types {
		candidateEvents = append(candidateEvents, parachainTypes.CandidateEvent(event))
	}

	for _, candidateEvent := range candidateEvents {
		e, err := candidateEvent.Value()
		if err != nil {
			return nil, fmt.Errorf("getting candidate event value: %w", err)
		}
		switch event := e.(type) {
		case parachainTypes.CandidateIncluded:
			candidateHash, err := event.CandidateReceipt.Hash()
			if err != nil {
				return nil, fmt.Errorf("getting candidate receipt hash: %w", err)
			}

			cs.IncludedCandidates.Insert(blockNumber, candidateHash)
			cs.Inclusions.Insert(candidateHash, blockHash, blockNumber)
			includedReceipts = append(includedReceipts, event.CandidateReceipt)
		case parachainTypes.CandidateBacked:
			candidateHash, err := event.CandidateReceipt.Hash()
			if err != nil {
				return nil, fmt.Errorf("getting candidate receipt hash: %w", err)
			}

			cs.BackedCandidates.Insert(blockNumber, candidateHash)
		default:
			// skip the rest
		}
	}

	return includedReceipts, nil
}

func (cs *ChainScrapper) GetRelevantBlockAncestors(
	sender overseer.Sender,
	head common.Hash,
	headNumber uint32,
) ([]common.Hash, error) {
	targetAncestor, err := getFinalisedBlockNumber(sender)
	if err != nil {
		return nil, fmt.Errorf("getting finalised block number: %w", err)
	}
	targetAncestor = saturatingSub(targetAncestor, DisputeCandidateLifetimeAfterFinalization)
	var ancestors []common.Hash

	// If headNumber <= targetAncestor + 1 the ancestry will be empty.
	if observedBlock := cs.LastObservedBlocks.Get(head); observedBlock != nil || headNumber <= targetAncestor+1 {
		return ancestors, nil
	}

	for {
		hashes, err := getBlockAncestors(sender, head, AncestryChunkSize)
		if err != nil {
			return nil, fmt.Errorf("getting block ancestors: %w", err)
		}

		earliestBlockNumber := headNumber - uint32(len(hashes))
		if earliestBlockNumber < 0 {
			// It's assumed that it's impossible to retrieve more than N ancestors for block number N.
			logger.Errorf("received %v ancestors for block number %v", len(hashes), headNumber)
			return ancestors, nil
		}

		var blockNumbers []uint32
		for i := headNumber; i > earliestBlockNumber; i++ {
			blockNumbers = append(blockNumbers, i)
		}

		for i, blockNumber := range blockNumbers {
			// Return if we either met target/cached block or hit the size limit for the returned ancestry of head.
			if cs.LastObservedBlocks.Get(hashes[i]) != nil ||
				blockNumber <= targetAncestor ||
				len(ancestors) >= int(AncestrySizeLimit) {
				return ancestors, nil
			}

			ancestors = append(ancestors, hashes[i])
		}

		if len(hashes) < 0 {
			break
		}

		head = hashes[len(hashes)-1]
		headNumber = earliestBlockNumber
	}

	return ancestors, nil
}

func (cs *ChainScrapper) IsPotentialSpam(voteState types.CandidateVoteState, candidateHash common.Hash) (bool, error) {
	isDisputed := voteState.IsDisputed()
	isIncluded := cs.IsCandidateIncluded(candidateHash)
	isBacked := cs.IsCandidateBacked(candidateHash)
	isConfirmed, err := voteState.IsConfirmed()
	if err != nil {
		return false, fmt.Errorf("is confirmed: %w", err)
	}

	return isDisputed && !isIncluded && !isBacked && !isConfirmed, nil
}

func NewChainScraper(
	sender overseer.Sender,
	initialHead overseer.ActivatedLeaf,
) (*ChainScrapper, *types.ScrappedUpdates, error) {
	chainScraper := &ChainScrapper{
		IncludedCandidates: NewScrappedCandidates(),
		BackedCandidates:   NewScrappedCandidates(),
		LastObservedBlocks: lrucache.LRUCache[common.Hash, *uint32]{},
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

func getFinalisedBlockNumber(sender overseer.Sender) (uint32, error) {
	tx := make(chan overseer.FinalizedBlockNumberResponse)
	message := overseer.FinalizedBlockNumberRequest{
		ResponseChannel: tx,
	}
	err := sender.SendMessage(message)
	if err != nil {
		return 0, fmt.Errorf("sending message to get finalised block number: %w", err)
	}

	response := <-tx
	if response.Err != nil {
		return 0, fmt.Errorf("getting finalised block number: %w", response.Err)
	}

	return response.Number, nil
}

func getBlockAncestors(
	sender overseer.Sender,
	head common.Hash,
	numAncestors uint32,
) ([]common.Hash, error) {
	tx := make(chan overseer.AncestorsResponse)
	message := overseer.AncestorsRequest{
		Hash:            head,
		K:               numAncestors,
		ResponseChannel: tx,
	}
	err := sender.SendMessage(message)
	if err != nil {
		return nil, fmt.Errorf("sending message to get block ancestors: %w", err)
	}

	response := <-tx
	if response.Error != nil {
		return nil, fmt.Errorf("getting block ancestors: %w", response.Error)
	}

	return response.Ancestors, nil
}

func saturatingSub(a, b uint32) uint32 {
	diff := a - b
	if diff < 0 {
		return 0
	}
	return diff
}
